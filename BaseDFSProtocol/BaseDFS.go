/*
	Base DFS protocol:

	Types of Messages:
	--------------------------------------------
	1- each server who is elected as leader (few servers in each ) send a msg of &NewLeader{Leaderinfo: bz.Name(), MCRoundNumber: bz.MCRoundNumber} to root node.
	2- each round, the root node send a msg of &NewRound{Seed:  seed, Power: power} to all servers including the target server's power
	3- in the bootstrapping phase, the root node send a message to all servers containing protocol config parameters.
	&HelloBaseDFS{
					Timeout:                  bz.timeout,
					PercentageTxPay:          bz.PercentageTxPay,
					MCRoundDuration:            bz.MCRoundDuration,
					BlockSize:                bz.BlockSize,
					SectorNumber:             bz.SectorNumber,
					NumberOfPayTXsUpperBound: bz.NumberOfPayTXsUpperBound,
					SimulationSeed:			  bz.SimulationSeed,
					nbrSubTrees:			  bz.nbrSubTrees,
					threshold:				  bz.threshold,
					SCRoundDuration:            bz.SCRoundDuration,
					CommitteeWindow:          bz.CommitteeWindow,
					EpochCount:					bz.EpochCount,
				}
	4- timeOut

*/
package BaseDFSProtocol

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	onet "github.com/basedfs"
	"github.com/basedfs/blockchain"
	"github.com/basedfs/blscosi/bdnproto"
	"github.com/basedfs/blscosi/protocol"
	"github.com/basedfs/log"
	"github.com/basedfs/network"
	"github.com/basedfs/por"
	"github.com/basedfs/vrf"
	"github.com/xuri/excelize/v2"
	"go.dedis.ch/kyber/v3/pairing"
	"golang.org/x/xerrors"
)

/* ------------------------------------------------------------------------------------------------
				--------------------------   TYPES  --------------------------------
------------------------------------------------------------------------------------------------  */
// ------  types used for communication (msgs) -------------------------
/* Hello is sent down the tree from the root node, every node who gets it starts the protocol and send it to its children */
type HelloBaseDFS struct {
	ProtocolTimeout time.Duration
	//---- ToDo: Do i need timeout?
	PercentageTxPay          int
	MCRoundDuration          int
	BlockSize                int
	SectorNumber             int
	NumberOfPayTXsUpperBound int
	SimulationSeed           int
	SCRoundDuration          int
	CommitteeWindow          int
	EpochCount               int
	// bls cosi config
	NbrSubTrees int
	Threshold   int
}
type HelloChan struct {
	*onet.TreeNode
	HelloBaseDFS
}

type NewLeader struct {
	LeaderTreeNodeID onet.TreeNodeID
	MCRoundNumber    int
}
type NewLeaderChan struct {
	*onet.TreeNode
	NewLeader
}

type NewRound struct {
	Seed  string
	Power uint64
}
type NewRoundChan struct {
	*onet.TreeNode
	NewRound
}

// ----------   side chain blscosi   ----------------------------------------------------------
// channel used by side chain's leader in each side chain's round
type RtLSideChainNewRound struct {
	SCRoundNumber            int
	CommitteeNodesTreeNodeID []onet.TreeNodeID
}
type RtLSideChainNewRoundChan struct {
	*onet.TreeNode
	RtLSideChainNewRound
}
type LtRSideChainNewRound struct { //fix this
	NewRound      bool
	SCRoundNumber int
}
type LtRSideChainNewRoundChan struct {
	*onet.TreeNode
	LtRSideChainNewRound
}

//  BaseDFS is the main struct for running the protocol ------------
type BaseDFS struct {
	// the node we are represented-in
	*onet.TreeNodeInstance
	ECPrivateKey vrf.VrfPrivkey
	// channel used to let all servers that the protocol has started
	HelloChan chan HelloChan
	// channel used by each round's leader to let all servers that a new round has come
	NewRoundChan chan NewRoundChan
	// channel to let nodes that the next round's leader has been specified
	NewLeaderChan chan NewLeaderChan
	// the suite we use
	//  suite network.Suite
	// to match the suit in blscosi
	Suite *pairing.SuiteBn256
	//startBCMeasure *monitor.TimeMeasure
	// onDoneCallback is the callback that will be called at the end of the protocol
	//onDoneCallback func() //ToDo: raha: define this function and call it when you want to finish the protocol + check when should it be called
	// channel to notify when we are done -- when a message is sent through this channel the runsimul.go file will catch it and finish the protocol.
	DoneBaseDFS chan bool
	// channel to notify leader elected
	LeaderProposeChan chan bool
	MCRoundNumber     int
	SCRoundNumber     int
	HasLeader         bool
	// --- just root node use these - these are used for delay evaluation
	FirstQueueWait  int
	SecondQueueWait int
	/* ------------------------------------------------------------------
	     -----  system-wide configurations params from the config file
	   ------------------------------------------------------------------
		these  params get initialized
		for the root node: "after" NewBaseDFSProtocol call (in func: Simulate in file: runsimul.go)
		for the rest of nodes node: while joining protocol by the HelloBaseDFS message
	--------------------------------------------------------------------- */
	PercentageTxPay          int
	MCRoundDuration          int
	BlockSize                int
	SectorNumber             int
	NumberOfPayTXsUpperBound int
	// ---
	ProtocolTimeout time.Duration
	SimulationSeed  int
	// -- blscosi related config params
	NbrSubTrees     int
	Threshold       int
	SCRoundDuration int
	CommitteeWindow int //ToDo: go down
	/* ------------------------------------------------------------------
	 ---------------------------  bls cosi protocol  ---------------
	--------------------------------------------------------------------- */
	BlsCosi                  *BlsCosi
	CommitteeNodesTreeNodeID []onet.TreeNodeID
	EpochCount               int
	NextSideChainLeader      onet.TreeNodeID
	// channel used by root node to trigger side chain's leader to run a new round of blscosi for side chain
	RtLSideChainNewRoundChan chan RtLSideChainNewRoundChan
	LtRSideChainNewRoundChan chan LtRSideChainNewRoundChan
	// it is initiated in the start function by root node
	BlsCosiStarted bool
}

/* ------------------------------------------------------------------------------------------------
				-------------------------- FUNCTIONS  -----------------------------
------------------------------------------------------------------------------------------------  */

/* ----------------------------------------------------------------------
	//Start: starts the protocol by sending hello msg to all nodes
------------------------------------------------------------------------ */
func (bz *BaseDFS) Start() error {
	// update the mainchainbc file with created nodes' information
	bz.finalMainChainBCInitialization()
	bz.BlsCosiStarted = true
	// ------------------------------------------------------------------------------
	// config params are sent from the leader to the other nodes in helloBasedDfs function
	bz.helloBaseDFS()
	//------- testing message sending ------

	// err := bz.SendTo(bz.Root(), &HelloBaseDFS{EpochCount: 0})
	// if err != nil {
	// 	return err
	// }
	return nil
}

// -------------------------------------- MC & SC protocols -------------------------------------- //

