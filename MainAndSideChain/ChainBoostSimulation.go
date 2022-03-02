/*
	Base DFS protocol:

	Types of Messages:
	--------------------------------------------
	1- each server who is elected as leader (few servers in each ) send a msg of &NewLeader{Leaderinfo: bz.Name(), MCRoundNumber: bz.MCRoundNumber} to root node.
	2- each round, the root node send a msg of &NewRound{Seed:  seed, Power: power} to all servers including the target server's power
	3- in the bootstrapping phase, the root node send a message to all servers containing protocol config parameters.
	&HelloChainBoost{
					Timeout:                  bz.timeout,
					PercentageTxPay:          bz.PercentageTxPay,
					MCRoundDuration:            bz.MCRoundDuration,
					MainChainBlockSize:                bz.MainChainBlockSize,
					SideChainBlockSize:			bz.SideChainBlockSize,
					SectorNumber:             bz.SectorNumber,
					NumberOfPayTXsUpperBound: bz.NumberOfPayTXsUpperBound,
					SimulationSeed:			  bz.SimulationSeed,
					nbrSubTrees:			  bz.nbrSubTrees,
					threshold:				  bz.threshold,
					SCRoundDuration:            bz.SCRoundDuration,
					CommitteeWindow:          bz.CommitteeWindow,
					MCRoundPerEpoch:					bz.MCRoundPerEpoch,
					SimState:					bz.SimState,
				}
	4- timeOut

*/
// ToDoRaha: it seems that simstate + some other config params aren't required to be sent to all nodes

package MainAndSideChain

import (
	"time"

	"github.com/chainBoostScale/ChainBoost/onet"
	"github.com/chainBoostScale/ChainBoost/onet/log"
	"github.com/chainBoostScale/ChainBoost/por"
	"github.com/chainBoostScale/ChainBoost/vrf"

	"github.com/chainBoostScale/ChainBoost/MainAndSideChain/BLSCoSi"
	"go.dedis.ch/kyber/v3/pairing"
)

/* ----------------------------------------- TYPES -------------------------------------------------
------------------------------------------------------------------------------------------------  */

// Hello is sent down the tree from the root node, every node who gets it starts the protocol and send it to its children
type HelloChainBoost struct {
	SimulationRounds int
	//---- ToDoRaha: Do i need timeout?
	PercentageTxPay          int
	MCRoundDuration          int
	SideChainBlockSize       int
	MainChainBlockSize       int
	SectorNumber             int
	NumberOfPayTXsUpperBound int
	SimulationSeed           int
	SCRoundDuration          int
	CommitteeWindow          int
	MCRoundPerEpoch          int
	// bls cosi config
	NbrSubTrees int
	Threshold   int
	// simulation
	SimState int
}
type HelloChan struct {
	*onet.TreeNode
	HelloChainBoost
}

