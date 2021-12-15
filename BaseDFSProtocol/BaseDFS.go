/*
	Base DFS protocol:

	Types of Messages:
	--------------------------------------------
	1- each server who is elected as leader (few servers in each round) send a msg of &NewLeader{Leaderinfo: bz.Name(), RoundNumber: bz.roundNumber} to root node.
	2- each round, the root node send a msg of &NewRound{Seed:  seed, Power: power} to all servers including the target server's power
	3- in the bootstrapping phase, the root node send a message to all servers containing protocol config parameters.
	&HelloBaseDFS{
					Timeout:                  bz.timeout,
					PercentageTxPay:          bz.PercentageTxPay,
					RoundDuration:            bz.RoundDuration,
					BlockSize:                bz.BlockSize,
					SectorNumber:             bz.SectorNumber,
					NumberOfPayTXsUpperBound: bz.NumberOfPayTXsUpperBound,
					SimulationSeed:			  bz.SimulationSeed,
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
	"sync"
	"time"

	onet "github.com/basedfs"
	"github.com/basedfs/blockchain"
	"github.com/basedfs/log"
	"github.com/basedfs/network"
	"github.com/basedfs/por"
	"github.com/basedfs/vrf"
	"github.com/xuri/excelize/v2"
)

/* -------------------------------------------------------------------- */
// ------- TYPES  --------------------------------------------
/* -------------------------------------------------------------------- */
//  -------------------  Transactions and Block Structurse -----------------
/* -------------------------------------------------------------------- */

// ------  types used for communication (msgs) -------------------------
/*Hello is sent down the tree from the root node, every node who gets it starts the protocol and send it to its children*/
type HelloBaseDFS struct {
	ProtocolTimeout time.Duration
	//---- ToDo: Do i need timeout?
	PercentageTxPay          int
	RoundDuration            int
	BlockSize                int
	SectorNumber             int
	NumberOfPayTXsUpperBound int
	SimulationSeed           int
}
type HelloChan struct {
	*onet.TreeNode
	HelloBaseDFS
}

type NewLeader struct {
	Leaderinfo  string
	RoundNumber int
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
	suite network.Suite
	//startBCMeasure *monitor.TimeMeasure
	// onDoneCallback is the callback that will be called at the end of the protocol
	//onDoneCallback func() //ToDo: raha: define this function and call it when you want to finish the protocol + check when should it be called
	// channel to notify when we are done -- when a message is sent through this channel a dispatch function in .. file will catch it and finish the protocol.
	DoneBaseDFS chan bool //it is not initiated in new proto!
	// channel to notify leader elected
	LeaderProposeChan chan bool
	// ------------------------------------------------------------------------------------------------------------------
	//  -----  system-wide configurations params from the config file
	// ------------------------------------------------------------------
	//these  params get initialized
	//for the root node: "after" NewBaseDFSProtocol call (in func: Simulate in file: runsimul.go)
	//for the rest of nodes node: while joining protocol by the HelloBaseDFS message
	PercentageTxPay          int
	RoundDuration            int
	BlockSize                int
	SectorNumber             int
	NumberOfPayTXsUpperBound int
	// ---
	ProtocolTimeout time.Duration
	timeoutMu       sync.Mutex
	SimulationSeed  int
	// ------------------------------------------------------------------
	roundNumber int
	hasLeader   bool
	// --- just root node use these
	FirstQueueWait  int
	SecondQueueWait int
	// ------------------------------------------------------------------------------------------------------------------
	//ToDo: raha: dol I need these items?
	//vcMeasure *monitor.TimeMeasure
	//lock associated
	//doneLock  sync.Mutex
}

/* -------------------------------------------------------------------- */
// ------- FUNCTIONS  -------------------------------------------------
/* -------------------------------------------------------------------- */
func init() {
	//network.RegisterMessage(ProofOfRetTxChan{})
	//network.RegisterMessage(PreparedBlockChan{})
	network.RegisterMessage(HelloBaseDFS{})
	network.RegisterMessage(NewRound{})
	network.RegisterMessage(NewLeader{})
	onet.GlobalProtocolRegister("BaseDFS", NewBaseDFSProtocol)
}

