/*
Base DFS protocol: The simple version:

we assume one server broadcast his por tx and all servers on recieving this tx will verify
and add it to their prepared block and then all server run leader election
to check if they are leader, then the leader add his proof and broadcast
his prepared block and all servers verify and append his block to their blockchain
---------------------------------------------
The BaseDFSProtocol goes as follows: it should change - its better now:)
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
--------------------------------------------
Types of Messages:
1- por
2- proposed block
3- Hello (to notify start of protocol)
4- epoch timeOut
*/

package BaseDFSProtocol

import (
	"encoding/binary"
	"io/ioutil"

	//"strings"

	onet "github.com/basedfs"
	"github.com/basedfs/blockchain"
	"github.com/basedfs/blockchain/blkparser"
	"github.com/basedfs/log"
	"github.com/basedfs/network"
	"github.com/basedfs/por"
	"github.com/basedfs/simul/monitor"
	crypto "github.com/basedfs/vrf"

	// "gonum.org/v1/gonum/stat/distuv" bionomial distribution in algorand leader election
	"math"
	"math/rand"
	"sync"
	"time"
)

func init() {

	network.RegisterMessage(ProofOfRetTxChan{})
	network.RegisterMessage(PreparedBlockChan{})
	network.RegisterMessage(HelloBaseDFS{})
	onet.GlobalProtocolRegister("BaseDFS", NewBaseDFSProtocol)
}

//-----------------------------------------------------------------------
// Hello is sent down the tree from the root node,
// every node who gets it send it to its children
// and start multicasting his por tx.s
type HelloBaseDFS struct {
	Timeout time.Duration
}

type HelloChan struct {
	*onet.TreeNode
	HelloBaseDFS
}

type ProofOfRetTxChan struct {
	*onet.TreeNode
	por.Por
}

type PreparedBlockChan struct {
	*onet.TreeNode
	PreparedBlock
}

type PreparedBlock struct {
	blockchain.Block
}

type leadershipProof struct {
	proof crypto.VrfProof
}
type Seed [32]byte

type tx struct {
	i int8
}

// baseDFS is the main struct for running the protocol
type BaseDFS struct {
	// the node we are represented-in
	*onet.TreeNodeInstance
	ECPrivateKey crypto.VrfPrivkey
	// channel for por from servers to miners
	PreparedBlockChan chan PreparedBlockChan
	// channel for por from leader to miners
	ProofOfRetTxChan chan ProofOfRetTxChan
	// channel to notify when we are done ?
	DoneBaseDFS chan bool //it is not initiated in new proto
	// channel used to let all servers that the protocol has started
	HelloChan chan HelloChan
	// block to be proposed by the leader - if 2/3 miners signal back it submitted it will be the final block
	tempBlock *blockchain.TrBlock
	// transactions is the slice of transactions that contains transactions
	// coming from servers (who convert the por into transaction?)
	transactions  []blkparser.Tx
	epochDuration time.Duration
	PoRTxDuration time.Duration
	// finale block that this BaseDFS epoch has produced
	finalBlock       *blockchain.TrBlock
	currentWeight    map[*network.ServerIdentity]int
	totalCurrency    int
	initialSeed      [32]byte
	currentRoundSeed [32]byte
	roundNumber      int
	blockChainSize   uint64
	/* -----------------------------------------------------------
	These fields are borrowed from ByzCoin
	and may be useful for future functionalities
	------------------------------------------------------------ */

	// the suite we use
	suite network.Suite //?
	// last block computed
	lastBlock string //?
	// last key block computed
	lastKeyBlock string //?
	// refusal to append por-tx to current block
	porTxRefusal bool //?
	// temporary buffer of ?
	//tempCommitResponse []blkparser.Tx//
	//tcrMut             sync.Mutex//
	// channel used to wait for the verification of the block
	//verifyBlockChan chan bool//

	// onDoneCallback is the callback that will be called at the end of the
	// protocol when all nodes have finished.
	// Either after refusing or accepting the proposed block
	// or at the end of a view change.
	// raha: when this function is called?
	onDoneCallback func() //?

	// onAppendPoRTxDone is the callback that will be called when a por tx has been verified and appended to our cuurent temp block. raha: we may need it later!
	//onAppendPoRTxDone func(*por) //?

	// rootTimeout is the timeout given to the root. It will be passed down the
	// tree so every nodes knows how much time to wait. This root is a very nice
	// malicious node.
	rootTimeout uint64      //?
	timeoutChan chan uint64 //?
	// onTimeoutCallback is the function that will be called if a timeout
	// occurs.
	//onTimeoutCallback func() //?

	// function to let callers of the protocol (or the server) add functionality
	// to certain parts of the protocol; mainly used in simulation to do
	// measurements. Hence functions will not be called in go routines

	// root fails:
	rootFailMode uint //?
	// Call back when we start broadcasting our por tx
	//onBroadcastPoRTx func()//
	// Call back when we start checking this epoch's leadership
	onCheckEpochLeadership func()
	// callback when we finished "verifying por tx" + act = "adding payment tx + appending both to our current block"
	//onVerifyActPoRTxDone func()//
	// callback when we finished "verifying new block" + act = "appending it to our current blockchain"
	//onVerifyActEpochBlockDone func()//
	// view change setup and measurement
	viewchangeChan chan struct { //?
		*onet.TreeNode
		viewChange
	}
	//raha: what does it do?
	vcMeasure *monitor.TimeMeasure //?
	// lock associated
	doneLock sync.Mutex //?
	// threshold for how much view change acceptance we need
	// basically n - threshold
	viewChangeThreshold int //?
	// how many view change request have we received
	vcCounter int //?
	// done processing is used to stop the processing of the channels
	doneProcessing chan bool //?
	//from opinion gathering protocol
	timeout   time.Duration //?
	timeoutMu sync.Mutex    //?
}