/* ----------------------------------------------------------------------
			 Dispatch listen on the different channels in main chain protocol
------------------------------------------------------------------------ */
func (bz *BaseDFS) Dispatch() error {
	// another dispatch function (DispatchProtocol) is called in runsimul.go that takes care of both chain's protocols
	// if !bz.IsRoot() || (bz.IsRoot() && bz.StartedBaseDFS) {
	// 	bz.DispatchProtocol()
	// } else {
	// 	return nil
	// }
	// return nil
	running := true
	var err error

	for running {
		select {
		// -----------------------------------------------------------------------------
		// ******** ALL nodes recieve this message to join the protocol and get the config values set
		// -----------------------------------------------------------------------------
		case msg := <-bz.HelloChan:
			log.Lvl2(bz.TreeNode().Name(), "received Hello/config params from", msg.TreeNode.ServerIdentity.Address)
			bz.PercentageTxPay = msg.PercentageTxPay
			bz.MCRoundDuration = msg.MCRoundDuration
			bz.BlockSize = msg.BlockSize
			bz.SectorNumber = msg.SectorNumber
			bz.NumberOfPayTXsUpperBound = msg.NumberOfPayTXsUpperBound
			bz.SimulationSeed = msg.SimulationSeed
			bz.SCRoundDuration = msg.SCRoundDuration
			bz.CommitteeWindow = msg.CommitteeWindow
			bz.EpochCount = msg.EpochCount
			// bls cosi config
			bz.NbrSubTrees = msg.NbrSubTrees
			bz.BlsCosi.Threshold = msg.Threshold
			bz.BlsCosi.Timeout = msg.ProtocolTimeout
			if msg.NbrSubTrees > 0 {
				err := bz.BlsCosi.SetNbrSubTree(msg.NbrSubTrees)
				if err != nil {
					return err
				}
			}
			bz.helloBaseDFS()
		// -----------------------------------------------------------------------------
		// ******** ALL nodes recieve this message to sync rounds
		// -----------------------------------------------------------------------------
		case msg := <-bz.NewRoundChan:
			bz.MCRoundNumber = bz.MCRoundNumber + 1
			log.Lvl4(bz.Name(), " round number ", bz.MCRoundNumber, " started at ", time.Now().Format(time.RFC3339))
			var vrfOutput [64]byte
			toBeHashed := []byte(msg.Seed)
			proof, ok := bz.ECPrivateKey.ProveBytes(toBeHashed[:])
			if !ok {
				log.Lvl2("error while generating proof")
			}
			_, vrfOutput = bz.ECPrivateKey.Pubkey().VerifyBytes(proof, toBeHashed[:])
			var vrfoutputInt64 uint64
			buf := bytes.NewReader(vrfOutput[:])
			err := binary.Read(buf, binary.LittleEndian, &vrfoutputInt64)
			if err != nil {
				// log.Lvl2("Panic Raised:\n\n")
				// panic(err)
				return xerrors.New("problem creatde after recieving msg from NewRoundChan:   " + err.Error())
			}
			// -----------
			// the criteria for selecting the leader
			if vrfoutputInt64 < msg.Power {
				// -----------
				log.Lvl4(bz.Name(), "I may be elected for round number ", bz.MCRoundNumber)
				bz.SendTo(bz.Root(), &NewLeader{LeaderTreeNodeID: bz.TreeNode().ID, MCRoundNumber: bz.MCRoundNumber})
			}
		// -----------------------------------------------------------------------------
		// ******* just the ROOT NODE (blockchain layer one) recieve this msg
		// -----------------------------------------------------------------------------
		case msg := <-bz.NewLeaderChan:
			// -----------------------------------------------------
			// rounds without a leader: in this case the leader info is filled with root node's info, transactions are going to be collected normally but
			// since the block is empty, no transaction is going to be taken from queues => leader = false
			// -----------------------------------------------------
			if msg.LeaderTreeNodeID == bz.TreeNode().ID && bz.MCRoundNumber != 1 && !bz.HasLeader && msg.MCRoundNumber == bz.MCRoundNumber {
				bz.HasLeader = true
				bz.updateBCPowerRound(bz.Tree().Search(msg.LeaderTreeNodeID).Name(), false)
				// in the case of a leader-less round
				log.Lvl1("final result MC: leader TreeNodeID: ", msg.LeaderTreeNodeID.String(), "(root node) filled round number", bz.MCRoundNumber, "with empty block")
				bz.updateBCTransactionQueueCollect()
				bz.readBCAndSendtoOthers()
				log.Lvl2("new round is announced")
				continue
			}
			// -----------------------------------------------------
			// normal rounds with a leader => leader = true
			// -----------------------------------------------------
			if !bz.HasLeader && msg.MCRoundNumber == bz.MCRoundNumber {
				// TODO: first validate the leadership proof
				log.Lvl1("final result MC: leader: ", bz.Tree().Search(msg.LeaderTreeNodeID).Name(), " is the round leader for round number ", bz.MCRoundNumber)

				// -----------------------------------------------
				// dynamically change the side chain's committee with last main chain's leader
				// the committee nodes is shifted by one and the new leader is added to be used for next epoch's side chain's committee
				// -----------------------------------------------
				// --- updating the CommitteeNodesTreeNodeID
				// -----------------------------------------------
				t := 0
				for _, a := range bz.CommitteeNodesTreeNodeID {
					if a != msg.LeaderTreeNodeID {
						t = t + 1
						continue
					} else {
						break
					}
				}
				if t != len(bz.CommitteeNodesTreeNodeID) {
					log.Lvl1("final result SC:", bz.Tree().Search(msg.LeaderTreeNodeID).Name(), " is already in the committee")
					for i, a := range bz.CommitteeNodesTreeNodeID {
						log.Lvl1("final result SC: BlsCosi committee queue: ", i, ":", bz.Tree().Search(a).Name())
					}
				} else {
					NextSideChainLeaderTreeNodeID := msg.LeaderTreeNodeID
					bz.CommitteeNodesTreeNodeID = append([]onet.TreeNodeID{NextSideChainLeaderTreeNodeID}, bz.CommitteeNodesTreeNodeID...)
					log.Lvl1("final result SC:", bz.Tree().Search(msg.LeaderTreeNodeID).Name(), "is added to side chain for the next epoch's committee")
					for i, a := range bz.CommitteeNodesTreeNodeID {
						log.Lvl1("final result SC: BlsCosi committee queue: ", i, ":", bz.Tree().Search(a).Name())
					}
				}
				// -----------------------------------------------
				bz.HasLeader = true
				bz.updateBCPowerRound(bz.Tree().Search(msg.LeaderTreeNodeID).Name(), true)
				bz.updateBCTransactionQueueCollect()
				bz.updateBCTransactionQueueTake()
			} else {
				log.Lvl2("this round already has a leader!")
			}
			//waiting for the time of round duration
			time.Sleep(time.Duration(bz.MCRoundDuration) * time.Second)
			// empty list of elected leaders  in this round
			for len(bz.NewLeaderChan) > 0 {
				<-bz.NewLeaderChan
			}
			// announce new round and give away required checkleadership info to nodes
			bz.readBCAndSendtoOthers()
			log.Lvl2("new round is announced")
		// -----------------------------------------------------------------------------
		// next side chain's leader recieves this message
		// -----------------------------------------------------------------------------
		case msg := <-bz.RtLSideChainNewRoundChan:
			bz.SCRoundNumber = msg.SCRoundNumber
			bz.BlsCosi.Msg = []byte{0xFF}
			// -----------------------------------------------
			// --- updating the next side chain's leader
			// -----------------------------------------------
			bz.CommitteeNodesTreeNodeID = msg.CommitteeNodesTreeNodeID
			var CommitteeNodesServerIdentity []*network.ServerIdentity
			if bz.SCRoundNumber == 1 {
				for _, a := range bz.CommitteeNodesTreeNodeID[0 : bz.CommitteeWindow-1] {
					CommitteeNodesServerIdentity = append(CommitteeNodesServerIdentity, bz.Tree().Search(a).ServerIdentity)
				}
				log.Lvl1("final result SC: ", bz.Name(), " is running next side chain's epoch with new committee")
				for i, a := range bz.CommitteeNodesTreeNodeID {
					log.Lvl1("final result SC: BlsCosi: next side chain's epoch committee number ", i, ":", bz.Tree().Search(a).Name())
				}
				log.Lvl1("wait")
			} else {
				log.LLvl1(len(bz.CommitteeNodesTreeNodeID))
				log.LLvl1(bz.CommitteeNodesTreeNodeID[len(bz.CommitteeNodesTreeNodeID)-(bz.CommitteeWindow):])
				for _, a := range bz.CommitteeNodesTreeNodeID[len(bz.CommitteeNodesTreeNodeID)-(bz.CommitteeWindow):] {
					CommitteeNodesServerIdentity = append(CommitteeNodesServerIdentity, bz.Tree().Search(a).ServerIdentity)
				}
				log.Lvl1("final result SC: ", bz.Name(), " is running next side chain's epoch with the same committee members:")
				for i, a := range CommitteeNodesServerIdentity {
					log.Lvl1("final result SC: BlsCosi: next side chain's epoch committee number ", i, ":", a.Address)
				}
				log.Lvl1("wait")
			}
			committeeRoster := onet.NewRoster(CommitteeNodesServerIdentity)
			// --- root should have root index of 0 (this is based on what happens in gen_tree.go)
			var x = *bz.TreeNode()
			x.RosterIndex = 0
			// ---
			bz.BlsCosi.SubTrees, err = protocol.NewBlsProtocolTree(onet.NewTree(committeeRoster, &x), bz.NbrSubTrees)
			if err == nil {
				log.Lvl1("Next bls cosi tree is: ", bz.BlsCosi.SubTrees[0].Roster.List,
					" with ", bz.Name(), " as Root \n running BlsCosi round number", bz.SCRoundNumber)
			} else {
				return xerrors.New("Problem in cosi protocol run:   " + err.Error())
			}
			// ----
			err = bz.BlsCosi.Start()
			if err != nil {
				return xerrors.New("Problem in cosi protocol run:   " + err.Error())
			}
		// -----------------------------------------------------------------------------
		// ******* just the ROOT NODE (blockchain layer one) recieve this msg
		// -----------------------------------------------------------------------------
		case msg := <-bz.LtRSideChainNewRoundChan:
			bz.SCRoundNumber = msg.SCRoundNumber
			if bz.MCRoundDuration*bz.EpochCount/bz.SCRoundDuration == bz.SCRoundNumber+1 {
				// generate and propose summery block (bz.BlsCosi.blockType = "summeryblock")
				// reset side chain round number
				log.Lvl1("final result SC: BlsCosi: epoch number ", bz.MCRoundNumber/bz.EpochCount, " finished ..")
				bz.SCRoundNumber = 0 // in side chain round number zero the summery blocks are published in side chain
				// from bc: update msg size with the "summery block"'s block size on side chain
				// call a function like collect and then take
			} else {
				// ------------- Epoch changed -----------
				if bz.SCRoundNumber == 0 { // i.e. the current published block on side chain is summery block
					// issue a sync transaction from last recent summery block to main chain
					// next meta block is going to be on top of last summery block (rather than last meta blocks!)
					log.LLvl2("final result SC: BlsCosi: a summery block Confirmed in Side Chain")
					//changing next side chain's leader for the next epoch rounds from the last miner in the main chain's window of miners
					bz.NextSideChainLeader = bz.CommitteeNodesTreeNodeID[0]
					// changing side chain's committee to last miners in the main chain's window of miners
					bz.CommitteeNodesTreeNodeID = bz.CommitteeNodesTreeNodeID[0:bz.CommitteeWindow]
					log.Lvl1("final result SC: BlsCosi: next side chain's epoch leader is: ", bz.Tree().Search(bz.NextSideChainLeader).Name())
					for i, a := range bz.CommitteeNodesTreeNodeID {
						log.Lvl1("final result SC: BlsCosi: next side chain's epoch committee number ", i, ":", bz.Tree().Search(a).Name())
					}
				}
				// increase side chain round number
				bz.SCRoundNumber = bz.SCRoundNumber + 1
				// from bc: update msg size with next "meta block"'s block size on side chain
				bz.BlsCosi.Msg = []byte{0xFF} // Msg is the meta block
			}
			// side chain round duration pause
			time.Sleep(time.Duration(bz.SCRoundDuration) * time.Second)
			// --------------------------------------------------------------------
			//triggering next side chain round leader to run next round of blscosi
			// --------------------------------------------------------------------
			err = bz.SendTo(bz.Tree().Search(bz.NextSideChainLeader), &RtLSideChainNewRound{
				SCRoundNumber:            bz.SCRoundNumber,
				CommitteeNodesTreeNodeID: bz.CommitteeNodesTreeNodeID,
			})
			if err != nil {
				log.Lvl2(bz.Name(), "can't send new side chain round msg to", bz.Tree().Search(bz.NextSideChainLeader).Name())
				return xerrors.New("can't send new side chain round msg to next leader" + err.Error())
			}
			// ------------------------------------------------------------------
			// this msg is catched in runsimul.go in func Simulate() function and means the baseddfs protocol has finished
			// -----------------------------------------------------------------------------
			// case <-bz.DoneBaseDFS:
			// running = false //TODO: do something about running!
		}
	}
	return err
}