/* ----------------------------------------------------------------------
------------------------------------------------------------------------ */
// NewBaseDFSProtocol returns a new BaseDFS struct
func NewBaseDFSProtocol(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
	bz := &BaseDFS{
		TreeNodeInstance:  n,
		suite:             n.Suite(),
		DoneBaseDFS:       make(chan bool, 1),
		LeaderProposeChan: make(chan bool, 1),
		roundNumber:       1,
		hasLeader:         false,
		FirstQueueWait:    0,
		SecondQueueWait:   0,
		ProtocolTimeout:   0,
	}
	if err := n.RegisterChannel(&bz.HelloChan); err != nil {
		return bz, err
	}
	if err := n.RegisterChannel(&bz.NewRoundChan); err != nil {
		return bz, err
	}
	// if err := n.RegisterChannel(&bz.NewLeaderChan); err != nil {
	// 	return bz, err
	// }
	if err := n.RegisterChannelLength(&bz.NewLeaderChan, len(bz.Tree().List())); err != nil {
		log.Error("Couldn't reister channel:", err)
	}
	// bls key pair for each node for VRF
	_, bz.ECPrivateKey = vrf.VrfKeygen()
	//--------------- These msgs were used for communication-----------
	/*
		bz.viewChangeThreshold = int(math.Ceil(float64(len(bz.Tree().List())) * 2.0 / 3.0))
		// --------   register channels
		if err := n.RegisterChannel(&bz.PreparedBlockChan); err != nil {
			return bz, err
		}
		if err := n.RegisterChannel(&bz.ProofOfRetTxChan); err != nil {
			return bz, err
		}
		if err := n.RegisterChannel(&bz.viewchangeChan); err != nil {
			return bz, err
		}
	*/
	//---------------------------------------------------------------------
	return bz, nil
}

/* ----------------------------------------------------------------------
------------------------------------------------------------------------ */
//Start: starts the protocol by sending hello msg to all nodes
func (bz *BaseDFS) Start() error {
	// update the centralbc file with created nodes' information
	bz.finalCentralBCInitialization()

	//bz.Testpor()
	//vrf.Testvrf()

	// config params are sent from the leader to the other nodes in helloBasedDfs function
	bz.helloBaseDFS()
	return nil
}

