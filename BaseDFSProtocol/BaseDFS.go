/* Base DFS protocol: The simple version:
we assume one server broadcast his por tx and all servers on recieving this tx will verify and add it to their prepared block and then all server run leader election to check if they are leader, then the leader add his proof and broadcast his prepared block and all servers verify and append his block to their blockchain

The BaseDFSProtocol goes as follows:
1- a server broadcast a por tx
(chanllenge's randomness comes from hash of the latest mined block)
2) all miners (who recieve this tx) verify the por tx + add a fixed payment tx +
		keep both in their current block
3) (each epoch) miners check the predefined leader election mechanism
		 to see if they are the leader
4) the leader broadcast his block, consisting por and payment tx.s
		and his election proof
5) all miners (who recieve this tx) verify proposed block and
		add it to their chain

Types of Messages:
1- por
2- proposed block
*/

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
	network.RegisterMessage(HelloBaseDFS{})
	onet.GlobalProtocolRegister("BaseDFS", NewBaseDFSProtocol)
}

// Hello is sent down the tree from the root node, every node who gets it send it to its children and start multicasting his por tx.s
type HelloBaseDFS struct {
	Timeout time.Duration
}

type HelloChan struct {
	*onet.TreeNode
	HelloBaseDFS
}

type ProofOfRetTxChan struct {
	*onet.TreeNode
	Por
}

type Por struct {
	Tx blkparser.Tx
	N  uint32
}

type PreparedBlockChan struct {
	*onet.TreeNode
	PreparedBlock
}

type PreparedBlock struct {
	blkparser.Block
}

type leadershipProof struct {
	U uint32
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
	DoneBaseDFS chan bool //it is not initiated in new proto
	// channel used to let all servers that the protocol has started
	HelloChan chan HelloChan
	// channel used to wait for the verification of the block
	//verifyBlockChan chan bool//
	// block to be proposed by the leader - if 2/3 miners signal back it submitted it will be the final block
	tempBlock *blockchain.TrBlock
	// transactions is the slice of transactions that contains transactions
	// coming from servers (who convert the por into transaction?)
	transactions []blkparser.Tx
	// last block computed
	lastBlock string //?
	// last key block computed
	lastKeyBlock string //?
	// temporary buffer of ?
	//tempCommitResponse []blkparser.Tx//
	//tcrMut             sync.Mutex//
	// refusal to append por-tx to current block
	porTxRefusal bool

	// onDoneCallback is the callback that will be called at the end of the
	// protocol when all nodes have finished.
	// Either after refusing or accepting the proposed block
	// or at the end of a view change.
	// raha: when this function is called?
	onDoneCallback func()

	// onAppendPoRTxDone is the callback that will be called when a por tx has been verified and appended to our cuurent temp block. raha: we may need it later!
	//onAppendPoRTxDone func(*por)//

	// rootTimeout is the timeout given to the root. It will be passed down the
	// tree so every nodes knows how much time to wait. This root is a very nice
	// malicious node.
	rootTimeout uint64
	timeoutChan chan uint64
	// onTimeoutCallback is the function that will be called if a timeout
	// occurs.
	//onTimeoutCallback func()//

	// function to let callers of the protocol (or the server) add functionality
	// to certain parts of the protocol; mainly used in simulation to do
	// measurements. Hence functions will not be called in go routines

	// root fails:
	rootFailMode uint
	// Call back when we start broadcasting our por tx
	//onBroadcastPoRTx func()//
	// Call back when we start checking this epoch's leadership
	//onCheckEpochLeadership func()//
	// callback when we finished "verifying por tx" + act = "adding payment tx + appending both to our current block"
	//onVerifyActPoRTxDone func()//
	// callback when we finished "verifying new block" + act = "appending it to our current blockchain"
	//onVerifyActEpochBlockDone func()//
	// view change setup and measurement
	viewchangeChan chan struct {
		*onet.TreeNode
		viewChange
	}
	//raha: what does it do?
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
	//from opinion gathering protocol
	timeout   time.Duration
	timeoutMu sync.Mutex
	epochNo   uint32
}

