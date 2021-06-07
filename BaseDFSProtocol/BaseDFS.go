// Base DFS protocol:

// Asumptions:
// 1) market matching is done
// (for each client we have one server who has stored his client's file
// and the storage contract is fixed for all - fixed payment)
// 2) clients have created their escrows -
// a fixed amount of money is in some addresses (clients addresses)

// The BaseDFSProtocol goes as follows:
// 1- a server broadcast a por tx
// (chanllenge's randomness comes from hash of the latest mined block)
// 2) all miners (who recieve this tx) verify the por tx + add a fixed payment tx +
// 		keep both in their current block
// 3) (each epoch) miners check the predefined leader election mechanism
//		 to see if they are the leader
// 4) the leader broadcast his block, consisting por and payment tx.s
//		and his election proof
// 5) all miners (who recieve this tx) verify proposed block and
//		add it to their chain

//Types of Messages:
// 1- por
// 2- proposed block

package BaseDFSProtocol

import (
	"math"
	"sync"
	"time"

	onet "github.com/basedfs"
	"github.com/basedfs/blockchain"
	"github.com/basedfs/blockchain/blkparser"
	"github.com/basedfs/log"
	"github.com/basedfs/network"
	"github.com/basedfs/simul/monitor"
	//same as sda in byzcoin - replaced all sda objects with onet/network
)

func init() {

	network.RegisterMessage(ProofOfRetTxChan{})
	network.RegisterMessage(PreparedBlockChan{})
	onet.GlobalProtocolRegister("BaseDFS", NewBaseDFSProtocol)
}

type ProofOfRetTxChan struct {
	*onet.TreeNode
	por
}
type por struct {
	por blkparser.Tx
}

type PreparedBlockChan struct {
	*onet.TreeNode
	preparedBlock
}
type preparedBlock struct {
	prb blkparser.Block
}

// baseDFS is the main struct for running the protocol
type BaseDFS struct {
	// the node we are represented-in
	*onet.TreeNodeInstance
	// the suite we use
	suite network.Suite
	// channel for por from servers to miners
	PreparedBlockChan chan PreparedBlockChan
	// channel for por from leader to miners
	ProofOfRetTxChan chan ProofOfRetTxChan
	// channel to notify when we are done ?
	done chan bool //it is not initiated in new proto
	// channel used to wait for the verification of the block
	verifyBlockChan chan bool
	// block to be prepared for next epoch
	tempBlock *blockchain.TrBlock
	// transactions is the slice of transactions that contains transactions
	// coming from servers (who convert the por into transaction?)
	transactions []blkparser.Tx
	// last block computed
	lastBlock string
	// last key block computed
	lastKeyBlock string
	// temporary buffer of por transactions (how miners are store por transactions? )
	tempCommitResponse []blkparser.Tx
	tcrMut             sync.Mutex
	// refusal to append por-tx to current block
	porTxRefusal bool

	// onDoneCallback is the callback that will be called at the end of the
	// protocol when all nodes have finished.
	// Either after refusing or accepting the proposed block
	// or at the end of a view change.
	onDoneCallback func()
	// onAppendPoRTxDone is the callback that will be called when a por tx has been verified and appended to our cuurent temp block we may need it later!
	onAppendPoRTxDone func(*por)
	// rootTimeout is the timeout given to the root. It will be passed down the
	// tree so every nodes knows how much time to wait. This root is a very nice
	// malicious node.
	rootTimeout uint64
	timeoutChan chan uint64
	// onTimeoutCallback is the function that will be called if a timeout
	// occurs.
	onTimeoutCallback func()

	// function to let callers of the protocol (or the server) add functionality
	// to certain parts of the protocol; mainly used in simulation to do
	// measurements. Hence functions will not be called in go routines

	// root fails:
	rootFailMode uint
	// Call back when we start broadcasting our por tx
	onBroadcastPoRTx func()
	// Call back when we start checking this epoch's leadership
	onCheckEpochLeadership func()
	// callback when we finished
	// "verifying por tx"
	// + act = "adding payment tx + appending both to our current block"
	onVerifyActPoRTxDone func()
	// callback when we finished
	// "verifying new block"
	// + act = "appending it to our current blockchain"
	onVerifyActEpochBlockDone func()
	// view change setup and measurement
	viewchangeChan chan struct {
		*onet.TreeNode
		viewChange
	}
	vcMeasure *monitor.TimeMeasure
	// lock associated
	doneLock sync.Mutex
	// threshold for how much view change acceptance we need
	// basically n - threshold
	viewChangeThreshold int
	// how many view change request have we received
	vcCounter int
	// done processing is used to stop the processing of the channels
	doneProcessing chan bool
	// finale block that this BaseDFS epoch has produced
	finalBlock *blockchain.TrBlock
}