/* ----------------------------------------------------------------------
------------------------------------------------------------------------ */
//finalCentralBCInitialization initialize the central bc file based on the config params defined in the config file (.toml file of the protocol)
// the info we hadn't before and we have now is nodes' info that this function add to the centralbc file
func (bz *BaseDFS) finalCentralBCInitialization() {
	var NodeInfoRow []string
	for _, a := range bz.Roster().List {
		NodeInfoRow = append(NodeInfoRow, a.String())
	}
	var err error
	f, err := excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
	if err != nil {
		log.Lvl2("Raha: ", err)
		panic(err)
	} else {
		log.Lvl2("opening bc")
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

	err = f.SaveAs("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	} else {
		log.Lvl2("closing bc")
	}
}

/* ----------------------------------------------------------------------
------------------------------------------------------------------------ */
// Dispatch listen on the different channels
func (bz *BaseDFS) Dispatch() error {
	running := true
	var err error

	for running {
		select {
		case msg := <-bz.HelloChan:
			log.Lvl2(bz.TreeNode().Name(), "received Hello/config params from", msg.TreeNode.ServerIdentity.Address)
			bz.PercentageTxPay = msg.PercentageTxPay
			bz.RoundDuration = msg.RoundDuration
			bz.BlockSize = msg.BlockSize
			bz.SectorNumber = msg.SectorNumber
			bz.NumberOfPayTXsUpperBound = msg.NumberOfPayTXsUpperBound
			bz.SimulationSeed = msg.SimulationSeed
			bz.helloBaseDFS()

		// this msg is catched in simulation codes
		case <-bz.DoneBaseDFS:
			running = false
		case msg := <-bz.NewRoundChan:
			bz.roundNumber = bz.roundNumber + 1
			log.Lvl2(bz.Name(), " round number ", bz.roundNumber, " started at ", time.Now().Format(time.RFC3339))
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
				log.Lvl2("Panic Raised:\n\n")
				panic(err)
			}
			// -----------
			// the criteria for selecting the leader
			if vrfoutputInt64 < msg.Power {
				// -----------
				log.Lvl2(bz.Name(), "I may be elected for round number ", bz.roundNumber)
				go bz.SendTo(bz.Root(), &NewLeader{Leaderinfo: bz.Name(), RoundNumber: bz.roundNumber})
			}

			// ******* just the ROOT NODE (blockchain layer one) recieve this msg
		case msg := <-bz.NewLeaderChan:

			// -----------------------------------------------------
			// rounds without a leader: in this case the leader info is filled with root node's info, transactions are going to be collected normally but
			// since the block is empty, no transaction is going to be taken from queues => leader = false
			// -----------------------------------------------------

			if msg.Leaderinfo == bz.Name() && bz.roundNumber != 1 && !bz.hasLeader && msg.RoundNumber == bz.roundNumber {
				bz.hasLeader = true
				bz.updateBCPowerRound(msg.Leaderinfo, false)
				// in the case of a leader-less round
				log.Lvl1("final round result: ", msg.Leaderinfo, "(root node) filled round number", bz.roundNumber, "with empty block")
				bz.updateBCTransactionQueueCollect()
				bz.readBCAndSendtoOthers()
				log.Lvl2("new round is announced")
				continue
			}

			// -----------------------------------------------------
			// normal rounds with a leader => leader = true
			// -----------------------------------------------------

			if !bz.hasLeader && msg.RoundNumber == bz.roundNumber {
				// first validate the leadership proof
				log.Lvl1("final round result: ", msg.Leaderinfo, "is the round leader for round number ", bz.roundNumber)
				bz.hasLeader = true
				bz.updateBCPowerRound(msg.Leaderinfo, true)
				bz.updateBCTransactionQueueCollect()
				bz.updateBCTransactionQueueTake()
			} else {
				log.Lvl2("this round already has a leader!")
			}
			//waiting for the time of round duration
			time.Sleep(time.Duration(bz.RoundDuration) * time.Second)
			// empty list of elected leaders  in this round
			for len(bz.NewLeaderChan) > 0 {
				<-bz.NewLeaderChan
			}
			// announce new round and give away required checkleadership info to nodes
			bz.readBCAndSendtoOthers()
			log.Lvl2("new round is announced")
		}
	}
	// -------- cases used for communication ------------------
	/*		case msg := <-bz.ProofOfRetTxChan:
			//log.Lvl2(bz.Info(), "received por", msg.Por.sigma, "tx from", msg.TreeNode.ServerIdentity.Address)
			if !fail {
				err = bz.handlePoRTx(msg)
			}*/
	/*		case msg := <-bz.PreparedBlockChan:
			log.Lvl2(bz.Info(), "received block from", msg.TreeNode.ServerIdentity.Address)
			if !fail {
				_, err = bz.handleBlock(msg)
	}*/
	// ------------------------------------------------------------------
	return err
}

/* each round THE ROOT NODE send a msg to all nodes,
let other nodes know that the new round has started and the information they need
from blockchain to check if they are next round's leader */
func (bz *BaseDFS) readBCAndSendtoOthers() {
	powers, seed := bz.readBCPowersAndSeed()
	bz.roundNumber = bz.roundNumber + 1
	bz.hasLeader = false
	for _, b := range bz.Tree().List() {
		power, found := powers[b.ServerIdentity.String()]
		if found && !b.IsRoot() {
			err := bz.SendTo(b, &NewRound{
				Seed:  seed,
				Power: power})
			if err != nil {
				log.Lvl2(bz.Info(), "can't send new round msg to", b.Name())
			}
		}
	}
	// detecting leader-less in next round
	go bz.startTimer(bz.roundNumber)
}