// NewBaseDFSProtocol returns a new BaseDFS struct
//func NewBaseDFSProtocol(n *onet.TreeNodeInstance) (*BaseDFS, error) {
func NewBaseDFSProtocol(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
	bz := &BaseDFS{
		TreeNodeInstance: n,
		suite:            n.Suite(),
		DoneBaseDFS:      make(chan bool, 1),
		//verifyBlockChan:  			make(chan bool),
		doneProcessing: make(chan bool, 2),
		timeoutChan:    make(chan uint64, 1),
		currentWeight:  make(map[*network.ServerIdentity]int, 100),
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
	/* ----------------------------------------------------
	 this section is in opinion gathering but byzcoin has
	 the above code instead! check later; which is enough and required
	-------------------------------------------------------*/
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
	// epoch duration on which after this time the leader get choosed and propose a new block (unit?!)
	bz.epochDuration = 30 // similar to Filecoin round durartion
	bz.PoRTxDuration = 5
	bz.totalCurrency = 0
	bz.roundNumber = 0
	// bls key pair for each node as its needed for VRF
	//ToDo: raha: check how nodes key pair are handeled in network layer
	_, bz.ECPrivateKey = crypto.VrfKeygen()
	//currentRoundSeed: crypto.NextRoundSeed(), // it should be called by the leader in previous round
	// test
	//t := [32] byte {1,2,3,4,5,1,2,3,4,5,1,2,3,4,5,1,2,3,4,5,1,2,3,4,5,1,2,3,4,5,1,2}
	//bz.currentRoundSeed = t
	// iniitial weight of all nodes
	//rand.Seed(int64(bz.currentRoundSeed))
	//rand.Seed(int64(blockchainRandomSeed))
	//rand.Seed(86)
	rng := rand.New(rand.NewSource(0))
	for _, si := range n.Roster().List {
		// ToDO : raha: they should have a similar view of the initial stake in the blockchain. it gives us different arrays for each node
		bz.currentWeight[si] = rng.Intn(100) // 100: a fixed number for the maximum stake of each node
		bz.totalCurrency = bz.totalCurrency + bz.currentWeight[si]
	}
	n.OnDoneCallback(bz.nodeDone) // raha: when this function is called?
	return bz, nil
}

//Start: starts the simplified protocol by sending hello msg to all nodes
// which later makes them start sending their por tx.s
func (bz *BaseDFS) Start() error {
	//por.Testpor()
	//crypto.Testvrf()
	bz.helloBaseDFS()
	log.Lvl2(bz.Info(), "Started the protocol by running Start function")
	csv, err := ioutil.ReadFile("centralbc.csv")
	log.ErrFatal(err)
	log.LLvl2("the csv is like: ", string(csv))
	//assert.Equal(t, 7, len(strings.Split(string(csv), "\n")))
	return nil
}

// Dispatch listen on the different channels
func (bz *BaseDFS) Dispatch() error {
	fail := (bz.rootFailMode != 0) && bz.IsRoot()
	var timeoutStarted bool
	running := true
	var err error
	for running {
		select {

		case msg := <-bz.HelloChan:
			log.Lvl2(bz.Info(), "received Hello from", msg.TreeNode.ServerIdentity.Address)
			log.Lvl2(bz.Info(), "check if he is the leader")
			bz.helloBaseDFS()

		case msg := <-bz.ProofOfRetTxChan:
			//log.Lvl2(bz.Info(), "received por", msg.Por.sigma, "tx from", msg.TreeNode.ServerIdentity.Address)
			if !fail {
				err = bz.handlePoRTx(msg)
			}

		case <-time.After(time.Second * bz.epochDuration):
			// bz.SendFinalBlock(bz.createEpochBlock(bz.checkLeadership()))
			// next round seed
			log.LLvl2("next round:", bz.roundNumber, "seen by", bz.Info())
			// call a function that next round starts
		case <-time.After(time.Second * time.Duration(bz.PoRTxDuration)):
			bz.sendPoRTx()

		case msg := <-bz.PreparedBlockChan:
			log.Lvl2(bz.Info(), "received block from", msg.TreeNode.ServerIdentity.Address)
			if !fail {
				_, err = bz.handleBlock(msg)
			}
		// this msg is catched in simulation codes
		// and do what? (FileName: ?)
		//case <-bz.DoneBaseDFS:
		// 	running = false

		/* -----------------------------------------------------------
		These section are borrowed from ByzCoin
		and may be useful for future functionalities
		------------------------------------------------------------ */

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
				//log.Lvl2(bz.Info(), "sending hello to", c.ServerIdentity.Address, c.ID, "timeout", bz.timeout)
				err := bz.SendTo(c, &HelloBaseDFS{Timeout: bz.timeout})
				if err != nil {
					log.Lvl2(bz.Info(), "couldn't send hello to child",
						c.Name())
				}
			}(child)
		}
		bz.createEpochBlock(bz.checkLeadership())
		//bz.sendPoRTx()
	} else {
		bz.createEpochBlock(bz.checkLeadership())
		//bz.sendPoRTx()
	}
}