// NewBaseDFSProtocol returns a new BaseDFS struct
//func NewBaseDFSProtocol(n *onet.TreeNodeInstance) (*BaseDFS, error) {
func NewBaseDFSProtocol(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
	// create the byzcoin
	bz := &BaseDFS{
		TreeNodeInstance: n,
		suite:            n.Suite(),
		verifyBlockChan:  make(chan bool),
		doneProcessing:   make(chan bool, 2),
		timeoutChan:      make(chan uint64, 1),
	}
	bz.viewChangeThreshold = int(math.Ceil(float64(len(bz.Tree().List())) * 2.0 / 3.0))

	// register channels
	if err := n.RegisterChannel(&bz.PreparedBlockChan); err != nil {
		return bz, err
	}
	if err := n.RegisterChannel(&bz.ProofOfRetTxChan); err != nil {
		return bz, err
	}

	n.OnDoneCallback(bz.nodeDone) //?

	go bz.listen()
	return bz, nil
}

// NewbaseDFSRootProtocol returns a new baseDFS struct with the block to sign that will be sent to all others nodes - when it has called???
func NewbaseDFSRootProtocol(n *onet.TreeNodeInstance, transactions []blkparser.Tx, timeOutMs uint64, failMode uint) (onet.ProtocolInstance, error) {
	bz, err := NewBaseDFSProtocol(n)
	if err != nil {
		return nil, err
	}
	//raha - fix later
	//bz.transactions = transactions
	//bz.rootFailMode = failMode
	//bz.rootTimeout = timeOutMs
	return bz, err
}

// The simple version:
// we assume one server broadcast his por tx
// - which is the root in tree structure -
// and all servers on recieving this tx will verify
// and add it to their prepared block
// and then all server run leader election to check if
// they are leader, then the leader add his proof
// and broadcast his prepared block
// and all servers verify and append his block to their blockchain

func (bz *BaseDFS) Start() error {
	txs, _ := blkparser.NewTx([]byte{'x'})
	portx := &ProofOfRetTxChan{
		TreeNode: bz.TreeNode(),
		por: por{
			por: *txs,
		},
	}
	if err := bz.Broadcast(portx); err != nil {
		//raha fix later : return err
		return nil
	}
	log.Lvl3(bz.Name(), "finished broadcasting his por")
	return nil
}

// Dispatch listen on the different channels
func (bz *BaseDFS) Dispatch() error {
	return nil
}
func (bz *BaseDFS) listen() {
	// FIXME handle different failure modes
	fail := (bz.rootFailMode != 0) && bz.IsRoot()
	var timeoutStarted bool
	for {
		var err error
		select {
		case msg := <-bz.ProofOfRetTxChan:
			// PoR
			if !fail {
				err = bz.handlePoRTx(msg.por)
			}
		case msg := <-bz.PreparedBlockChan:
			// Next Block
			if !fail {
				err = bz.handleBlock(msg.preparedBlock)
			}
			//-------------------------------------------------------------
		case timeout := <-bz.timeoutChan:
			// start the timer
			if timeoutStarted {
				continue
			}
			timeoutStarted = true
			go bz.startTimer(timeout)
		case msg := <-bz.viewchangeChan:
			// receive view change
			err = bz.handleViewChange(msg.TreeNode, &msg.viewChange)
		case <-bz.doneProcessing:
			// we are done
			log.Lvl2(bz.Name(), "BaseDFS Dispatches stop.")
			bz.tempBlock = nil
			return
		}
		if err != nil {
			log.Error(bz.Name(), "Error handling messages:", err)
		}
	}
}