/* ----------------------------------------------------------------------*/
func (bz *BaseDFS) readBCPowersAndSeed() (powers map[string]uint64, seed string) {
	minerspowers := make(map[string]uint64)
	f, err := excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
	if err != nil {
		log.Lvl2("Raha: ", err)
		panic(err)
	} else {
		log.Lvl2("opening bc")
	}
	//var err error
	var rows *excelize.Rows
	var row []string
	rowNumber := 0
	// looking for last round's seed in the round table sheet in the centralbc file
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
		// 	if bz.roundNumber, err = strconv.Atoi(colCell); err != nil {
		// 		log.Lvl2("Panic Raised:\n\n")
		// 		panic(err)
		// 	}
		// }
		if i == 1 {
			seed = colCell // last round's seed
		}
	}
	// looking for all nodes' power in the last round in the power table sheet in the centralbc file
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
------------------------------------------------------------------------ */
//helloBaseDFS
func (bz *BaseDFS) helloBaseDFS() {
	log.Lvl2(bz.TreeNode().Name(), " joined to the protocol")
	//bz.startBCMeasure = monitor.NewTimeMeasure("viewchange") //ToDo: raha: check monitor.measure
	// ----------------------------------------
	if !bz.IsLeaf() {
		for _, child := range bz.Children() {
			go func(c *onet.TreeNode) {
				err := bz.SendTo(c, &HelloBaseDFS{
					ProtocolTimeout:          bz.ProtocolTimeout,
					PercentageTxPay:          bz.PercentageTxPay,
					RoundDuration:            bz.RoundDuration,
					BlockSize:                bz.BlockSize,
					SectorNumber:             bz.SectorNumber,
					NumberOfPayTXsUpperBound: bz.NumberOfPayTXsUpperBound,
					SimulationSeed:           bz.SimulationSeed,
				})
				if err != nil {
					log.Lvl2(bz.Info(), "couldn't send hello to child", c.Name())
				}
			}(child)
		}
	} //ToDo: later check what happens to these go routines!

	// the root node is filling the first block in first round
	if bz.TreeNodeInstance.IsRoot() {
		// let other nodes join the protocol first -
		time.Sleep(time.Duration(len(bz.Roster().List)) * time.Second)

		log.Lvl2(bz.Name(), "Filling round number ", bz.roundNumber)
		// for the first round we have the root node set as a round leader, so  it is true! and he takes txs from the queue
		bz.updateBCPowerRound(bz.Name(), true)
		bz.updateBCTransactionQueueCollect()
		bz.updateBCTransactionQueueTake()
		time.Sleep(time.Duration(bz.RoundDuration) * time.Second)
		bz.readBCAndSendtoOthers()
		bz.StartTimeOut()
	}
}

// -- the time specified in config file for a protocol simulation run
// --- this function will be called by root node at the end of timeout for a simulation run
func (bz *BaseDFS) StartTimeOut() {
	select {
	case <-time.After(time.Duration(bz.ProtocolTimeout)):
		log.Lvl1(bz.Info(), "protocol ends with timed out: ", bz.ProtocolTimeout)
		bz.DoneBaseDFS <- true
	}
}

