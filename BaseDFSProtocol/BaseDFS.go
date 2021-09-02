/*
Base DFS protocol: The simple version:

we assume one server broadcast his por tx and all servers on recieving this tx will verify and add it to their prepared block and then all server run leader election to check if they are leader, then the leader add his proof and broadcast his prepared block and all servers verify and append his block to their blockchain
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
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/group/edwards25519"
	"go.dedis.ch/kyber/v3/sign/schnorr"
	"go.dedis.ch/kyber/v3/suites"
	"go.dedis.ch/kyber/v3/util/key"
	"go.dedis.ch/kyber/v3/xof/blake2xb"
	"math"
	"strconv"
	"sync"
	"time"

	onet "github.com/basedfs"
	"github.com/basedfs/blockchain"
	"github.com/basedfs/blockchain/blkparser"
	"github.com/basedfs/log"
	"github.com/basedfs/network"
	"github.com/basedfs/simul/monitor"
)

func init() {

	network.RegisterMessage(ProofOfRetTxChan{})
	network.RegisterMessage(PreparedBlockChan{})
	network.RegisterMessage(HelloBaseDFS{})
	onet.GlobalProtocolRegister("BaseDFS", NewBaseDFSProtocol)
}

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
	por
}

type PreparedBlockChan struct {
	*onet.TreeNode
	PreparedBlock
}

type PreparedBlock struct {
	blockchain.Block
}

type leadershipProof struct {
	U uint32
}

type tx struct{
	i int8
}

// baseDFS is the main struct for running the protocol
type BaseDFS struct {
	// the node we are represented-in
	*onet.TreeNodeInstance
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
	epochDuration uint32
	PoRTxDuration uint32
	// finale block that this BaseDFS epoch has produced
	finalBlock *blockchain.TrBlock
	currentRandomSeed string
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
	//onCheckEpochLeadership func()//
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
		currentRandomSeed: "testrandomseed",
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
	bz.epochDuration = 5
	bz.PoRTxDuration = 5

	n.OnDoneCallback(bz.nodeDone) // raha: when this function is called?
	return bz, nil
}

//Start: starts the simplified protocol by sending hello msg to all nodes which later makes them to start sending their por tx.s
func (bz *BaseDFS) Start() error {
	bz.randomizedFileStoring()
	//bz.helloBaseDFS()
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
		select {

		case msg := <-bz.HelloChan:
			log.Lvl2(bz.Info(), "received Hello from", msg.TreeNode.ServerIdentity.Address)
			bz.helloBaseDFS()

		case msg := <-bz.ProofOfRetTxChan:
			log.Lvl2(bz.Info(), "received por", msg.por.sigma, "tx from", msg.TreeNode.ServerIdentity.Address)
			if !fail {
				err = bz.handlePoRTx(msg)
			}

		case <-time.After(time.Second * time.Duration(bz.epochDuration)):
			bz.SendFinalBlock(bz.createEpochBlock(bz.checkLeadership()))

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
		bz.sendPoRTx()
	} else {
		bz.sendPoRTx()
	}
}

//sendPoRTx send a por tx
func (bz *BaseDFS) sendPoRTx() {
	//txs := bz.createPoRTx()
	//portx := &Por{*txs, uint32(bz.Index())}
	log.Lvl2(bz.Name(), ": multicasting por tx")
	//bz.Multicast(portx, bz.List()...)

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
	if refuse, err := verifyPoR(proofOfRet); err == nil {
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

//appendPoRTx: append the recieved tx to his current temporary block
func (bz *BaseDFS) appendPoRTx(p ProofOfRetTxChan) error {
	//bz.transactions = append(bz.transactions, p.por.Tx)
	// for test creating block:
	//bz.createEpochBlock(&leadershipProof{1})
	return nil
}

//checkLeadership
func (bz *BaseDFS) checkLeadership() network.ServerIdentityID {
	n := bz.Roster().RandomServerIdentity()
	log.Lvl2(bz.Info(), "random server is", n.Address)
	return n.GetID()
}

//createEpochBlock: by leader
func (bz *BaseDFS) createEpochBlock(leaderServerIdentityID network.ServerIdentityID) *blockchain.
	TrBlock {
	if bz.ServerIdentity().GetID().Equal(leaderServerIdentityID) {
		// later: add appropriate payment tx.s for each por tx in temp transactions list
		var h blockchain.TransactionList = blockchain.NewTransactionList(bz.transactions, len(bz.transactions))
		bz.tempBlock = blockchain.NewTrBlock(h, blockchain.NewHeader(h, "raha", "raha"))
		return bz.tempBlock
	} else {
		return nil
	}
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

//--------------- Compact PoR -----------------
type Tau struct{
	tau string
}
type processedFile struct{
	sigma_i []string
	m_ij [][]string
}
type randomQuery struct{
	i int
	v_i int
}
type por struct {
	mu []int
	sigma kyber.Point
}
//
func (bz *BaseDFS) randomizedFileStoring()/*(Tau,processedFile)*/{

	//randomizedKeyGeneration: pubK=(alpha,ssk),prK=(v,spk)
	clientKeyPair := key.NewKeyPair(bz.suite)
	ssk := clientKeyPair.Private
	spk := clientKeyPair.Public
	//BLS keyPair
	suite := suites.MustFind("Ed25519")		// Use the edwards25519-curve
	alpha := suite.Scalar().Pick(suite.RandomStream()) // private key
	v := suite.Point().Mul(alpha, nil)          // public key

	//	------   createFileTag(Tau)
	//u1,..,us random G
	const s = 10 // number of sectors in eac block (sys. par.)
	var n int64 = 10 // number of blocks (sys. par.)
	ns := strconv.FormatInt(n, 10)

	var u[s]kyber.Scalar
	var U[s]kyber.Point
	var st string
	for j := 0; j<s; j++ {
		rand := blake2xb.New([]byte("seed"))
		suite := edwards25519.NewBlakeSHA256Ed25519WithRand(rand)
		u[j] = suite.Scalar().Pick(rand)
		U[j] = suite.Point().Mul(u[j], nil)
		st = st + u[j].String()
	}

	//Tau0 := "name"||string(n)||u1||...||us
	//Tau=Tau0||Ssig(ssk)(Tau0)
	Tau0 := "aRandomFileName"+ ns +st
	sg, _ := schnorr.Sign(bz.suite,ssk,[]byte(Tau0))
	Tau := Tau0 + string(sg)
	log.LLvl2(ssk,spk,v,st,Tau)

	//createAuthValue(Sigma_i) for block i
	//Sigma_i = Hash(name||i).P(j=1,..,s)u_i^m_ij
	//Sigma_i := hash


	//mStar = &processedFile
	//	{sigma_i:	Sigma_i,
	//	m_ij:		...
	//}
	//return Tau, mStar
}
//
func randomizedVerifyingQuery()/*randomQuery*/{
	// n,|I|: sys. par.
	//bz.currentRandomSeed
	//rq = &randomQuery{
	//	i: 		random [1,n],
	//	v_i:	v_i random G
	//return rq
}
//createPoR:
func (bz *BaseDFS) createPoR() *por {
	// input: pubK, Tau, mStar
	//Mu_j= S(Q)(v_i.m_ij)
	//sigma=P(Q)(sigma_i^v_i)
	return &por{
		//mu: ..,
		sigma: nil,
	}
}

// verifyPoR: servers will verify por tx.s when they recieve it
func verifyPoR(p ProofOfRetTxChan) (bool, error) {
	//input: PoR , pubK, Tau
	//check: e(sigma, g) =? e(PHash(name||i)^v_i.P(j=1,..,s)(u_j^mu_j,v)
	var refuse = false
	var err error = nil
	return refuse, err
}