// handleAnnouncement pass the announcement to the right CoSi struct.
func (bz *BaseDFS) handlePoRTx(proofOfRet por) error {
	if err := verifyPoRTx(proofOfRet); err == nil {
		e := bz.appendPoRTx(proofOfRet)
		if e == nil {
			log.Lvl1(bz.TreeNode, "PoR tx appended to current temp block")
		} else {
			log.Lvl1(bz.TreeNode, "PoR tx appending error:", e)
		}
	} else {
		log.Lvl1(bz.TreeNode, "verifying PoR tx error:", err)
	}
	return nil
}

// verifyPoRTx: servers will verify por taxs when they recieve it
func verifyPoRTx(p por) error {
	return nil
}

//appenPoRTx:
func (bz *BaseDFS) appendPoRTx(p por) error {
	return nil
}

// handle the arrival of a commitment
func (bz *BaseDFS) handleBlock(pb preparedBlock) error {
	if err := verifyBlock(pb); err == nil {
		e := bz.appendBlock(pb)
		if e == nil {
			log.Lvl1(bz.TreeNode, "proposed block appended to blockchain")
		} else {
			log.Lvl1(bz.TreeNode, "proposed block appending error:", e)
		}
	} else {
		log.Lvl1(bz.TreeNode, "proposed block verifying error:", err)
	}
	return nil
}

// verifyPoRTx: servers will verify por taxs when they recieve it
func verifyBlock(pb preparedBlock) error {
	return nil
}

//appenPoRTx:
func (bz *BaseDFS) appendBlock(pb preparedBlock) error {
	return nil
}

//---------------------------------------------------------------------------
// startTimer starts the timer to decide whether we should request a view change after a certain timeout or not.
// If the signature is done, we don't. otherwise
// we start the view change protocol.
func (bz *BaseDFS) startTimer(millis uint64) {
	if bz.rootFailMode != 0 {
		log.Lvl3(bz.Name(), "Started timer (", millis, ")...")
		select {
		case <-bz.done:
			return
		case <-time.After(time.Millisecond * time.Duration(millis)):
			bz.sendAndMeasureViewchange()
		}
	}
}

// sendAndMeasureViewChange is a method that creates the viewchange request,
// broadcast it and measures the time it takes to accept it.
func (bz *BaseDFS) sendAndMeasureViewchange() {
	log.Lvl3(bz.Name(), "Created viewchange measure")
	bz.vcMeasure = monitor.NewTimeMeasure("viewchange")
	vc := newViewChange()
	var err error
	for _, n := range bz.Tree().List() {
		// don't send to ourself
		if n.ID.Equal(bz.TreeNode().ID) {
			continue
		}
		err = bz.SendTo(n, vc)
		if err != nil {
			log.Error(bz.Name(), "Error sending view change", err)
		}
	}
}

// viewChange is simply the last hash / id of the previous leader.
// type viewChange struct {
// 	LastBlock [sha256.Size]byte
// }

// newViewChange creates a new view change.
// func newViewChange() *viewChange {
// 	res := &viewChange{}
// 	for i := 0; i < sha256.Size; i++ {
// 		res.LastBlock[i] = 0
// 	}
// 	return res
// }

// handleViewChange receives a view change request and if received more than
// 2/3, accept the view change.
func (bz *BaseDFS) handleViewChange(tn *onet.TreeNode, vc *viewChange) error {
	bz.vcCounter++
	// only do it once
	if bz.vcCounter == bz.viewChangeThreshold {
		if bz.vcMeasure != nil {
			bz.vcMeasure.Record()
		}
		if bz.IsRoot() {
			log.Lvl3(bz.Name(), "Viewchange threshold reached (2/3) of all nodes")
			go bz.Done()
			//	bz.endProto.Start()
		}
		return nil
	}
	return nil
}

// nodeDone is either called by the end of EndProtocol or by the end of the
// response phase of the commit round.
func (bz *BaseDFS) nodeDone() bool {
	log.Lvl3(bz.Name(), "nodeDone()      ----- ")
	bz.doneProcessing <- true
	log.Lvl3(bz.Name(), "nodeDone()      +++++  ", bz.onDoneCallback)
	if bz.onDoneCallback != nil {
		bz.onDoneCallback()
	}
	return true
}