/* ---------------------------------------------------------------------- */
//updateBC: by leader
// each round, adding one row in power table based on the information in market matching sheet,
// assuming that servers are honest and have honestly publish por for their actice (not expired) contracts,
// for each storage server and each of their active contract, add the stored file size to their current power
/* ----------------------------------------------------------------------*/
func (bz *BaseDFS) updateBCPowerRound(Leaderinfo string, leader bool) {
	var err error
	var rows *excelize.Rows
	var row []string
	var seed string
	rowNumber := 0

	f, err := excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
	if err != nil {
		log.Lvl2("Raha: ", err)
		panic(err)
	} else {
		log.Lvl2(bz.Name(), "opening bc")
	}

	// looking for last round's seed in the round table sheet in the centralbc file
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
		//if i == 0 {if bz.roundNumber,err = strconv.Atoi(colCell); err!=nil {log.Lvl2(err)}}  // i dont want to change the round number now, even if it is changed, i will re-check it at the end!
		if i == 1 {
			seed = colCell
		} // next round's seed is the hash of this seed
	}
	// --------------------------------------------------------------------
	// updating the current last row in the "BCsize" column

	// no longer nodes read from centralbc file, instead the round are determined by their local round number variable
	currentRow := strconv.Itoa(rowNumber)
	nextRow := strconv.Itoa(rowNumber + 1) //ToDo: raha: remove these, use bz.roundnumber instead!
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
	err = f.SetCellValue("RoundTable", axisMiner, Leaderinfo)
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}
	// --------------------------------------------------------------------
	// adding one row in round table (round number and seed columns)
	axisRoundNumber := "A" + nextRow
	err = f.SetCellValue("RoundTable", axisRoundNumber, bz.roundNumber+1)
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
	var ContractDuration, ContractStartedRoundNumber, FileSize int
	var MinerServer string
	rowNum := 0
	MinerServers := make(map[string]int)
	for rows.Next() {
		rowNum++
		if rowNum == 1 {
			row, err = rows.Columns()
		} else {
			row, err = rows.Columns()
			if err != nil {
				log.Lvl2("Panic Raised:\n\n")
				panic(err)
			} else {
				for i, colCell := range row {
					/* --- in MarketMatching: i = 0 is Server's Info,
					i = 1 is FileSize, i=2 is ContractDuration,
					i=3 is RoundNumber, i=4 is ContractID, i=5 is Client's PK,
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
						ContractStartedRoundNumber, err = strconv.Atoi(colCell)
						if err != nil {
							log.Lvl2("Panic Raised:\n\n")
							panic(err)
						}
					}
				}
			}
			if bz.roundNumber-ContractStartedRoundNumber <= ContractDuration {
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
	axisRoundNumber = "B" + currentRow
	err = f.SetSheetRow("PowerTable", axisRoundNumber, &PowerInfoRow)
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}
	// --------------------------------------------------------------------
	// adding current round's round number
	axisRoundNumber = "A" + currentRow
	err = f.SetCellValue("PowerTable", axisRoundNumber, bz.roundNumber)
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}
	// ----
	err = f.SaveAs("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	} else {
		log.Lvl2("closing bc")
	}
}

/* ----------------------------------------------------------------------
------------------------------------------------------------------------ */
//updateBC: by leader
func (bz *BaseDFS) updateBCTransactionQueueCollect() {
	var err error
	var rows *excelize.Rows
	var row []string

	f, err := excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
	if err != nil {
		log.Lvl2("Raha: ", err)
		panic(err)
	} else {
		log.Lvl2("opening bc")
	}

	//	each round, adding one row in power table based on the information in market matching sheet,assuming that servers are honest  and have honestly publish por for their actice (not expired) contracts,for each storage server and each of their active contracst, add the stored file size to their current power
	if rows, err = f.Rows("MarketMatching"); err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}
	var ContractDuration, ContractStartedRoundNumber, FileSize, ContractPublished, ContractID int
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
					i=3 is RoundNumber,
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
						ContractStartedRoundNumber, err = strconv.Atoi(colCell)
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
			} else if bz.roundNumber-ContractStartedRoundNumber <= ContractDuration { // contract is not expired
				t[0] = FileSize //if each server one contract
				// Add TxPor
				t[4] = 1
				transactionQueue[MinerServer] = t
			} else if bz.roundNumber-ContractStartedRoundNumber > ContractDuration {
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
	3) issuedRoundNumber
	4) contractId */
	var newTransactionRow [5]string
	s := make([]interface{}, len(newTransactionRow)) //todo: raha:  check this out later: https://stackoverflow.com/questions/23148812/whats-the-meaning-of-interface/23148998#23148998

	newTransactionRow[2] = time.Now().Format(time.RFC3339)
	newTransactionRow[3] = strconv.Itoa(bz.roundNumber)
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
					log.Lvl2("a TxPor added to queue in round number", bz.roundNumber)
				} else if newTransactionRow[0] == "TxStoragePayment" {
					log.Lvl2("a TxStoragePayment added to queue in round number", bz.roundNumber)
				} else if newTransactionRow[0] == "TxContractPropose" {
					log.Lvl2("a TxContractPropose added to queue in round number", bz.roundNumber)
				}
			}
		}
		/* second row added in case of having the first row to be contract propose tx which then we will add
		contract commit tx right away.
		warning: Just in one case it may cause irrational statistics which doesn’t worth taking care of!
		when a propose contract tx is added to a block which causes the contract to become active but
		the commit contract transaction is not yet! */
		if addCommitTx == true {
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
					log.Lvl2("a TxContractCommit added to queue in round number", bz.roundNumber)
				}
			}
		}
	}
	// ----------------------------------------------------------------------
	// ------ add payment transactions into transaction queue payment sheet
	// ----------------------------------------------------------------------
	/* each transaction has the following column stored on the transaction queue payment sheet:
	0) size
	1) time
	2) issuedRoundNumber */

	newTransactionRow[0] = strconv.Itoa(PayTxSize)
	newTransactionRow[1] = time.Now().Format(time.RFC3339)
	newTransactionRow[2] = strconv.Itoa(bz.roundNumber)
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
	log.Lvl2("Number of regular payment transactions in round number", bz.roundNumber, "is", numberOfRegPay)
	for i := 1; i <= numberOfRegPay; i++ {
		if err = f.InsertRow("SecondQueue", 2); err != nil {
			log.Lvl2("Panic Raised:\n\n")
			panic(err)
		} else {
			if err = f.SetSheetRow("SecondQueue", "A2", &s); err != nil {
				log.Lvl2("Panic Raised:\n\n")
				panic(err)
			} else {
				log.Lvl2("a regular payment transaction added to queue in round number", bz.roundNumber)
			}
		}
	}
	// ---
	err = f.SaveAs("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	} else {
		log.Lvl2("closing bc")
		log.Lvl2(bz.Name(), " finished collecting new transactions to queue in round number ", bz.roundNumber)
	}
}