//sendPoRTx send a por tx
func (bz *BaseDFS) sendPoRTx() {
	//txs := bz.createPoRTx()
	//portx := &Por{*txs, uint32(bz.Index())}
	//log.Lvl2(bz.Name(), ": multicasting por tx")
	//bz.Multicast(portx, bz.List()...)

	// for _, child := range bz.Children() {
	// 	err := bz.SendTo(child, portx)
	// 	if err != nil {
	// 		log.Lvl1(bz.Info(), "couldn't send to child:", child.ServerIdentity.Address)
	//	} else {
	// 		log.Lvl1(bz.Info(), "sent his PoR to his children:", child.ServerIdentity.Address)
	//	}
	//}
}

// handleAnnouncement pass the announcement to the right CoSi struct.
func (bz *BaseDFS) handlePoRTx(proofOfRet ProofOfRetTxChan) error {
	/*if refuse, err := verifyPoR(proofOfRet); err == nil {
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
	}*/
	return nil
}

//appendPoRTx: append the recieved tx to his current temporary block
func (bz *BaseDFS) appendPoRTx(p ProofOfRetTxChan) error {
	//bz.transactions = append(bz.transactions, p.por.Tx)
	// for test creating block:
	//bz.createEpochBlock(&leadershipProof{1})
	return nil
}