/* ----------------------------------------------------------------------
 DispatchProtocol listen on the different channels in side chain protocol
------------------------------------------------------------------------ */
func (bz *BaseDFS) DispatchProtocol() error {

	running := true
	var err error

	for running {
		select {

		// --------------------------------------------------------
		// message recieved from BLSCoSi (SideChain):
		// ******* just the current side chain's leader on recieve this msg
		// --------------------------------------------------------
		case sig := <-bz.BlsCosi.FinalSignature:
			if err := bdnproto.BdnSignature(sig).Verify(bz.BlsCosi.Suite, bz.BlsCosi.Msg, bz.BlsCosi.SubTrees[0].Roster.Publics()); err == nil {
				log.LLvl2("final result SC:", bz.Name(), " : ", bz.BlsCosi.BlockType, "with side chain's round number", bz.SCRoundNumber, "Confirmed in Side Chain")
				err := bz.SendTo(bz.Root(), &LtRSideChainNewRound{
					NewRound:      true,
					SCRoundNumber: bz.SCRoundNumber,
				})
				if err != nil {
					return xerrors.New("can't send new round msg to root" + err.Error())
				}
			} else {
				return xerrors.New("The recieved final signature is not accepted:  " + err.Error())
			}
		}
	}
	return err
}

/* ----------------------------------------------------------------------
 helloBaseDFS
------------------------------------------------------------------------ */

func (bz *BaseDFS) helloBaseDFS() {
	var err error
	log.Lvl2(bz.TreeNode().Name(), " joined to the protocol")
	// all nodes get here and start to listen for blscosi protocol messages
	go func() {
		err := bz.DispatchProtocol()
		if err != nil {
			log.Lvl1("protocol dispatch calling error: " + err.Error())
			panic("protocol dispatch calling error")
		}
	}()

	if bz.IsRoot() && bz.BlsCosiStarted {
		// ----------------------------------------------------------------------------------------------
		// ---------------- BLS CoSi protocol (Initialization for root node) --------
		// ----------------------------------------------------------------------------------------------
		// -----------------------------------------------
		// --- initializing side chain's msg and side chain's committee roster index for the second run and the next runs
		// -----------------------------------------------
		for i := 0; i < bz.CommitteeWindow; i++ {
			bz.CommitteeNodesTreeNodeID = append(bz.CommitteeNodesTreeNodeID, bz.Tree().List()[i].ID)
		}
		bz.BlsCosi.Msg = []byte{0xFF}
		// -----------------------------------------------
		// --- initializing next side chain's leader
		// -----------------------------------------------
		bz.CommitteeNodesTreeNodeID = append([]onet.TreeNodeID{bz.Tree().List()[bz.CommitteeWindow+1].ID}, bz.CommitteeNodesTreeNodeID[:bz.CommitteeWindow-1]...)
		bz.NextSideChainLeader = bz.Tree().List()[bz.CommitteeWindow+1].ID
		// --------------------------------------------------------------------
		//triggering next side chain round leader to run next round of blscosi
		// --------------------------------------------------------------------
		err = bz.SendTo(bz.Tree().Search(bz.NextSideChainLeader), &RtLSideChainNewRound{
			SCRoundNumber:            bz.SCRoundNumber,
			CommitteeNodesTreeNodeID: bz.CommitteeNodesTreeNodeID,
		})
		if err != nil {
			log.Lvl2(bz.Name(), "can't send new side chain round msg to", bz.Tree().Search(bz.NextSideChainLeader).Name())
			panic("can't send new side chain round msg to the first leader")
		}
		// ----------------------------------------------------------------------------------------------
		// -------------------------- main chain's protocol ---------------------------------------
		// ----------------------------------------------------------------------------------------------
		// let other nodes join the protocol first -
		//time.Sleep(time.Duration(len(bz.Roster().List)) * time.Second)

		// the root node is filling the first block in first round
		log.Lvl2(bz.Name(), "Filling round number ", bz.MCRoundNumber)
		// for the first round we have the root node set as a round leader, so  it is true! and he takes txs from the queue
		bz.updateBCPowerRound(bz.TreeNode().Name(), true)
		bz.updateBCTransactionQueueCollect()
		bz.updateBCTransactionQueueTake()
		time.Sleep(time.Duration(bz.MCRoundDuration) * time.Second)
		bz.readBCAndSendtoOthers()
		// protocol timeout
		go bz.StartTimeOut()
	}
	// ----------------------------------------------------------------------------------------------
	// ---------------- BLS CoSi protocol (running for the very first time) --------
	// ----------------------------------------------------------------------------------------------
	// this node has been set to start running blscosi in simulation level. (runsimul.go):
	/* 	if bz.Tree().List()[bz.CommitteeWindow] == bz.TreeNode() {
		// -----------------------------------------------
		// --- initializing side chain's msg and side chain's committee roster index for the first run
		// -----------------------------------------------
		for i := 0; i < bz.CommitteeWindow; i++ {
			bz.CommitteeNodesTreeNodeID = append(bz.CommitteeNodesTreeNodeID, bz.Tree().List()[i].ID)
		}
		bz.BlsCosi.Msg = []byte{0xFF}

		bz.CommitteeNodesTreeNodeID = append([]onet.TreeNodeID{bz.Tree().List()[bz.CommitteeWindow].ID}, bz.CommitteeNodesTreeNodeID[:bz.CommitteeWindow-1]...)
		var CommitteeNodesServerIdentity []*network.ServerIdentity
		for _, a := range bz.CommitteeNodesTreeNodeID {
			CommitteeNodesServerIdentity = append(CommitteeNodesServerIdentity, bz.Tree().Search(a).ServerIdentity)
		}
		committeeRoster := onet.NewRoster(CommitteeNodesServerIdentity)
		// --- root should have root index of 0 (this is based on what happens in gen_tree.go)
		var x = *bz.Tree().List()[bz.CommitteeWindow]
		x.RosterIndex = 0
		// ---
		bz.BlsCosi.SubTrees, err = protocol.NewBlsProtocolTree(onet.NewTree(committeeRoster, &x), bz.NbrSubTrees)
		if err == nil {
			log.Lvl1("final result: BlsCosi: First bls cosi tree is: ", bz.BlsCosi.SubTrees[0].Roster.List,
				" with ", bz.BlsCosi.Name(), " as Root starting BlsCosi")
		} else {
			log.Lvl1("Raha: error: ", err)
		}
		// ------------------------------------------
		err = bz.BlsCosi.Start() // first leader in side chain is this node
		if err != nil {
			log.Lvl1(bz.Info(), "couldn't start side chain")
		}
	} */
}