/* ----------------------------------------------------------------------
------------------------------------------------------------------------ */
//updateBC: by leader
func (bz *BaseDFS) updateBCTransactionQueueTake() {
	var err error
	var rows [][]string
	// --- reset
	bz.FirstQueueWait = 0
	bz.SecondQueueWait = 0

	f, err := excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
	if err != nil {
		log.Lvl2("Raha: ", err)
		panic(err)
	} else {
		log.Lvl2("opening bc")
	}

	var accumulatedTxSize, txsize int
	blockIsFull := false
	NextRow := strconv.Itoa(bz.roundNumber + 2)

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
		2) issuedRoundNumber */

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
					log.Lvl2("a regular payment transaction added to block number", bz.roundNumber, " from the queue")
					// row[1] is transaction's collected time
					if TakeTime, err = time.Parse(time.RFC3339, row[1]); err != nil {
						log.Lvl2("Panic Raised:\n\n")
						panic(err)
					}
					bz.SecondQueueWait = bz.SecondQueueWait + int(time.Now().Sub(TakeTime).Seconds())
					f.RemoveRow("SecondQueue", i)
				} else {
					blockIsFull = true
					log.Lvl1("final: regular  payment share is full!")
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
		3) issuedRoundNumber
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
								err = f.SetCellValue("MarketMatching", startRoundCellMarketMatching, bz.roundNumber)
								if err != nil {
									log.Lvl2("Panic Raised:\n\n")
									panic(err)
								} else {
									log.Lvl2("a TxContractCommit tx added to block number", bz.roundNumber, " from the queue")
									numberOfContractCommitTx++
								}
							}
						}
					} else if row[0] == "TxStoragePayment" {
						log.Lvl2("a TxStoragePayment tx added to block number", bz.roundNumber, " from the queue")
						numberOfStoragePayTx++
					} else if row[0] == "TxPor" {
						log.Lvl2("a por tx added to block number", bz.roundNumber, " from the queue")
						numberOfPoRTx++
					} else if row[0] == "TxContractPropose" {
						log.Lvl2("a TxContractPropose tx added to block number", bz.roundNumber, " from the queue")
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
					log.Lvl1("final: block is full! ")
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

	log.Lvl1("In total in round number ", bz.roundNumber,
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

	log.Lvl1("final: \n", allocatedBlockSizeForRegPayTx,
		" for regular payment txs,\n and ", accumulatedTxSize, " for other types of txs")
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}

	log.Lvl1("In total in round number ", bz.roundNumber,
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

	err = f.SetCellValue("OverallEvaluation", axisRound, bz.roundNumber)
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
	err = f.SaveAs("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	} else {
		log.Lvl2("closing bc")
		log.Lvl2(bz.Name(), " Finished taking transactions from queue (FIFO) into new block in round number ", bz.roundNumber)
	}
}

/* ------------------------------------------------------------------------
startTimer starts the timer to detect the rounds that dont have any leader elected (if any)
and publish an empty block in those rounds
------------------------------------------------------------------------ */
func (bz *BaseDFS) startTimer(roundNumber int) {
	select {
	case <-time.After(time.Duration(bz.RoundDuration) * time.Second):
		if bz.IsRoot() && bz.roundNumber == roundNumber && !bz.hasLeader {
			log.Lvl2("No leader for round number ", bz.roundNumber, "an empty block is added")
			bz.NewLeaderChan <- NewLeaderChan{bz.TreeNode(), NewLeader{Leaderinfo: bz.Name(), RoundNumber: bz.roundNumber}}
		}
	}
}

/* ----------------------------------------------------------------------
------------------------------------------------------------------------ */
// SetTimeout sets the new timeout
//todo: raha:  do we need the following functions?
// func (bz *BaseDFS) SetTimeout(t time.Duration) {
// 	bz.timeoutMu.Lock()
// 	bz.Timeout = t
// 	bz.timeoutMu.Unlock()
// }

/* ----------------------------------------------------------------------
-----------------
------------------------------------------------------------------------ */
// Timeout returns the current timeout
// func (bz *BaseDFS) Timeout() time.Duration {
// 	bz.timeoutMu.Lock()
// 	defer bz.timeoutMu.Unlock()
// 	return bz.timeout
// }

// ------------   Sortition Algorithm from ALgorand: ---------------------
// ⟨hash,π⟩←VRFsk(seed||role)
// p←τ/W
// j←0
// while hash/ 2^hashleng </ [ sigma(k=0,j) (B(k,w,p),  sigma(k=0,j+1) (B(k,w,p)] do
//	----	 j++
// return <hash,π, j>
// ----------------------------------------------------------------------

//structures
/* ---------------------------------------------------------------------
---------------- these channel were used for communication----------
   ---------------------------------------------------------------------
type ProofOfRetTxChan struct {
	*onet.TreeNode
	por.Por
}

type PreparedBlockChan struct {
	*onet.TreeNode
	PreparedBlock
}
-------------------------------------------------------------------- */

//Functions
/* ----------------------------------------------------------------------
------------------------------------------------------------------------ */
// test file access : central bc file from inside the protocol
/*func (bz *BaseDFS) readwriteBC() () {
	//---- test file read / write access
	data, err := ioutil.ReadFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
	if err != nil {
		log.Lvl2(bz.Info(), "error reading from centralbc", err)
	} else {
		log.Lvl2(bz.Info(), "This is the file content:", string(data))
	}
	//----------------
	f, err := excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/simple.xlsx")
	if err != nil {
		log.Lvl2(err)
	}
	c1, err := f.GetCellValue("Sheet1", "A1")
	if err != nil {
		log.Lvl2(err)
	}
	log.Lvl2(c1)
	c2, err := f.GetCellValue("Sheet1", "A4")
	if err != nil {
		log.Lvl2(err)
	}
	log.Lvl2(c2)
	c3, err := f.GetCellValue("Sheet1", "B2")
	result, err := f.SearchSheet("Sheet1", "100")
	if err != nil {
		log.Lvl2(err)
	}
	log.Lvl2(c3)
	log.Lvl2(result)
	f.SetCellValue("Sheet1", "A3", 42)
	if err := f.SaveAs("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/simple.xlsx"); err != nil {
		log.Lvl2(err)
	}
	log.Lvl2("wait")
	// --------------------------- write -----
	//---- test file read / write access
	f, err := os.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx", os.O_APPEND|os.O_WRONLY, 0600)
	defer f.Close()
	_, err = f.WriteString(bz.TreeNode().String() + "\n")
	w := bufio.NewWriter(f)
	_, err = fmt.Fprintf(w, "%v\n", 10)
	_, err = fmt.Fprintf(w, "%v\n", "hi")
	w.Flush()
	data, err := ioutil.ReadFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
	if err!=nil{
		log.Lvl2(bz.Info(), "error reading from centralbc", err)
	} else {
		log.Lvl2(bz.Info(), "This is the file content which leader see:" , string(data))
	}
	//----------------
	_, err := ioutil.ReadFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.csv")
	check(err)
	log.Lvl2(RoundDuration)
	//assert.Equal(t, 7, len(strings.Split(string(csv), "\n")))
	f, err := os.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.csv", os.O_APPEND|os.O_WRONLY, 0600)
	check(err)
	defer f.Close()
	_, err = f.WriteString("Header\n")
	check(err)

	data, err := ioutil.ReadFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.csv")
	check(err)
	fmt.Print(string(data))
	//--------------------
	f, err = os.Create("/Users/raha/Documents/GitHub/basedfs/simul/platform/testfileaccess.csv")
	check(err)
	defer f.Close()
	//--------------------
	w := bufio.NewWriter(f)
	//choose random number for recipe
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	i := r.Perm(5)
	_, err = fmt.Fprintf(w, "%v\n", "RoundDuration")
	_, err = fmt.Fprintf(w, "%v\n", RoundDuration)

	check(err)
	w.Flush()
}*/
//sendPoRTx send a por tx
/*func (bz *BaseDFS) sendPoRTx() {
	txs := bz.createPoRTx()
	portx := &Por{*txs, uint32(bz.Index())}
	log.Lvl2(bz.Name(), ": multicasting por tx")
	bz.Multicast(portx, bz.List()...)

	for _, child := range bz.Children() {
		err := bz.SendTo(child, portx)
		if err != nil {
			log.Lvl1(bz.Info(), "couldn't send to child:", child.ServerIdentity.Address)
		} else {
			log.Lvl1(bz.Info(), "sent his PoR to his children:", child.ServerIdentity.Address)
		}
	}
}*/
// handleAnnouncement pass the announcement to the right CoSi struct.
/*func (bz *BaseDFS) handlePoRTx(proofOfRet ProofOfRetTxChan) error {
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
}*/
//appendPoRTx: append the recieved tx to his current temporary block
/*func (bz *BaseDFS) appendPoRTx(p ProofOfRetTxChan) error {
	bz.transactions = append(bz.transactions, p.por.Tx)
	for test creating block:
	bz.createEpochBlock(&leadershipProof{1})
	return nil
}*/
//SendFinalBlock   bz.SendFinalBlock(createEpochBlock)
/*func (bz *BaseDFS) SendFinalBlock(fb *blockchain.TrBlock) {
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
}*/
// verifyBlock: servers will verify proposed block when they recieve it
/*func verifyBlock(pb PreparedBlock) error {
	return nil
}*/
//appendBlock:
/*func (bz *BaseDFS) appendBlock(pb PreparedBlock) (*blockchain.TrBlock, error) {
	//measure block's size
	return nil, nil
}*/
// handle the arrival of a block
/*func (bz *BaseDFS) handleBlock(pb PreparedBlockChan) (*blockchain.TrBlock, error) {
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
}*/

/* ----------------------------------------------------------------------
------------------------------------------------------------------------ */
/*
func (bz *BaseDFS) readBCForNewRound(f *excelize.File) bool {
	var err error
	var rows *excelize.Rows
	var row []string
	rowNumber := 0
	var lastRound int
	// looking for last round's seed in the round table sheet in the centralbc file
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
		if i == 0 {
			lastRound, err = strconv.Atoi(colCell)
			if err != nil {
				log.Lvl2("Panic Raised:\n\n")
				panic(err)
			}
		}
	}
	if lastRound != bz.roundNumber {
		return true // a new round has been started
	} else {
		return false
	}
}
*/

/* ----------------------------------------------------------------------
------------------------------------------------------------------------ */

/* ----------------------------------------------------------------------——
------------------------------------------------------------------------ */
/*
func (bz *BaseDFS) refreshBC() {
	log.Lvl2(bz.Name(), "Lock: locked BC")
	//bz.fMu.Lock()
	//defer bz.fMu.Unlock()
	// ---
	f2, err := excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
	failedAttempts := 0
	if err != nil {
		for err != nil {
			if failedAttempts == 11 {
				log.Lvl2("Can't open the centralbc file: 10 attempts!")
			}
			time.Sleep(1 * time.Second)
			f2, err = excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
			failedAttempts = failedAttempts + 1
		}
		log.Lvl2("---------- Opened the centralbc file after ", failedAttempts, " attempts")
	} else {
		log.Lvl2(bz.Name(), "opened BC")
	}

	if !bz.isLeader {
		bz.f = f2
		if bz.readBCForNewRound(bz.f) {
			bz.checkLeadership()
		}

	} else {
		if !bz.readBCForNewRound(f2) {
			log.Lvl2(bz.Name(), "adding new block")
			err := bz.f.SaveAs("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
			if err != nil {
				log.Lvl2("Panic Raised:\n\n")
				panic(err)
			} else {
				log.Lvl2(bz.TreeNode().Name(), "is the final leader of round number ", bz.roundNumber, "$$$$$$$$$$$")
				bz.roundNumber = bz.roundNumber + 1
				bz.isLeader = false
			}
		} else {
			log.Lvl2("another leader has already published his block for round number", bz.roundNumber)
			bz.roundNumber = bz.roundNumber + 1
			bz.isLeader = false
			bz.f = f2
		}
		bz.checkLeadership()
	}
	log.Lvl2(bz.Name(), "Un-Lock BC")
} */
/* ----------------------------------------------------------------------
------------------------------------------------------------------------ */
//checkLeadership
// func (bz *BaseDFS) checkLeadership() {
// 	if bz.leaders[int(math.Mod(float64(bz.roundNumber), float64(len(bz.Roster().List))))] == bz.ServerIdentity().String() {
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
	if d == false {
		log.Lvl2(d)
	}
}