// ------------   Sortition Algorithm from ALgorand:
// ⟨hash,π⟩←VRFsk(seed||role)
// p←τ/W
// j←0
// while hash/ 2^hashleng </ [ sigma(k=0,j) (B(k,w,p),  sigma(k=0,j+1) (B(k,w,p)] do
//	----	 j++
// return <hash,π, j>
// ------------
//checkLeadership
func (bz *BaseDFS) checkLeadership() (bool, crypto.VrfProof) {
	//the seed of this round is publicly known in advance
	var toBeHashed = bz.currentRoundSeed
	proof, ok := bz.ECPrivateKey.ProveBytes(toBeHashed[:])
	if !ok {
		log.LLvl2("error while generating proof")
	}
	_, vrfOutput := bz.ECPrivateKey.Pubkey().VerifyBytes(proof, toBeHashed[:])
	t := binary.BigEndian.Uint64(vrfOutput[:])
	if math.Mod(float64(t), 7) == 0 {
		return true, proof
	} // ToDo : raha:  replace with the number fo nodes! temp leader election
	return false, proof
}

// verifyLeadership
//func verifyLeadership(roundNumber int, )

//createEpochBlock: by leader
func (bz *BaseDFS) createEpochBlock(ok bool, p crypto.VrfProof) *blockchain.TrBlock {
	if ok == false {
		log.LLvl2(bz.Info(), "is not the leader")
	} else {
		log.LLvl2(bz.Info(), "is the leader, generating block ... ")
		// later: add appropriate payment tx.s for each por tx in temp transactions list
		var h blockchain.TransactionList = blockchain.NewTransactionList(bz.transactions, len(bz.transactions))
		bz.tempBlock = blockchain.NewTrBlock(h, blockchain.NewHeader(h, "raha", "raha"))
		bz.roundNumber = bz.roundNumber + 1
		return bz.tempBlock
	}
	return nil
}

//SendFinalBlock   bz.SendFinalBlock(createEpochBlock)
func (bz *BaseDFS) SendFinalBlock(fb *blockchain.TrBlock) {
	if fb == nil {
		return
	} else { // it's the leader
		var err error
		for _, n := range bz.Tree().List() {
			// don't send to ourself
			if n.ID.Equal(bz.TreeNode().ID) {
				continue
			}
			err = bz.SendTo(n, &PreparedBlock{fb.Block})
			if err != nil {
				log.Error(bz.Name(), "Error sending new block", err)
			}
		}
	}
	log.Lvl1("final block hash is:", fb.Block.HeaderHash)
	return
}

// handle the arrival of a block
func (bz *BaseDFS) handleBlock(pb PreparedBlockChan) (*blockchain.TrBlock, error) {
	t, _ := bz.checkLeadership()
	if t == false {
		if err := verifyBlock(pb.PreparedBlock); err == nil {
			//they eliminate existing tx.s in block from their current temp tx list
			cap := cap(bz.transactions)
			txs := bz.transactions
			bz.transactions = nil
			for _, tx1 := range pb.PreparedBlock.Block.Txs {
				for index, tx2 := range txs {
					if tx1.Hash == tx2.Hash {
						if cap > index+1 {
							txs = append(txs[:index], txs[index+1:]...)
						} else {
							txs = txs[:index]
						}
					}
				}
			}
			bz.transactions = append(bz.transactions, txs...)
			return (bz.appendBlock(pb.PreparedBlock))
			//-----
			bz.roundNumber = bz.roundNumber + 1
			bz.blockChainSize = bz.blockChainSize + 1 //uint64(length(blockchain.Block))
		} else {
			log.Error(bz.Name(), "Error verying block", err)
			return nil, nil
		}
	}
	return nil, nil
}

// verifyBlock: servers will verify proposed block when they recieve it
func verifyBlock(pb PreparedBlock) error {
	return nil
}

//appendBlock:
func (bz *BaseDFS) appendBlock(pb PreparedBlock) (*blockchain.TrBlock, error) {
	//measure block's size
	return nil, nil
}

//-------------------------------------------------------------------
// this section is borrowed from ByzCoin - may need them later but not sure we should keep them
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
// type viewChange2 struct {
// 	LastBlock [sha256.Size]byte
// }

// // newViewChange creates a new view change.
// func newViewChange2() *viewChange2 {
// 	res := &viewChange2{}
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