// -------------------------------------- MC Blockchain -------------------------------------- //

/* ----------------------------------------------------------------------
	finalMainChainBCInitialization initialize the mainchainbc file based on the config params defined in the config file
	(.toml file of the protocol) the info we hadn't before and we have now is nodes' info that this function add to the mainchainbc file
------------------------------------------------------------------------ */
func (bz *BaseDFS) finalMainChainBCInitialization() {
	var NodeInfoRow []string
	for _, a := range bz.Roster().List {
		NodeInfoRow = append(NodeInfoRow, a.String())
	}
	var err error
	f, err := excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/mainchainbc.xlsx")
	if err != nil {
		xerrors.New("problem creatde while opening bc:   " + err.Error())
	} else {
		log.Lvl3("opening bc")
	}

	// --- market matching sheet
	index := f.GetSheetIndex("MarketMatching")
	f.SetActiveSheet(index)
	// column format
	for i := 2; i <= len(NodeInfoRow)+1; i++ {
		contractRow := strconv.Itoa(i)
		t := "A" + contractRow
		err = f.SetCellValue("MarketMatching", t, NodeInfoRow[i-2])
	}
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}
	// --- power table sheet
	index = f.GetSheetIndex("PowerTable")
	f.SetActiveSheet(index)
	err = f.SetSheetRow("PowerTable", "B1", &NodeInfoRow)
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}

	err = f.SaveAs("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/mainchainbc.xlsx")
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	} else {
		log.Lvl3("closing bc")
	}
}

/* ----------------------------------------------------------------------
each round THE ROOT NODE send a msg to all nodes,
let other nodes know that the new round has started and the information they need
from blockchain to check if they are next round's leader
------------------------------------------------------------------------ */
func (bz *BaseDFS) readBCAndSendtoOthers() {
	powers, seed := bz.readBCPowersAndSeed()
	bz.MCRoundNumber = bz.MCRoundNumber + 1
	bz.HasLeader = false
	for _, b := range bz.Tree().List() {
		power, found := powers[b.ServerIdentity.String()]
		if found && !b.IsRoot() {
			err := bz.SendTo(b, &NewRound{
				Seed:  seed,
				Power: power})
			if err != nil {
				log.Lvl2(bz.Info(), "can't send new round msg to", b.Name())
				panic(err)
			}
		}
	}
	// detecting leader-less in next round
	go bz.startTimer(bz.MCRoundNumber)
}

/* ----------------------------------------------------------------------*/
func (bz *BaseDFS) readBCPowersAndSeed() (powers map[string]uint64, seed string) {
	minerspowers := make(map[string]uint64)
	f, err := excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/mainchainbc.xlsx")
	if err != nil {
		log.Lvl2("Raha: ", err)
		panic(err)
	} else {
		log.Lvl3("opening bc")
	}
	//var err error
	var rows *excelize.Rows
	var row []string
	rowNumber := 0
	// looking for last round's seed in the round table sheet in the mainchainbc file
	if rows, err = f.Rows("RoundTable"); err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}
	for rows.Next() {
		rowNumber++
		if row, err = rows.Columns(); err != nil {
			log.Lvl2("Panic Raised:\n\n")
			panic(err)
		}
	}
	// last row:
	for i, colCell := range row {
		// --- in RoundTable: i = 0 is (next) round number, i = 1 is (next) round seed, i=2 is blockchain size (empty now, will be updated by the leader)
		// if i == 0 {
		// 	if bz.MCRoundNumber, err = strconv.Atoi(colCell); err != nil {
		// 		log.Lvl2("Panic Raised:\n\n")
		// 		panic(err)
		// 	}
		// }
		if i == 1 {
			seed = colCell // last round's seed
		}
	}
	// looking for all nodes' power in the last round in the power table sheet in the mainchainbc file
	rowNumber = 0 //ToDo: later it can go straight to last row based on the round number found in round table
	if rows, err = f.Rows("PowerTable"); err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}
	for rows.Next() {
		rowNumber++
	}
	// last row in power table:
	// if row, err = rows.Columns(); err != nil {
	// 	log.Lvl2("Panic Raised:\n\n")
	// 	panic(err)
	// }

	for _, a := range bz.Roster().List {
		var myColumnHeader []string
		var myCell string
		var myColumn string
		myColumnHeader, err = f.SearchSheet("PowerTable", a.Address.String())
		if err != nil {
			log.Lvl2("Panic Raised:\n\n")
			panic(err)
		}
		for _, character := range myColumnHeader[0] {
			if character >= 'A' && character <= 'Z' { // a-z isn't needed! just to make sure
				myColumn = myColumn + string(character)
			}
		}

		myCell = myColumn + strconv.Itoa(rowNumber) //such as A2,B3,C3..
		var p string
		p, err = f.GetCellValue("PowerTable", myCell)
		if err != nil {
			log.Lvl2("Panic Raised:\n\n")
			panic(err)
		}
		var t int
		t, err = strconv.Atoi(p)
		if err != nil {
			log.Lvl2("Panic Raised:\n\n")
			panic(err)
		}
		minerspowers[a.Address.String()] = uint64(t)
		// add accumulated power to recently added power in last round
		for i := 2; i < rowNumber; i++ {
			upperPowerCell := myColumn + strconv.Itoa(i)
			p, err = f.GetCellValue("PowerTable", upperPowerCell)
			if err != nil {
				log.Lvl2("Panic Raised:\n\n")
				panic(err)
			}
			t, er := strconv.Atoi(p)
			if er != nil {
				log.Lvl2("Panic Raised:\n\n")
				panic(err)
			}
			upperPower := uint64(t)
			minerspowers[a.Address.String()] = minerspowers[a.Address.String()] + upperPower
		}
	}

	return minerspowers, seed
}

/* ----------------------------------------------------------------------
	updateBC: by leader
	each round, adding one row in power table based on the information in market matching sheet,
	assuming that servers are honest and have honestly publish por for their actice (not expired) contracts,
	for each storage server and each of their active contract, add the stored file size to their current power
 ----------------------------------------------------------------------*/