// NewBaseDFSProtocol returns a new BaseDFS struct
//func NewBaseDFSProtocol(n *onet.TreeNodeInstance) (*BaseDFS, error) {
func NewBaseDFSProtocol(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
	// create the byzcoin
	bz := &BaseDFS{
		TreeNodeInstance: n,
		suite:            n.Suite(),
		DoneBaseDFS:      make(chan bool, 1),
		//verifyBlockChan:  			make(chan bool),
		doneProcessing: make(chan bool, 2),
		timeoutChan:    make(chan uint64, 1),
	}
	bz.viewChangeThreshold = int(math.Ceil(float64(len(bz.Tree().List())) * 2.0 / 3.0))

	// register channels
	if err := n.RegisterChannel(&bz.PreparedBlockChan); err != nil {
		return bz, err
	}
	if err := n.RegisterChannel(&bz.ProofOfRetTxChan); err != nil {
		return bz, err
	}
	if err := n.RegisterChannel(&bz.viewchangeChan); err != nil {
		return bz, err
	}
	if err := n.RegisterChannel(&bz.HelloChan); err != nil {
		return bz, err
	}
	// --------------------------  this section is in opinion gathering but byzcoin has the above code instead! check later; which is enough and required
	// t := n.Tree()
	// if t == nil {
	// 	return nil, nil //raha: fix later: raise an error
	// }
	// if err := bz.RegisterChannelsLength(len(t.List()),
	// 	&bz.HelloChan, &bz.PreparedBlockChan, bz.ProofOfRetTxChan, &bz.viewchangeChan); err != nil {
	// 	log.Error("Couldn't reister channel:", err)
	// }

	bz.transactions = nil // raha: transactions was an in param in NewbaseDFSRootProtocol
	bz.rootFailMode = 0   // raha: failMode was an in param in NewbaseDFSRootProtocol
	bz.rootTimeout = 300  // raha: timeOutMs was an in param in NewbaseDFSRootProtocol
	//epochNo = 1

	n.OnDoneCallback(bz.nodeDone) // raha: when this function is called?
	return bz, nil
}

//Start: starts the simplified protocol by sending hello msg to all nodes which later makes them to start sending their por tx.s
func (bz *BaseDFS) Start() error {
	//bz.sendPoRTx()
	bz.helloBaseDFS()
	//var leaderNo  *network.ServerIdentity = bz.checkLeadership()
	log.Lvl2(bz.Info(), "Started the protocol by running Start function")
	return nil
}

// Dispatch listen on the different channels
func (bz *BaseDFS) Dispatch() error {
	fail := (bz.rootFailMode != 0) && bz.IsRoot()
	var timeoutStarted bool
	running := true
	var err error
	for running {
		//log.Lvl2(bz.Info(), "waiting for message for", bz.Timeout())
		select {
		case msg := <-bz.HelloChan:
			log.Lvl2(bz.Info(), "received Hello from", msg.TreeNode.ServerIdentity.Address)
			bz.helloBaseDFS()
		case msg := <-bz.ProofOfRetTxChan:
			log.Lvl2(bz.Info(), "received por", msg.Por.N, "tx from", msg.TreeNode.ServerIdentity.Address)
			if !fail {
				err = bz.handlePoRTx(msg)
			}
		case msg := <-bz.PreparedBlockChan:
			log.Lvl2(bz.Info(), "received block from", msg.TreeNode.ServerIdentity.Address)
			if !fail {
				_, err = bz.handleBlock(msg)
			}
			//-------------------------------------
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
		case <-bz.DoneBaseDFS:
			// what now?
			// log.Lvl2(bz.Name(), "doneBaseDFS just been called")
			// running = false
		}
		if err != nil {
			log.Error(bz.Name(), "Error handling messages:", err)
		}
	}
	return err
}