//  ChainBoost is the main struct that has required parameters for running both protocols ------------
type ChainBoost struct {
	// the node we are represented-in
	*onet.TreeNodeInstance
	ECPrivateKey vrf.VrfPrivkey
	// channel used to let all servers that the protocol has started
	HelloChan chan HelloChan
	// channel used by each round's leader to let all servers that a new round has come
	MainChainNewRoundChan chan MainChainNewRoundChan
	// channel to let nodes that the next round's leader has been specified
	MainChainNewLeaderChan chan MainChainNewLeaderChan
	// the suite we use
	//  suite network.Suite
	// to match the suit in blscosi
	Suite *pairing.SuiteBn256
	//startBCMeasure *monitor.TimeMeasure
	// onDoneCallback is the callback that will be called at the end of the protocol
	//onDoneCallback func() //ToDoRaha: define this function and call it when you want to finish the protocol + check when should it be called
	// channel to notify when we are done -- when a message is sent through this channel the runsimul.go file will catch it and finish the protocol.
	DoneChainBoost chan bool
	// channel to notify leader elected
	LeaderProposeChan chan bool
	MCRoundNumber     int
	SCRoundNumber     int
	HasLeader         bool
	// --- just root node use these - these are used for delay evaluation
	FirstQueueWait  int
	SecondQueueWait int
	// sc
	SideChainQueueWait int
	// side chain queue wait
	FirstSCQueueWait int
	/* ------------------------------------------------------------------
	     -----  system-wide configurations params from the config file
	   ------------------------------------------------------------------
		these  params get initialized
		for the root node: "after" NewMainAndSideChain call (in func: Simulate in file: runsimul.go)
		for the rest of nodes node: while joining protocol by the HelloChainBoost message
	--------------------------------------------------------------------- */
	PercentageTxPay          int
	MCRoundDuration          int
	MainChainBlockSize       int
	SideChainBlockSize       int
	SectorNumber             int
	NumberOfPayTXsUpperBound int
	// ---
	SimulationRounds int
	SimulationSeed   int
	SimState         int
	// -- blscosi related config params
	NbrSubTrees     int
	Threshold       int
	SCRoundDuration int
	CommitteeWindow int //ToDoRaha: go down
	/* ------------------------------------------------------------------
	 ---------------------------  bls cosi protocol  ---------------
	--------------------------------------------------------------------- */
	BlsCosi                  *BLSCoSi.BlsCosi
	CommitteeNodesTreeNodeID []onet.TreeNodeID
	MCRoundPerEpoch          int
	NextSideChainLeader      onet.TreeNodeID
	// channel used by root node to trigger side chain's leader to run a new round of blscosi for side chain
	RtLSideChainNewRoundChan chan RtLSideChainNewRoundChan
	LtRSideChainNewRoundChan chan LtRSideChainNewRoundChan
	// it is initiated in the start function by root node
	BlsCosiStarted bool
	// -- meta block temp summery
	SummPoRTxs map[int]int // server agreement ID --> number of not summerized submitted PoRs in the meta blocks for this agreement
}

/* ----------------------------------- FUNCTIONS -------------------------------------------------
------------------------------------------------------------------------------------------------  */

/* ----------------------------------------------------------------------
	//Start: starts the protocol by sending hello msg to all nodes
------------------------------------------------------------------------ */
func (bz *ChainBoost) Start() error {
	// update the mainchainbc file with created nodes' information
	bz.finalMainChainBCInitialization()
	bz.BlsCosiStarted = true
	// ------------------------------------------------------------------------------
	// config params are sent from the leader to the other nodes in helloBasedDfs function
	bz.helloChainBoost()

	//------- testing message sending ------
	// err := bz.SendTo(bz.Root(), &HelloChainBoost{MCRoundPerEpoch: 0})
	// if err != nil {
	// 	return err
	// }
	return nil
}

/* ----------------------------------------------------------------------
			 Dispatch listen on the different channels in main chain protocol
------------------------------------------------------------------------ */
func (bz *ChainBoost) Dispatch() error {
	// another dispatch function (DispatchProtocol) is called in runsimul.go that takes care of both chain's protocols
	// if !bz.IsRoot() || (bz.IsRoot() && bz.StartedChainBoost) {
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
		// ******* ALL nodes recieve this message to join the protocol and get the config values set
		// -----------------------------------------------------------------------------
		case msg := <-bz.HelloChan:
			log.Lvl2(bz.TreeNode().Name(), "received Hello/config params from", msg.TreeNode.ServerIdentity.Address)
			bz.PercentageTxPay = msg.PercentageTxPay
			bz.MCRoundDuration = msg.MCRoundDuration
			bz.MainChainBlockSize = msg.MainChainBlockSize
			bz.SideChainBlockSize = msg.SideChainBlockSize
			bz.SectorNumber = msg.SectorNumber
			bz.NumberOfPayTXsUpperBound = msg.NumberOfPayTXsUpperBound
			bz.SimulationSeed = msg.SimulationSeed
			bz.SCRoundDuration = msg.SCRoundDuration
			bz.CommitteeWindow = msg.CommitteeWindow
			bz.MCRoundPerEpoch = msg.MCRoundPerEpoch
			// bls cosi config
			bz.NbrSubTrees = msg.NbrSubTrees
			bz.BlsCosi.Threshold = msg.Threshold
			//bz.BlsCosi.Timeout = msg.ProtocolTimeout
			bz.SimState = msg.SimState
			if msg.NbrSubTrees > 0 {
				err := bz.BlsCosi.SetNbrSubTree(msg.NbrSubTrees)
				if err != nil {
					return err
				}
			}
			bz.helloChainBoost()
		// -----------------------------------------------------------------------------
		// *** MC *** ALL nodes recieve this message to sync rounds
		// -----------------------------------------------------------------------------
		case msg := <-bz.MainChainNewRoundChan:
			bz.MCRoundNumber = bz.MCRoundNumber + 1
			log.Lvl4(bz.Name(), " round number ", bz.MCRoundNumber, " started at ", time.Now().Format(time.RFC3339))
			bz.MainChainCheckLeadership(msg)
		// -----------------------------------------------------------------------------
		// *** MC *** just the ROOT NODE (blockchain layer one) recieve this msg
		// -----------------------------------------------------------------------------
		case msg := <-bz.MainChainNewLeaderChan:
			bz.RootPreNewRound(msg)
		// -----------------------------------------------------------------------------
		// *** SC *** next side chain's leader recieves this message
		// -----------------------------------------------------------------------------
		case msg := <-bz.RtLSideChainNewRoundChan:
			bz.SideChainLeaderPreNewRound(msg)
		// -----------------------------------------------------------------------------
		// *** SC *** just the ROOT NODE (blockchain layer one) recieve this msg
		// -----------------------------------------------------------------------------
		case msg := <-bz.LtRSideChainNewRoundChan:
			bz.RootPostNewRound(msg)
		}
		// running = false //ToDoRaha: do something about running!
	}
	return err
}