func (bz *BaseDFS) updateBCPowerRound(LeaderName string, leader bool) {
	var err error
	var rows *excelize.Rows
	var row []string
	var seed string
	rowNumber := 0

	f, err := excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/mainchainbc.xlsx")
	if err != nil {
		log.Lvl2("Raha: ", err)
		panic(err)
	} else {
		log.Lvl3(bz.Name(), "opening bc")
	}

	// looking for last round's seed in the round table sheet in the mainchainbc file
	if rows, err = f.Rows("RoundTable"); err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}
	for rows.Next() {
		rowNumber++
		if row, err = rows.Columns(); err != nil {
			log.Lvl2("Panic Raised:\n\n")
			panic(err)
		}
	}
	for i, colCell := range row {
		// --- in RoundTable: i = 0 is (next) round number, i = 1 is (next) round seed, i=2 is blockchain size (empty now, will be updated by the leader)
		//if i == 0 {if bz.MCRoundNumber,err = strconv.Atoi(colCell); err!=nil {log.Lvl2(err)}}  // i dont want to change the round number now, even if it is changed, i will re-check it at the end!
		if i == 1 {
			seed = colCell
		} // next round's seed is the hash of this seed
	}
	// --------------------------------------------------------------------
	// updating the current last row in the "BCsize" column

	// no longer nodes read from mainchainbc file, instead the round are determined by their local round number variable
	currentRow := strconv.Itoa(rowNumber)
	nextRow := strconv.Itoa(rowNumber + 1) //ToDo: raha: remove these, use bz.MCRoundNumber instead!
	// ---
	axisBCSize := "C" + currentRow
	err = f.SetCellValue("RoundTable", axisBCSize, bz.BlockSize)
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}
	// --------------------------------------------------------------------
	// --- set starting round time
	// --------------------------------------------------------------------
	cellStartTime := "J" + currentRow
	if leader {
		err = f.SetCellValue("RoundTable", cellStartTime, time.Now().Format(time.RFC3339))
	} else {
		err = f.SetCellValue("RoundTable", cellStartTime, time.Now().Format(time.RFC3339)+" - round duration")
	}

	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}
	// --------------------------------------------------------------------
	// updating the current last row in the "miner" column
	axisMiner := "D" + currentRow
	err = f.SetCellValue("RoundTable", axisMiner, LeaderName)
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}
	// --------------------------------------------------------------------
	// adding one row in round table (round number and seed columns)
	axisMCRoundNumber := "A" + nextRow
	err = f.SetCellValue("RoundTable", axisMCRoundNumber, bz.MCRoundNumber+1)
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}
	// ---  next round's seed is the hash of current seed
	data := fmt.Sprintf("%v", seed)
	sha := sha256.New()
	if _, err := sha.Write([]byte(data)); err != nil {
		log.Error("Couldn't hash header:", err)
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}
	hash := sha.Sum(nil)
	axisSeed := "B" + nextRow
	err = f.SetCellValue("RoundTable", axisSeed, hex.EncodeToString(hash))
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}

	//	each round, adding one row in power table based on the information in market matching sheet,assuming that servers are honest  and have honestly publish por for their actice (not expired) contracts,for each storage server and each of their active contracst, add the stored file size to their current power
	if rows, err = f.Rows("MarketMatching"); err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}
	var ContractDuration, ContractStartedMCRoundNumber, FileSize int
	var MinerServer string
	rowNum := 0
	MinerServers := make(map[string]int)
	for rows.Next() {
		rowNum++
		if rowNum == 1 {
			row, _ = rows.Columns()
		} else {
			row, err = rows.Columns()
			if err != nil {
				log.Lvl2("Panic Raised:\n\n")
				panic(err)
			} else {
				for i, colCell := range row {
					/* --- in MarketMatching: i = 0 is Server's Info,
					i = 1 is FileSize, i=2 is ContractDuration,
					i=3 is MCRoundNumber, i=4 is ContractID, i=5 is Client's PK,
					i = 6 is ContractPublished */
					if i == 0 {
						MinerServer = colCell
					}
					if i == 1 {
						FileSize, err = strconv.Atoi(colCell)
						if err != nil {
							log.Lvl2("Panic Raised:\n\n")
							panic(err)
						}
					}
					if i == 2 {
						ContractDuration, err = strconv.Atoi(colCell)
						if err != nil {
							log.Lvl2("Panic Raised:\n\n")
							panic(err)
						}
					}
					if i == 3 {
						ContractStartedMCRoundNumber, err = strconv.Atoi(colCell)
						if err != nil {
							log.Lvl2("Panic Raised:\n\n")
							panic(err)
						}
					}
				}
			}
			if bz.MCRoundNumber-ContractStartedMCRoundNumber <= ContractDuration {
				MinerServers[MinerServer] = FileSize //if each server one contract
			} else {
				MinerServers[MinerServer] = 0
			}
		}
	}
	// --- Power Table sheet  ----------------------------------------------
	index := f.GetSheetIndex("PowerTable")
	f.SetActiveSheet(index)
	var PowerInfoRow []int
	for _, a := range bz.Roster().List {
		PowerInfoRow = append(PowerInfoRow, MinerServers[a.Address.String()])
	}
	axisMCRoundNumber = "B" + currentRow
	err = f.SetSheetRow("PowerTable", axisMCRoundNumber, &PowerInfoRow)
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}
	// --------------------------------------------------------------------
	// adding current round's round number
	axisMCRoundNumber = "A" + currentRow
	err = f.SetCellValue("PowerTable", axisMCRoundNumber, bz.MCRoundNumber)
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}
	// ----
	err = f.SaveAs("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/mainchainbc.xlsx")
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	} else {
		log.Lvl3("closing bc")
	}
}