//helloBaseDFS
func (bz *BaseDFS) helloBaseDFS() {
	if !bz.IsLeaf() {
		for _, child := range bz.Children() {
			go func(c *onet.TreeNode) {
				log.Lvl2(bz.Info(), "sending hello to", c.ServerIdentity.Address, c.ID, "timeout", bz.timeout)
				err := bz.SendTo(c, &HelloBaseDFS{Timeout: bz.timeout})
				if err != nil {
					log.Lvl2(bz.Info(), "couldn't send hello to child",
						c.Name())
				}
			}(child)
		}
		bz.sendPoRTx()
	} else {
		bz.sendPoRTx()
	}
}

//createPoRTx: later we want peridic broadcasting of por txs by random servers in the roster.
func (bz *BaseDFS) createPoRTx() *blkparser.Tx {
	// x := []byte("010000000101820e2169131a77976cf204ce28685e49a6d2278861c33b6241ba3ae3e0a49f020000008b48304502210098a2851420e4daba656fd79cb60cb565bd7218b6b117fda9a512ffbf17f8f178022005c61f31fef3ce3f906eb672e05b65f506045a65a80431b5eaf28e0999266993014104f0f86fa57c424deb160d0fc7693f13fce5ed6542c29483c51953e4fa87ebf247487ed79b1ddcf3de66b182217fcaf3fcef3fcb44737eb93b1fcb8927ebecea26ffffffff02805cd705000000001976a91429d6a3540acfa0a950bef2bfdc75cd51c24390fd88ac80841e00000000001976a91417b5038a413f5c5ee288caa64cfab35a0c01914e88ac00000000")
	// // tx, _ := blkparser.ParseTxs(x)
	// // tx1 := tx[len(tx)-1]
	// txIns := []*blkparser.TxIn{{
	// 	InputHash: "b6f6991d03df0e2e04dafffcd6bc418aac66049e2cd74b80f14ac86db1e3f0da",
	// 	InputVout: 5,
	// 	ScriptSig: x,
	// 	Sequence:  12,
	// }, {
	// 	InputHash: "b6f6991d03df0e2e04dafffcd6bc418aac66049e2cd74b80f14ac86db1e3f0da",
	// 	InputVout: 5,
	// 	ScriptSig: x,
	// 	Sequence:  12,
	// },
	// }
	// tx1 := &blkparser.Tx{
	// 	Hash:     "b6f6991d03df0e2e04dafffcd6bc418aac66049e2cd74b80f14ac86db1e3f0da",
	// 	Size:     2,
	// 	LockTime: 0,
	// 	Version:  1,
	// 	TxInCnt:  1,
	// 	TxOutCnt: 1,
	// 	TxIns:    txIns,
	// 	TxOuts:   nil,
	// }
	x := []byte("fds")
	// tx, _ := blkparser.ParseTxs(x)
	// tx1 := tx[len(tx)-1]
	txIns := []*blkparser.TxIn{{
		InputHash: "ff",
		InputVout: 5,
		ScriptSig: x,
		Sequence:  12,
	}, {
		InputHash: "sd",
		InputVout: 5,
		ScriptSig: x,
		Sequence:  12,
	},
	}
	tx1 := &blkparser.Tx{
		Hash:     "s",
		Size:     2,
		LockTime: 0,
		Version:  1,
		TxInCnt:  1,
		TxOutCnt: 1,
		TxIns:    txIns,
		TxOuts:   nil,
	}
	return tx1
}

//sendPoRTx send a por tx
func (bz *BaseDFS) sendPoRTx() {
	txs := bz.createPoRTx()
	portx := &Por{*txs, uint32(bz.Index())}
	log.Lvl2(bz.Name(), ": multicasting por tx")
	bz.Multicast(portx, bz.List()...)

	// for _, child := range bz.Children() {
	// 	err := bz.SendTo(child, portx)
	// 	if err != nil {
	// 		log.Lvl1(bz.Info(), "couldn't send to child:", child.ServerIdentity.Address)
	// 	} else {
	// 		log.Lvl1(bz.Info(), "sent his PoR to his children:", child.ServerIdentity.Address)
	// 	}
	// }
}