/* ----------------------------------------------------------------------
 helloChainBoost
------------------------------------------------------------------------ */

func (bz *ChainBoost) helloChainBoost() {
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
		if bz.SimState == 2 {
			bz.StartSideChainProtocol()
			bz.StartMainChainProtocol()
		} else if bz.SimState == 1 {
			bz.StartMainChainProtocol()
		} else {
			panic("sim state config param is not set correctly")
		}
	}
}

// -------------------------------------- timeouts -------------------------------------- //

/* ------------------------------------------------------------------------
this function will be called just by ROOT NODE when:
- at the end of a round duration (when no "I am a leader message is recieved): startTimer starts the timer to detect the rounds that dont have any leader elected (if any)
and publish an empty block in those rounds
------------------------------------------------------------------------ */
func (bz *ChainBoost) startTimer(MCRoundNumber int) {
	select {
	case <-time.After(time.Duration(bz.MCRoundDuration) * time.Second):
		// bz.IsRoot() is here just to make sure!,
		// bz.MCRoundNumber == MCRoundNumber is for when the round number has changed!,
		// bz.hasLeader is for when the round number has'nt changed but the leader has been announced
		if bz.IsRoot() && bz.MCRoundNumber == MCRoundNumber && !bz.HasLeader {
			log.Lvl2("No leader for round number ", bz.MCRoundNumber, "an empty block is added")
			bz.MainChainNewLeaderChan <- MainChainNewLeaderChan{bz.TreeNode(), NewLeader{ /*Leaderinfo: bz.TreeNode(), */ MCRoundNumber: bz.MCRoundNumber}}
		}
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

// func (bz *ChainBoost) checkLeadership() {
// 	if bz.leaders[int(math.Mod(float64(bz.MCRoundNumber), float64(len(bz.Roster().List))))] == bz.ServerIdentity().String() {
// 		bz.LeaderPropose <- true
// 	}
// }
/* func (bz *ChainBoost) checkLeadership(power uint64, seed string) {
	for {
		select {
		case <-bz.MainChainNewLeaderChan:
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
func (bz *ChainBoost) Testpor() {

	sk, pk := por.RandomizedKeyGeneration()
	Tau, pf := por.RandomizedFileStoring(sk, por.GenerateFile(bz.SectorNumber), bz.SectorNumber)
	p := por.CreatePoR(pf, bz.SectorNumber, bz.SimulationSeed)
	d, _ := por.VerifyPoR(pk, Tau, p, bz.SectorNumber, bz.SimulationSeed)
	if !d {
		log.Lvl2(d)
	}
}