/* ----------------------------------------------------------------------
    updateBC: this is a connection between first layer of blockchain - ROOT NODE - on the second layer - xlsx file -
------------------------------------------------------------------------ */
func (bz *BaseDFS) updateBCTransactionQueueCollect() {
	var err error
	var rows *excelize.Rows
	var row []string

	f, err := excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/mainchainbc.xlsx")
	if err != nil {
		log.Lvl2("Raha: ", err)
		panic(err)
	} else {
		log.Lvl3("opening bc")
	}
	// -------------------------------------------------------------------------------
	// each round, adding one row in power table based on the information in market matching sheet,
	// assuming that servers are honest and have honestly publish por for their actice (not expired) contracts,
	// for each storage server and each of their active contracst, add the stored file size to their current power
	// -------------------------------------------------------------------------------
	if rows, err = f.Rows("MarketMatching"); err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}
	var ContractDuration, ContractStartedMCRoundNumber, FileSize, ContractPublished, ContractID int
	var MinerServer, ContractIDString string

	rowNum := 0
	transactionQueue := make(map[string][5]int)
	// first int: stored file size in this round,
	// second int: corresponding contract id
	// third int: TxContractPropose required
	// fourth int: TxStoragePayment required
	for rows.Next() {
		rowNum++
		if rowNum == 1 { // first row is header
			_, _ = rows.Columns()
		} else {
			row, err = rows.Columns()
			if err != nil {
				log.Lvl2("Panic Raised:\n\n")
				panic(err)
			} else {
				for i, colCell := range row {
					/* --- in MarketMatching:
					i = 0 is Server's Info,
					i = 1 is FileSize,
					i=2 is ContractDuration,
					i=3 is MCRoundNumber,
					i=4 is ContractID,
					i=5 is Client's PK,
					i = 6 is ContractPublished */
					if i == 0 {
						MinerServer = colCell
					}
					if i == 1 {
						FileSize, err = strconv.Atoi(colCell)
						if err != nil {
							log.Lvl2("Panic Raised:\n\n")
							panic(err)
						}
					}
					if i == 2 {
						ContractDuration, err = strconv.Atoi(colCell)
						if err != nil {
							log.Lvl2("Panic Raised:\n\n")
							panic(err)
						}
					}
					if i == 3 {
						ContractStartedMCRoundNumber, err = strconv.Atoi(colCell)
						if err != nil {
							log.Lvl2("Panic Raised:\n\n")
							panic(err)
						}
					}
					if i == 4 {
						ContractIDString = colCell
						ContractID, err = strconv.Atoi(colCell)
						if err != nil {
							log.Lvl2("Panic Raised:\n\n")
							panic(err)
						}
					}
					if i == 5 {
						ContractPublished, err = strconv.Atoi(colCell)
						if err != nil {
							log.Lvl2("Panic Raised:\n\n")
							panic(err)
						}
					}
				}
			}
			t := [5]int{0, ContractID, 0, 0, 0}
			// map transactionQueue:
			// t[0]: stored file size in this round,
			// t[1]: corresponding contract id
			// t[2]: TxContractPropose required
			// t[3]: TxStoragePayment required
			// t[4]: TxPor required
			if ContractPublished == 0 {
				// Add TxContractPropose
				t[2] = 1
				transactionQueue[MinerServer] = t
			} else if bz.MCRoundNumber-ContractStartedMCRoundNumber <= ContractDuration { // contract is not expired
				t[0] = FileSize //if each server one contract
				// Add TxPor
				t[4] = 1
				transactionQueue[MinerServer] = t
			} else if bz.MCRoundNumber-ContractStartedMCRoundNumber > ContractDuration {
				// Set ContractPublished to false
				if contractIdCellMarketMatching, err := f.SearchSheet("MarketMatching", ContractIDString); err != nil {
					log.Lvl2("Panic Raised:\n\n")
					panic(err)
				} else {
					publishedCellMarketMatching := "F" + contractIdCellMarketMatching[0][1:]
					err = f.SetCellValue("MarketMatching", publishedCellMarketMatching, 0)
					if err != nil {
						log.Lvl2("Panic Raised:\n\n")
						panic(err)
					}
				}
				// Add TxStoragePayment
				t[3] = 1
				transactionQueue[MinerServer] = t
			}
		}
	}

	// ----------------------------------------------------------------------
	// ------ add 5 types of transactions into transaction queue sheet -----
	// ----------------------------------------------------------------------
	/* each transaction has the following column stored on the transaction queue sheet:
	0) name
	1) size
	2) time
	3) issuedMCRoundNumber
	4) contractId */
	var newTransactionRow [5]string
	s := make([]interface{}, len(newTransactionRow)) //todo: raha:  check this out later: https://stackoverflow.com/questions/23148812/whats-the-meaning-of-interface/23148998#23148998

	newTransactionRow[2] = time.Now().Format(time.RFC3339)
	newTransactionRow[3] = strconv.Itoa(bz.MCRoundNumber)
	// this part can be moved to protocol initialization
	var PorTxSize, ContractProposeTxSize, PayTxSize, StoragePayTxSize, ContractCommitTxSize int
	PorTxSize, ContractProposeTxSize, PayTxSize, StoragePayTxSize, ContractCommitTxSize = blockchain.TransactionMeasurement(bz.SectorNumber, bz.SimulationSeed)
	// ---
	addCommitTx := false
	// map transactionQueue:
	// [0]: stored file size in this round,
	// [1]: corresponding contract id
	// [2]: TxContractPropose required
	// [3]: TxStoragePayment required
	// [4]: TxPor required
	for _, a := range bz.Roster().List {
		if transactionQueue[a.Address.String()][2] == 1 { //TxContractPropose required
			newTransactionRow[0] = "TxContractPropose"
			newTransactionRow[1] = strconv.Itoa(ContractProposeTxSize)
			newTransactionRow[4] = strconv.Itoa(transactionQueue[a.Address.String()][1]) // corresponding contract id
			addCommitTx = true                                                           // another row will be added containing "TxContractCommit"
		} else if transactionQueue[a.Address.String()][3] == 1 { // TxStoragePayment required
			newTransactionRow[0] = "TxStoragePayment"
			newTransactionRow[1] = strconv.Itoa(StoragePayTxSize)
			newTransactionRow[4] = strconv.Itoa(transactionQueue[a.Address.String()][1])
		} else if transactionQueue[a.Address.String()][4] == 1 { // TxPor required
			newTransactionRow[0] = "TxPor"
			newTransactionRow[1] = strconv.Itoa(PorTxSize)
			newTransactionRow[4] = strconv.Itoa(transactionQueue[a.Address.String()][1])
		}

		for i, v := range newTransactionRow {
			s[i] = v
		}
		if err = f.InsertRow("FirstQueue", 2); err != nil {
			log.Lvl2("Panic Raised:\n\n")
			panic(err)
		} else {
			if err = f.SetSheetRow("FirstQueue", "A2", &s); err != nil {
				log.Lvl2("Panic Raised:\n\n")
				panic(err)
			} else {
				if newTransactionRow[0] == "TxPor" {
					log.Lvl3("a TxPor added to queue in round number", bz.MCRoundNumber)
				} else if newTransactionRow[0] == "TxStoragePayment" {
					log.Lvl3("a TxStoragePayment added to queue in round number", bz.MCRoundNumber)
				} else if newTransactionRow[0] == "TxContractPropose" {
					log.Lvl3("a TxContractPropose added to queue in round number", bz.MCRoundNumber)
				}
			}
		}
		/* second row added in case of having the first row to be contract propose tx which then we will add
		contract commit tx right away.
		warning: Just in one case it may cause irrational statistics which doesn’t worth taking care of!
		when a propose contract tx is added to a block which causes the contract to become active but
		the commit contract transaction is not yet! */
		if addCommitTx {
			newTransactionRow[0] = "TxContractCommit"
			newTransactionRow[1] = strconv.Itoa(ContractCommitTxSize)
			newTransactionRow[4] = strconv.Itoa(transactionQueue[a.Address.String()][1]) // corresponding contract id
			//--
			for i, v := range newTransactionRow {
				s[i] = v
			}
			if err = f.InsertRow("FirstQueue", 2); err != nil {
				log.Lvl2("Panic Raised:\n\n")
				panic(err)
			} else {
				if err = f.SetSheetRow("FirstQueue", "A2", &s); err != nil {
					log.Lvl2("Panic Raised:\n\n")
					panic(err)
				} else {
					addCommitTx = false
					log.Lvl3("a TxContractCommit added to queue in round number", bz.MCRoundNumber)
				}
			}
		}
	}
	// -------------------------------------------------------------------------------
	// ------ add payment transactions into transaction queue payment sheet
	// -------------------------------------------------------------------------------
	/* each transaction has the following column stored on the transaction queue payment sheet:
	0) size
	1) time
	2) issuedMCRoundNumber */

	newTransactionRow[0] = strconv.Itoa(PayTxSize)
	newTransactionRow[1] = time.Now().Format(time.RFC3339)
	newTransactionRow[2] = strconv.Itoa(bz.MCRoundNumber)
	newTransactionRow[3] = ""
	newTransactionRow[4] = ""
	for i, v := range newTransactionRow {
		s[i] = v
	}

	rand.Seed(int64(bz.SimulationSeed))

	// avoid having zero regular payment txs
	var numberOfRegPay int
	for numberOfRegPay == 0 {
		numberOfRegPay = rand.Intn(bz.NumberOfPayTXsUpperBound)
	}
	log.Lvl2("Number of regular payment transactions in round number", bz.MCRoundNumber, "is", numberOfRegPay)
	for i := 1; i <= numberOfRegPay; i++ {
		if err = f.InsertRow("SecondQueue", 2); err != nil {
			log.Lvl2("Panic Raised:\n\n")
			panic(err)
		} else {
			if err = f.SetSheetRow("SecondQueue", "A2", &s); err != nil {
				log.Lvl2("Panic Raised:\n\n")
				panic(err)
			} else {
				log.Lvl3("a regular payment transaction added to queue in round number", bz.MCRoundNumber)
			}
		}
	}
	// -------------------------------------------------------------------------------

	// ---
	err = f.SaveAs("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/mainchainbc.xlsx")
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	} else {
		log.Lvl3("closing bc")
		log.Lvl2(bz.Name(), " finished collecting new transactions to queue in round number ", bz.MCRoundNumber)
	}
}