// handleAnnouncement pass the announcement to the right CoSi struct.
func (bz *BaseDFS) handlePoRTx(proofOfRet ProofOfRetTxChan) error {
	if refuse, err := verifyPoRTx(proofOfRet); err == nil {
		if refuse == true {
			bz.porTxRefusal = true
			return nil
		} else {
			e := bz.appendPoRTx(proofOfRet)
			if e == nil {
				log.Lvl2(bz.Info(), "PoR tx appended to current temp block")
				//bz.DoneBaseDFS <- true
			} else {
				log.Lvl2(bz.TreeNode, "PoR tx appending error:", e)
			}
		}
	} else {
		log.Lvl2(bz.TreeNode, "verifying PoR tx error:", err)
	}
	return nil
}

// verifyPoRTx: servers will verify por tx.s when they recieve it
func verifyPoRTx(p ProofOfRetTxChan) (bool, error) {
	//test if new repo in chainBst is connected to local Git
	var refuse = false
	var err error = nil
	return refuse, err
}

//appenPoRTx: append the recieved tx to his current temporary block
func (bz *BaseDFS) appendPoRTx(p ProofOfRetTxChan) error {
	bz.transactions = append(bz.transactions, p.Por.Tx)
	// for test creating block:
	//bz.createEpochBlock(&leadershipProof{1})
	return nil
}

//checkLeadership
func (bz *BaseDFS) checkLeadership() *network.ServerIdentity {
	n := bz.Roster().RandomServerIdentity()
	return n
}

//createEpochBlock: by leader
func (bz *BaseDFS) createEpochBlock(lp *leadershipProof) *blockchain.
	TrBlock {
	//if bz.ServerIdentity().Equal(){
	var h blockchain.TransactionList = blockchain.NewTransactionList(bz.transactions, len(bz.transactions))
	bz.tempBlock = blockchain.NewTrBlock(h, blockchain.NewHeader(h, "raha", "raha"))
	return bz.tempBlock
	//}else{
	//	return nil
	//}
}

//SendFinalBlock   bz.SendFinalBlock(createEpochBlock)
func (bz *BaseDFS) SendFinalBlock(fb *blockchain.TrBlock) {
	log.Lvl1("final block hash is:", fb.Block.HeaderHash)
	return
}

// handle the arrival of a block
func (bz *BaseDFS) handleBlock(pb PreparedBlockChan) (*blockchain.TrBlock, error) {
	if err := verifyBlock(pb.PreparedBlock); err == nil {
		return (bz.appendBlock(pb.PreparedBlock))
	} else {
		log.Error(bz.Name(), "Error verying block", err)
		return nil, nil
	}
}

// verifyBlock: servers will verify proposed block when they recieve it
func verifyBlock(pb PreparedBlock) error {
	return nil
}

//appendBlock:
func (bz *BaseDFS) appendBlock(pb PreparedBlock) (*blockchain.TrBlock, error) {
	return nil, nil
}

//-------------------------------------------------------------------
// startTimer starts the timer to decide whether we should request a view change after a certain timeout or not.
// If the signature is done, we don't. otherwise
// we start the view change protocol.
func (bz *BaseDFS) startTimer(millis uint64) {
	if bz.rootFailMode != 0 {
		log.Lvl3(bz.Name(), "Started timer (", millis, ")...")
		select {
		case <-bz.DoneBaseDFS:
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

// handleViewChange receives a view change request and if received more than 2/3, accept the view change.
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
	//raha
	bz.DoneBaseDFS <- true
	return true
}

// -------------------
// from OpinionGathering Protocol
// -------------------
// SetTimeout sets the new timeout
// SetTimeout sets the new timeout
func (bz *BaseDFS) SetTimeout(t time.Duration) {
	bz.timeoutMu.Lock()
	bz.timeout = t
	bz.timeoutMu.Unlock()
}

// Timeout returns the current timeout
func (bz *BaseDFS) Timeout() time.Duration {
	bz.timeoutMu.Lock()
	defer bz.timeoutMu.Unlock()
	return bz.timeout
}