/* ----------------------------------------------------------------------
    updateBC: this is a connection between first layer of blockchain - ROOT NODE - on the second layer - xlsx file -
------------------------------------------------------------------------ */
func (bz *BaseDFS) updateBCTransactionQueueTake() {
	var err error
	var rows [][]string
	// --- reset
	bz.FirstQueueWait = 0
	bz.SecondQueueWait = 0

	f, err := excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/mainchainbc.xlsx")
	if err != nil {
		log.Lvl2("Raha: ", err)
		panic(err)
	} else {
		log.Lvl3("opening bc")
	}

	var accumulatedTxSize, txsize int
	blockIsFull := false
	NextRow := strconv.Itoa(bz.MCRoundNumber + 2)

	axisNumRegPayTx := "E" + NextRow

	axisQueue2IsFull := "N" + NextRow
	axisQueue1IsFull := "O" + NextRow

	numberOfRegPayTx := 0
	BlockSizeMinusTransactions := blockchain.BlockMeasurement()
	var TakeTime time.Time

	/* -----------------------------------------------------------------------------
		-- take regular payment transactions from sheet: SecondQueue
	----------------------------------------------------------------------------- */
	if rows, err = f.GetRows("SecondQueue"); err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}

	for i := len(rows); i > 1 && !blockIsFull; i-- {
		row := rows[i-1][:]
		/* each transaction has the following column stored on the Transaction Queue Payment sheet:
		0) size
		1) time
		2) issuedMCRoundNumber */

		for j, colCell := range row {
			if j == 0 {
				if txsize, err = strconv.Atoi(colCell); err != nil {
					log.Lvl2("Panic Raised:\n\n")
					panic(err)
				} else if 100*(accumulatedTxSize+txsize) <= (bz.PercentageTxPay)*(bz.BlockSize-BlockSizeMinusTransactions) {
					accumulatedTxSize = accumulatedTxSize + txsize
					numberOfRegPayTx++
					/* transaction name in transaction queue payment is just "TxPayment"
					the transactions are just removed from queue and their size are added to included transactions' size in block */
					log.Lvl2("a regular payment transaction added to block number", bz.MCRoundNumber, " from the queue")
					// row[1] is transaction's collected time
					if TakeTime, err = time.Parse(time.RFC3339, row[1]); err != nil {
						log.Lvl2("Panic Raised:\n\n")
						panic(err)
					}
					bz.SecondQueueWait = bz.SecondQueueWait + int(time.Now().Sub(TakeTime).Seconds())
					f.RemoveRow("SecondQueue", i)
				} else {
					blockIsFull = true
					log.Lvl3("final result MC: regular  payment share is full!")
					f.SetCellValue("RoundTable", axisQueue2IsFull, 1)
					break
				}
			}
		}
	}
	allocatedBlockSizeForRegPayTx := accumulatedTxSize
	/* -----------------------------------------------------------------------------
		 -- take 5 types of transactions from sheet: FirstQueue
	----------------------------------------------------------------------------- */

	if rows, err = f.GetRows("FirstQueue"); err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}
	// reset variables
	txsize = 0
	blockIsFull = false
	accumulatedTxSize = 0
	// new variables for first queue
	var contractIdCellMarketMatching []string

	numberOfPoRTx := 0
	numberOfStoragePayTx := 0
	numberOfContractProposeTx := 0
	numberOfContractCommitTx := 0

	axisBlockSize := "C" + NextRow

	axisNumPoRTx := "F" + NextRow
	axisNumStoragePayTx := "G" + NextRow
	axisNumContractProposeTx := "H" + NextRow
	axisNumContractCommitTx := "I" + NextRow

	axisTotalTxsNum := "K" + NextRow
	axisAveFirstQueueWait := "L" + NextRow
	axisAveSecondQueueWait := "M" + NextRow

	for i := len(rows); i > 1 && !blockIsFull; i-- {
		row := rows[i-1][:]
		/* each transaction has the following column stored on the transaction queue sheet:
		0) name
		1) size
		2) time
		3) issuedMCRoundNumber
		4) contractId */
		for j, colCell := range row {
			if j == 1 {
				if txsize, err = strconv.Atoi(colCell); err != nil {
					log.Lvl2("Panic Raised:\n\n")
					panic(err)
				} else if accumulatedTxSize+txsize <= bz.BlockSize-BlockSizeMinusTransactions-allocatedBlockSizeForRegPayTx {
					accumulatedTxSize = accumulatedTxSize + txsize
					/* transaction name in transaction queue can be "TxContractPropose", "TxStoragePayment", or "TxPor"
					in case of "TxContractCommit":
					1) The corresponding contract in marketmatching should be updated to published //todo: raha: replace the word "published" with "active"
					2) set start round number to current round
					other transactions are just removed from queue and their size are added to included transactions' size in block */
					if row[0] == "TxContractCommit" {
						/* when tx TxContractCommit left queue:
						1) set ContractPublished to True
						2) set start round number to current round */
						cid := row[4]
						if contractIdCellMarketMatching, err = f.SearchSheet("MarketMatching", cid); err != nil {
							log.Lvl2("Panic Raised:\n\n")
							panic(err)
						} else {
							publishedCellMarketMatching := "F" + contractIdCellMarketMatching[0][1:]
							err = f.SetCellValue("MarketMatching", publishedCellMarketMatching, 1)
							if err != nil {
								log.Lvl2("Panic Raised:\n\n")
								panic(err)
							} else {
								startRoundCellMarketMatching := "D" + contractIdCellMarketMatching[0][1:]
								err = f.SetCellValue("MarketMatching", startRoundCellMarketMatching, bz.MCRoundNumber)
								if err != nil {
									log.Lvl2("Panic Raised:\n\n")
									panic(err)
								} else {
									log.Lvl3("a TxContractCommit tx added to block number", bz.MCRoundNumber, " from the queue")
									numberOfContractCommitTx++
								}
							}
						}
					} else if row[0] == "TxStoragePayment" {
						log.Lvl3("a TxStoragePayment tx added to block number", bz.MCRoundNumber, " from the queue")
						numberOfStoragePayTx++
					} else if row[0] == "TxPor" {
						log.Lvl3("a por tx added to block number", bz.MCRoundNumber, " from the queue")
						numberOfPoRTx++
					} else if row[0] == "TxContractPropose" {
						log.Lvl3("a TxContractPropose tx added to block number", bz.MCRoundNumber, " from the queue")
						numberOfContractProposeTx++
					} else {
						log.Lvl2("Panic Raised:\n\n")
						panic("the type of transaction in the queue is un-defined")
					}
					// row[2] is transaction's collected time
					if TakeTime, err = time.Parse(time.RFC3339, row[2]); err != nil {
						log.Lvl2("Panic Raised:\n\n")
						panic(err)
					}
					bz.FirstQueueWait = bz.FirstQueueWait + int(time.Now().Sub(TakeTime).Seconds())
					f.RemoveRow("FirstQueue", i)
				} else {
					blockIsFull = true
					log.Lvl3("final result MC: block is full! ")
					f.SetCellValue("RoundTable", axisQueue1IsFull, 1)
					break
				}
			}
		}
	}

	f.SetCellValue("RoundTable", axisNumPoRTx, numberOfPoRTx)
	f.SetCellValue("RoundTable", axisNumStoragePayTx, numberOfStoragePayTx)
	f.SetCellValue("RoundTable", axisNumContractProposeTx, numberOfContractProposeTx)
	f.SetCellValue("RoundTable", axisNumContractCommitTx, numberOfContractCommitTx)

	log.Lvl3("In total in round number ", bz.MCRoundNumber,
		"\n number of published PoR transactions is", numberOfPoRTx,
		"\n number of published Storage payment transactions is", numberOfStoragePayTx,
		"\n number of published Propose Contract transactions is", numberOfContractProposeTx,
		"\n number of published Commit Contract transactions is", numberOfContractCommitTx,
		"\n number of published regular payment transactions is", numberOfRegPayTx,
	)
	TotalNumTxsInBothQueue := numberOfPoRTx + numberOfStoragePayTx + numberOfContractProposeTx + numberOfContractCommitTx + numberOfRegPayTx
	TotalNumTxsInFirstQueue := numberOfPoRTx + numberOfStoragePayTx + numberOfContractProposeTx + numberOfContractCommitTx

	//-- accumulated block size
	// --- total throughput
	f.SetCellValue("RoundTable", axisNumRegPayTx, numberOfRegPayTx)
	f.SetCellValue("RoundTable", axisBlockSize, accumulatedTxSize+allocatedBlockSizeForRegPayTx)
	f.SetCellValue("RoundTable", axisTotalTxsNum, TotalNumTxsInBothQueue)

	f.SetCellValue("RoundTable", axisAveFirstQueueWait, bz.FirstQueueWait/TotalNumTxsInFirstQueue)
	f.SetCellValue("RoundTable", axisAveSecondQueueWait, bz.SecondQueueWait/numberOfRegPayTx)

	log.Lvl3("final result MC: \n", allocatedBlockSizeForRegPayTx,
		" for regular payment txs,\n and ", accumulatedTxSize, " for other types of txs")
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}

	log.Lvl3("In total in round number ", bz.MCRoundNumber,
		"\n number of all types of submitted txs is", TotalNumTxsInBothQueue)

	// ---- overall results
	axisRound := "A" + NextRow
	axisBCSize := "B" + NextRow
	axisOverallRegPayTX := "C" + NextRow
	axisOverallPoRTX := "D" + NextRow
	axisOverallStorjPayTX := "E" + NextRow
	axisOverallCntPropTX := "F" + NextRow
	axisOverallCntComTX := "G" + NextRow
	axisAveWaitOtherTx := "H" + NextRow
	axisOveralAveWaitRegPay := "I" + NextRow
	axisOverallBlockSpaceFull := "J" + NextRow
	var FormulaString string

	err = f.SetCellValue("OverallEvaluation", axisRound, bz.MCRoundNumber)
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}
	FormulaString = "=SUM(RoundTable!C2:C" + NextRow + ")"
	err = f.SetCellFormula("OverallEvaluation", axisBCSize, FormulaString)
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}
	FormulaString = "=SUM(RoundTable!E2:E" + NextRow + ")"
	err = f.SetCellFormula("OverallEvaluation", axisOverallRegPayTX, FormulaString)
	if err != nil {
		log.Lvl2(err)
	}
	FormulaString = "=SUM(RoundTable!F2:F" + NextRow + ")"
	err = f.SetCellFormula("OverallEvaluation", axisOverallPoRTX, FormulaString)
	if err != nil {
		log.Lvl2(err)
	}
	FormulaString = "=SUM(RoundTable!G2:G" + NextRow + ")"
	err = f.SetCellFormula("OverallEvaluation", axisOverallStorjPayTX, FormulaString)
	if err != nil {
		log.Lvl2(err)
	}
	FormulaString = "=SUM(RoundTable!H2:H" + NextRow + ")"
	err = f.SetCellFormula("OverallEvaluation", axisOverallCntPropTX, FormulaString)
	if err != nil {
		log.Lvl2(err)
	}
	FormulaString = "=SUM(RoundTable!I2:I" + NextRow + ")"
	err = f.SetCellFormula("OverallEvaluation", axisOverallCntComTX, FormulaString)
	if err != nil {
		log.Lvl2(err)
	}
	FormulaString = "=AVERAGE(RoundTable!L2:L" + NextRow + ")"
	err = f.SetCellFormula("OverallEvaluation", axisAveWaitOtherTx, FormulaString)
	if err != nil {
		log.Lvl2(err)
	}
	FormulaString = "=AVERAGE(RoundTable!M2:M" + NextRow + ")"
	err = f.SetCellFormula("OverallEvaluation", axisOveralAveWaitRegPay, FormulaString)
	if err != nil {
		log.Lvl2(err)
	}
	FormulaString = "=SUM(RoundTable!O2:O" + NextRow + ")"
	err = f.SetCellFormula("OverallEvaluation", axisOverallBlockSpaceFull, FormulaString)
	if err != nil {
		log.Lvl2(err)
	}

	// ----
	err = f.SaveAs("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/mainchainbc.xlsx")
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	} else {
		log.Lvl3("closing bc")
		log.Lvl2(bz.Name(), " Finished taking transactions from queue (FIFO) into new block in round number ", bz.MCRoundNumber)
	}
}

// -------------------------------------- timeouts -------------------------------------- //

/* ------------------------------------------------------------------------
this function will be called just by ROOT NODE when:
- at the end of a round duration (when no "I am a leader message is recieved): startTimer starts the timer to detect the rounds that dont have any leader elected (if any)
and publish an empty block in those rounds
------------------------------------------------------------------------ */
func (bz *BaseDFS) startTimer(MCRoundNumber int) {
	select {
	case <-time.After(time.Duration(bz.MCRoundDuration) * time.Second):
		// bz.IsRoot() is here just to make sure!,
		// bz.MCRoundNumber == MCRoundNumber is for when the round number has changed!,
		// bz.hasLeader is for when the round number has'nt changed but the leader has been announced
		if bz.IsRoot() && bz.MCRoundNumber == MCRoundNumber && !bz.HasLeader {
			log.Lvl2("No leader for round number ", bz.MCRoundNumber, "an empty block is added")
			bz.NewLeaderChan <- NewLeaderChan{bz.TreeNode(), NewLeader{ /*Leaderinfo: bz.TreeNode(), */ MCRoundNumber: bz.MCRoundNumber}}
		}
	}
}

/* ----------------------------------------------------------------------
	this function will be called just by ROOT NODE at the end of timeout for a simulation run
	(the time specified in config file for a protocol simulation run)
 ----------------------------------------------------------------------*/
func (bz *BaseDFS) StartTimeOut() {
	select {
	case <-time.After(time.Duration(bz.ProtocolTimeout)):
		log.Lvl1(bz.Info(), "protocol ends with timed out: ", bz.ProtocolTimeout)
		bz.DoneBaseDFS <- true
	}
}

// ------------   Sortition Algorithm from ALgorand: ---------------------
// ⟨hash,π⟩←VRFsk(seed||role)
// p←τ/W
// j←0
// while hash/ 2^hashleng </ [ sigma(k=0,j) (B(k,w,p),  sigma(k=0,j+1) (B(k,w,p)] do
//	----	 j++
// return <hash,π, j>
// ----------------------------------------------------------------------

/* ----------------------------------------------------------------------
//checkLeadership
------------------------------------------------------------------------ */

// func (bz *BaseDFS) checkLeadership() {
// 	if bz.leaders[int(math.Mod(float64(bz.MCRoundNumber), float64(len(bz.Roster().List))))] == bz.ServerIdentity().String() {
// 		bz.LeaderPropose <- true
// 	}
// }
/* func (bz *BaseDFS) checkLeadership(power uint64, seed string) {
	for {
		select {
		case <-bz.NewLeaderChan:
			return
		default:

			//power := 1
			//var seed []byte
			var vrfOutput [64]byte

			toBeHashed := []byte(seed)
			proof, ok := bz.ECPrivateKey.ProveBytes(toBeHashed[:])
			if !ok {
				log.Lvl2("error while generating proof")
			}
			_, vrfOutput = bz.ECPrivateKey.Pubkey().VerifyBytes(proof, toBeHashed[:])

			//------------------------- working with big int ----------------------
			//"math/big" imported
			//func generateRandomValuesBigInt(nodes int) [] *big.Int {
			//	var bigIntlist [] *big.Int
			//	for i := 1; i<= nodes;i++{
			//		bigIntlist = append(bigIntlist, new(big.Int).Rand(rand.New(rand.NewSource(time.Now().UnixNano())),<some big int value> ))
			//	}
			//	return bigIntlist
			//}
			// --------------------------------------------------------------------
			//For future refrence: the next commented line is a wrong way to convert a big number to int - it wont raise overflow but the result is incorrect
			//t := binary.BigEndian.Uint64(vrfOutput[:])
			//--------------------------------------------------
			//For future refrence: the next commented line is converting a big number to big int (built in type in go)
			//var bi *big.Int
			//bi = new(big.Int).SetBytes(vrfOutput[:])
			//--------------------------------------------------

			var vrfoutputInt64 uint64
// 			buf := bytes.NewReader(vrfOutput[:])
// 			err := binary.Read(buf, binary.LittleEndian, &vrfoutputInt64)
// 			if err != nil {
// 				log.Lvl2("Panic Raised:\n\n")
// 				panic(err)
// 			}
// 			// I want to ask the previous leader to find next leader based on its output and announce it to her via next round msg
// 			if vrfoutputInt64 < power {
// 				// let other know i am the leader
// 				for _, b := range bz.Tree().List() {
// 					err := bz.SendTo(b, &NewLeader{})
// 					if err != nil {
// 						log.Lvl2(bz.Info(), "can't send new round msg to", b.Name())
// 					}
// 				}
// 				bz.LeaderProposeChan <- true
// 			}
// 			//else {
// 			// 	log.Lvl2("my power:", power, "is", vrfoutputInt64-power, "less than my vrf output :| ")
// 			// }
// 		}
// 	}
// }
*/

// Testpor
func (bz *BaseDFS) Testpor() {

	sk, pk := por.RandomizedKeyGeneration()
	Tau, pf := por.RandomizedFileStoring(sk, por.GenerateFile(bz.SectorNumber), bz.SectorNumber)
	p := por.CreatePoR(pf, bz.SectorNumber, bz.SimulationSeed)
	d, _ := por.VerifyPoR(pk, Tau, p, bz.SectorNumber, bz.SimulationSeed)
	if !d {
		log.Lvl2(d)
	}
}
