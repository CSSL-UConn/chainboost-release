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
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"
	"time"

	onet "github.com/basedfs"
	"github.com/basedfs/blockchain"
	"github.com/basedfs/blockchain/blkparser"
	"github.com/basedfs/log"
	"github.com/basedfs/network"
	"github.com/basedfs/por"
	"github.com/basedfs/simul/monitor"
	"github.com/basedfs/vrf"
	"github.com/xuri/excelize/v2"
)

/* -------------------------------------------------------------------- */
// ------- TYPES  --------------------------------------------
/* -------------------------------------------------------------------- */
//  -------------------  Transactions and Block Structurse -----------------
/* -------------------------------------------------------------------- */
/*  payment transactions are close to byzcoin's and from the structure
provided in: https://developer.bitcoin.org/reference/transactions.html */
type TxPay struct {
	LockTime uint
	Version  [4]byte
	TxInCnt  uint // compactSize uint
	TxOutCnt uint
	TxPayIns []*TxPayIn
	TxOuts   []*TxPayOut
}
type TxPayIn struct {
	outpoint            *outpoint // 36 bytes
	UnlockingScriptSize uint
	UnlockinScript      string // signature script
	SequenceNumber      uint   //?
}
type outpoint struct {
	hash  [32]string //TXID
	index [4]byte
}
type TxPayOut struct {
	Addr              string
	Amount            [8]byte
	LockingScript     string // pubkey script
	LockingScriptSize uint
}

/* ---------------- market matching transactions ---------------- */
type Contract struct {
	clientPk      por.PublicKey
	duration      time.Duration
	serverAddress string //ServerIdentity.Address.String() = treenode().name()
}

/*after matching is done (contract is created) the client creates an escrow*/
type TxEscrow struct {
	tx       *TxPay
	clientId por.PublicKey
}

/* ---------------- transactions that will be issued until a contract is active ---------------- */
/* por txs are designed in away that the verifier (any miner) has sufficient information to verify it */
type TxPoR struct {
	cId         *Contract
	por         *por.Por
	Tau         []byte
	roundNumber uint // to determine the random query used for it
}

/* ---------------- transactions that will be issued for each round ---------------- */
/* the proof is generated as an output of fucntion call "ProveBytes" in vrf package*/
/* ---------------- block structure and its metadata ----------------
(from algorand) : "Blocks consist of a list of transactions,  along with metadata needed by MainChain miners.
Specifically, the metadata consists of
	- the round number,
	- the proposer’s VRF-based seed,
	- a hash of the previous block in the ledger,and
	- a timestamp indicating when the block was proposed
The list of transactions in a block logically translates to a set of weights for each user’s public key
(based on the balance of currency for that key), along with the total weight of all outstanding currency." //ToDo: compelete this later
*/
type TransactionList struct {
	//---
	TxPays   []*TxPay
	TxPayCnt uint
	//---
	TxPoRs   []*TxPoR
	TxPoRCnt uint
	//---
	TxEscrows   []*TxEscrow
	TxEscrowCnt uint
	//---
	Fees float64
}
type BlockHeader struct {
	RoundNumber       uint
	RoundSeed         string //proposer’s VRF-based seed
	PreviousBlockHash string
	Timestamp         uint
	//--
	MerkleRootHash string
	// -- ToDo: these two are required as well, right?
	Version             [4]byte
	SignedBlockByLeader []byte //length?
	LeaderAddress       string //ToDo: public key?
}
type Block struct {
	BlockSize uint
	*BlockHeader
	*TransactionList
}

// ------  types used for communication (msgs) -------------------------
/*Hello is sent down the tree from the root node, every node who gets it starts the protocol and send it to its children*/
type HelloBaseDFS struct {
	Timeout time.Duration
	//---- ToDo: Do i need timeout?
	PercentageTxEscrow string
	PercentageTxPoR    string
	PercentageTxPay    string
	RoundDuration      time.Duration
	BlockSize          string
}
type HelloChan struct {
	*onet.TreeNode
	HelloBaseDFS
}

//  BaseDFS is the main struct for running the protocol ------------
type BaseDFS struct {
	// the node we are represented-in
	*onet.TreeNodeInstance
	//ToDo : check if the TreeNodeInstance's private key can be used here
	ECPrivateKey vrf.VrfPrivkey
	// channel used to let all servers that the protocol has started
	HelloChan chan HelloChan
	// transactions is the slice of transactions that contains transactions
	transactions []blkparser.Tx //ToDo : Do I need it now?
	// finale block that this BaseDFS epoch has produced
	finalBlock *blockchain.TrBlock //ToDo: finalize block's structure
	// the suite we use
	suite network.Suite //ToDo: check what suit it is
	//  ToDo: I want to sync the nodes' time,it happens when nodes recieve hello msg. So after each roundDuration, nodes go after next round's block
	startBCMeasure *monitor.TimeMeasure
	// onDoneCallback is the callback that will be called at the end of the protocol
	onDoneCallback func() //ToDo: define this function and call it when you want to finish the protocol + check when should it be called
	// channel to notify when we are done -- when a message is sent through this channel a dispatch function in .. file will catch it and finish the protocol.
	DoneBaseDFS chan bool //it is not initiated in new proto!
	// channel to notify leader elected
	LeaderPropose chan bool
	// ------------------------------------------------------------------------------------------------------------------
	//  -----  system-wide configurations params from the config file
	// ------------------------------------------------------------------
	//these  params get initialized
	//for the root node: "after" NewBaseDFSProtocol call (in func: Simulate in file: runsimul.go)
	//for the rest of nodes node: while joining protocol by the HelloBaseDFS message
	PercentageTxEscrow string
	PercentageTxPoR    string
	PercentageTxPay    string
	RoundDuration      time.Duration
	BlockSize          string
	timeout            time.Duration
	// ------------------------------------------------------------------
	roundNumber int
	isLeader    bool
	// ------------------------------------------------------------------------------------------------------------------
	//ToDo: dol I need these items?
	vcMeasure *monitor.TimeMeasure
	// lock associated
	doneLock  sync.Mutex
	timeoutMu sync.Mutex
	// ------------------------------------------------------------------------------------------------------------------
	f   *excelize.File
	fMu sync.Mutex
}

/* -------------------------------------------------------------------- */
// ------- FUNCTIONS  -------------------------------------------------
/* -------------------------------------------------------------------- */
func init() {
	//network.RegisterMessage(ProofOfRetTxChan{})
	//network.RegisterMessage(PreparedBlockChan{})
	network.RegisterMessage(HelloBaseDFS{})
	onet.GlobalProtocolRegister("BaseDFS", NewBaseDFSProtocol)
}

/* ----------------------------------------------------------------------
-----------------
------------------------------------------------------------------------ */
// NewBaseDFSProtocol returns a new BaseDFS struct
func NewBaseDFSProtocol(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
	bz := &BaseDFS{
		TreeNodeInstance: n,
		suite:            n.Suite(),
		DoneBaseDFS:      make(chan bool, 1),
		LeaderPropose:    make(chan bool, 1),
		roundNumber:      0,
	}
	if err := n.RegisterChannel(&bz.HelloChan); err != nil {
		return bz, err
	}
	// bls key pair for each node for VRF
	//ToDo: raha: check how nodes key pair are handeled in network layer
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
	var err error
	bz.fMu.Lock()
	defer bz.fMu.Unlock()
	bz.f, err = excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
	if err != nil {
		log.LLvl2("Raha: ", err)
		panic(err)
	}
	//---------------------------------------------------------------------
	return bz, nil
}

/* ----------------------------------------------------------------------
-----------------
------------------------------------------------------------------------ */
//Start: starts the protocol by sending hello msg to all nodes //ToDo : could I don't send any mgs?
func (bz *BaseDFS) Start() error {
	// update the centralbc file with created nodes' information
	bz.finalCentralBCInitialization()
	//por.Testpor()
	//vrf.Testvrf()
	bz.helloBaseDFS()
	log.Lvl2(bz.TreeNode().Name(), "Started the protocol")
	return nil
}

/* ----------------------------------------------------------------------
-----------------
------------------------------------------------------------------------ */
//finalCentralBCInitialization initialize the central bc file based on the config params defined in the config file (.toml file of the protocol)
// the info we hadn't before and we have now is nodes' info that this function add to the centralbc file
func (bz *BaseDFS) finalCentralBCInitialization() {
	var NodeInfoRow []string
	for _, a := range bz.Roster().List {
		NodeInfoRow = append(NodeInfoRow, a.String())
	}

	/* var (
		f  *excelize.File
		mu sync.Mutex // guards f,
	)*/
	var err error

	//failedAttempts := 0
	bz.fMu.Lock()
	defer bz.fMu.Unlock()

	//f, err = excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
	/* file, err := os.Open("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	defer file.Close()
	f, err := excelize.OpenReader(file)
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	f.Path = "/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx"
	// ---------- */
	//if err != nil {
	//	log.LLvl2("Raha: ", err)
	//}
	/* {
		log.LLvl2(bz.TreeNode().Name(), "Can't open the centralbc file: pause 1 sec. for each attempt ----------------")
		for err != nil {
			if failedAttempts == 11 {
				log.LLvl2("Can't open the centralbc file: 10 attempts!")
			}
			time.Sleep(1 * time.Second)
			f, err = excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
			/* file, err := os.Open("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
			if err != nil {
				log.LLvl2("Panic Raised:\n\n")
				panic(err)
			}
			defer file.Close()
			f, err := excelize.OpenReader(file)
			if err != nil {
				log.LLvl2("Panic Raised:\n\n")
				panic(err)
			}
			f.Path = "/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx"
			// ----------
			failedAttempts = failedAttempts + 1
		}
		log.LLvl2("---------- Opened the centralbc file after ", failedAttempts, " attempts")
	} */
	// --- market matching sheet
	index := bz.f.GetSheetIndex("MarketMatching")
	bz.f.SetActiveSheet(index)
	// column format
	for i := 2; i <= len(NodeInfoRow)+1; i++ {
		contractRow := strconv.Itoa(i)
		t := "A" + contractRow
		err = bz.f.SetCellValue("MarketMatching", t, NodeInfoRow[i-2])
	}
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	// --- power table sheet
	index = bz.f.GetSheetIndex("PowerTable")
	bz.f.SetActiveSheet(index)
	err = bz.f.SetSheetRow("PowerTable", "B1", &NodeInfoRow)
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	// test sheet
	bz.f.NewSheet("TestSheet")

	err = bz.f.SaveAs("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	} else {
		bz.f, err = excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
		if err != nil {
			log.LLvl2("Panic Raised:\n\n")
			panic(err)
		}
	}
}

/* ----------------------------------------------------------------------
-----------------
------------------------------------------------------------------------ */
// Dispatch listen on the different channels
func (bz *BaseDFS) Dispatch() error {
	//var timeoutStarted bool
	running := true
	var err error
	for running {
		select {
		case msg := <-bz.HelloChan:
			log.Lvl2(bz.TreeNode().Name(), "received Hello/config params from", msg.TreeNode.ServerIdentity.Address)
			bz.PercentageTxEscrow = msg.PercentageTxEscrow
			bz.PercentageTxPoR = msg.PercentageTxPoR
			bz.PercentageTxPay = msg.PercentageTxPay
			bz.RoundDuration = msg.RoundDuration
			bz.BlockSize = msg.BlockSize
			bz.helloBaseDFS()
		// this msg is catched in simulation codes
		case <-bz.DoneBaseDFS:
			running = false
		case <-time.After(2 * time.Second):
			if bz.isLeader == false {
				//log.LLvl2(bz.Name(), "Lock: locked to refresh")
				//bz.fMu.Lock()
				bz.f, err = excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
				if err != nil {
					log.LLvl2("Raha: ", bz.Name(), "--", err)
					panic(err)
				} else {
					log.LLvl2(bz.Name(), "refresh BC")
				}
				//log.LLvl2(bz.Name(), "Lock: un-locked")
				//bz.fMu.Unlock()
				// --
				if bz.readBCForNewRound(bz.f) == true {
					// a timer should get started now for the next selected leader to use it later
					// and after round duration of the timer, if its a leader, propose the block,
					// if its not and if no body else have prposed any block, some body shoudl!
					//bz.SetTimeout(bz.RoundDuration)
					bz.checkLeadership()
				}
			} else {
				log.LLvl2(bz.Name(), "Lock: locked to add")
				bz.fMu.Lock()
				//---
				f2, err2 := excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
				if err2 != nil {
					log.LLvl2("Raha: ", bz.Name(), "--", err2)
					panic(err2)
				}
				r := bz.readBCForNewRound(f2)
				if r == false {
					//---
					log.LLvl2(bz.Name(), "Lock: --", "adding new block")
					err := bz.f.SaveAs("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
					if err != nil {
						log.LLvl2("Panic Raised:\n\n")
						panic(err)
					} else {
						log.LLvl2(bz.TreeNode().Name(), "is the final leader of round number ", bz.roundNumber, "$$$$$$$$$$$")
						bz.roundNumber = bz.roundNumber + 1
						bz.isLeader = false
					}
				} else {
					log.LLvl2("another leader has already published his block for round number", bz.roundNumber)
					bz.isLeader = false
				}
				log.LLvl2(bz.Name(), "Lock: un-locked")
				bz.fMu.Unlock()
				bz.checkLeadership()
			}

		case <-bz.LeaderPropose:
			log.LLvl2(bz.Name(), "I am elected :)")
			time.Sleep(bz.RoundDuration) //ToDO: it should be based on timer "until" round duration
			if bz.readBCForNewRound(bz.f) == false {
				bz.updateBCPostLeadership()
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
		if err != nil {
			log.Error(bz.Name(), "Error handling messages:", err)
			panic(err)
		}
	}
	return err
}

/* ----------------------------------------------------------------------
-----------------
------------------------------------------------------------------------ */
//helloBaseDFS
func (bz *BaseDFS) helloBaseDFS() {
	log.Lvl2(bz.TreeNode().Name(), " joined to the protocol")
	//bz.startBCMeasure = monitor.NewTimeMeasure("viewchange") //ToDo: check monitor.measure
	// ----------------------------------------
	if !bz.IsLeaf() {
		for _, child := range bz.Children() {
			go func(c *onet.TreeNode) {
				err := bz.SendTo(c, &HelloBaseDFS{
					Timeout:            bz.timeout,
					PercentageTxEscrow: bz.PercentageTxEscrow,
					PercentageTxPoR:    bz.PercentageTxPoR,
					PercentageTxPay:    bz.PercentageTxPay,
					RoundDuration:      bz.RoundDuration,
					BlockSize:          bz.BlockSize})
				if err != nil {
					log.Lvl2(bz.Info(), "couldn't send hello to child", c.Name())
				}
			}(child)
		}
		bz.checkLeadership()
	} else {
		bz.checkLeadership()
	}
}

/* ----------------------------------------------------------------------
-----------------
------------------------------------------------------------------------ */
//checkLeadership
func (bz *BaseDFS) checkLeadership() {
	power, seed := bz.readBCMyPowerNextSeed()
	var vrfOutput [64]byte

	toBeHashed := []byte(seed)
	proof, ok := bz.ECPrivateKey.ProveBytes(toBeHashed[:])
	if !ok {
		log.LLvl2("error while generating proof")
	}
	_, vrfOutput = bz.ECPrivateKey.Pubkey().VerifyBytes(proof, toBeHashed[:])

	/* ------------------------- working with big int ----------------------
	"math/big" imported
	func generateRandomValuesBigInt(nodes int) [] *big.Int {
		var bigIntlist [] *big.Int
		for i := 1; i<= nodes;i++{
			bigIntlist = append(bigIntlist, new(big.Int).Rand(rand.New(rand.NewSource(time.Now().UnixNano())),<some big int value> ))
		}
		return bigIntlist
	}
	// --------------------------------------------------------------------
	For future refrence: the next commented line is a wrong way to convert a big number to int - it wont raise overflow but the result is incorrect
	t := binary.BigEndian.Uint64(vrfOutput[:])
	--------------------------------------------------
	For future refrence: the next commented line is converting a big number to big int (built in type in go)
	var bi *big.Int
	bi = new(big.Int).SetBytes(vrfOutput[:])
	--------------------------------------------------   */

	var vrfoutputInt64 uint64
	buf := bytes.NewReader(vrfOutput[:])
	err := binary.Read(buf, binary.LittleEndian, &vrfoutputInt64)
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	if vrfoutputInt64 < power {
		bz.LeaderPropose <- true
	} else {
		log.LLvl2("my power:", power, "is", vrfoutputInt64-power, "less than my vrf output :| ")
	}
}

/* ----------------------------------------------------------------------
-----------------
------------------------------------------------------------------------ */
func (bz *BaseDFS) readBCMyPowerNextSeed() (power uint64, seed string) {
	/* 	var (
		f  *excelize.File
		mu sync.Mutex // guards f,
	) */
	var err error

	//failedAttempts := 0
	/* 	bz.fMu.Lock()
	   	defer bz.fMu.Unlock() */

	//f, err = excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
	/* file, err := os.Open("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	defer file.Close()
	f, err := excelize.OpenReader(file)
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	f.Path = "/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx"
	// ---------- */

	/* 	if err != nil {
		log.LLvl2("Raha: ", err)
	} */

	/* {
		log.LLvl2(bz.TreeNode().Name(), "Can't open the centralbc file: pause 1 sec. for each attempt ---------------- error msg: ", err)
		for err != nil {
			if failedAttempts == 11 {
				log.LLvl2("Can't open the centralbc file: 10 attempts!")
			}
			time.Sleep(1 * time.Second)
			f, err = excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
			/* file, err := os.Open("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
			if err != nil {
				log.LLvl2("Panic Raised:\n\n")
				panic(err)
			}
			defer file.Close()
			f, err := excelize.OpenReader(file)
			if err != nil {
				log.LLvl2("Panic Raised:\n\n")
				panic(err)
			}
			f.Path = "/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx"
			// ----------
			failedAttempts = failedAttempts + 1

		}
		log.LLvl2("---------- Opened the centralbc file after ", failedAttempts, " attempts")
	} */
	var rows *excelize.Rows
	var row []string
	rowNumber := 0
	// looking for last round's seed in the round table sheet in the centralbc file
	if rows, err = bz.f.Rows("RoundTable"); err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	for rows.Next() {
		rowNumber++
		if row, err = rows.Columns(); err != nil {
			log.LLvl2("Panic Raised:\n\n")
			panic(err)
		}
	}
	for i, colCell := range row {
		// --- in RoundTable: i = 0 is (next) round number, i = 1 is (next) round seed, i=2 is blockchain size (empty now, will be updated by the leader)
		if i == 0 {
			if bz.roundNumber, err = strconv.Atoi(colCell); err != nil {
				log.LLvl2("Panic Raised:\n\n")
				panic(err)
			}
		}
		if i == 1 {
			seed = colCell
		}
	}
	// looking for my power in the last round in the power table sheet in the centralbc file
	rowNumber = 0 //ToDo: later it can go straight to last row based on the round number found in round table
	if rows, err = bz.f.Rows("PowerTable"); err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	for rows.Next() {
		rowNumber++
		if row, err = rows.Columns(); err != nil {
			log.LLvl2("Panic Raised:\n\n")
			panic(err)
		}
	}
	var myColumnHeader []string
	var myCell string
	var myColumn string
	myColumnHeader, err = bz.f.SearchSheet("PowerTable", bz.ServerIdentity().String())
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	for _, character := range myColumnHeader[0] {
		if character >= 'A' && character <= 'Z' { // a-z isn't needed! just to make sure
			myColumn = myColumn + string(character)
		}
	}

	myCell = myColumn + strconv.Itoa(rowNumber) //such as A2,B3,C3..
	var p string
	p, err = bz.f.GetCellValue("PowerTable", myCell)
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	var t int
	t, err = strconv.Atoi(p)
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	power = uint64(t)
	// add accumulated power to recently added power in last round
	for i := 2; i < rowNumber; i++ {
		upperPowerCell := myColumn + strconv.Itoa(i)
		p, err = bz.f.GetCellValue("PowerTable", upperPowerCell)
		if err != nil {
			log.LLvl2("Panic Raised:\n\n")
			panic(err)
		}
		t, er := strconv.Atoi(p)
		if er != nil {
			log.LLvl2("Panic Raised:\n\n")
			panic(err)
		}
		upperPower := uint64(t)
		power = power + upperPower
	}

	return power, seed
}

/* ----------------------------------------------------------------------
-----------------
------------------------------------------------------------------------ */
func (bz *BaseDFS) readBCForNewRound(f *excelize.File) bool {
	/* 	var (
		f  *excelize.File
		mu sync.Mutex // guards f,
	) */
	var err error

	//failedAttempts := 0

	/* 	file, err := os.Open("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
	   	if err != nil {
	   		log.LLvl2("Panic Raised:\n\n")
	   		panic(err)
	   	}
	   	defer file.Close()
	   	f, err := excelize.OpenReader(file)
	   	if err != nil {
	   		log.LLvl2("Panic Raised:\n\n")
	   		panic(err)
	   	}
	   	f.Path = "/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx"
	   	// ---------- */

	/* 	{
		log.LLvl2(bz.TreeNode().Name(), "Can't open the centralbc file: pause 1 sec. for each attempt ----------------")
		for err != nil {
			if failedAttempts == 11 {
				log.LLvl2("Can't open the centralbc file: 10 attempts!")
			}
			time.Sleep(1 * time.Second)
			f, err = excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
			/* 		file, err := os.Open("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
			if err != nil {
				log.LLvl2("Panic Raised:\n\n")
				panic(err)
			}
			defer file.Close()
			f, err := excelize.OpenReader(file)
			if err != nil {
				log.LLvl2("Panic Raised:\n\n")
				panic(err)
			}
			f.Path = "/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx"
			// ----------
			failedAttempts = failedAttempts + 1
		}
		log.LLvl2("---------- Opened the centralbc file after ", failedAttempts, " attempts")
	} */
	var rows *excelize.Rows
	var row []string
	rowNumber := 0
	var lastRound int
	// looking for last round's seed in the round table sheet in the centralbc file
	if rows, err = f.Rows("RoundTable"); err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	for rows.Next() {
		rowNumber++
		if row, err = rows.Columns(); err != nil {
			log.LLvl2("Panic Raised:\n\n")
			panic(err)
		}
	}
	for i, colCell := range row {
		// --- in RoundTable: i = 0 is (next) round number, i = 1 is (next) round seed, i=2 is blockchain size (empty now, will be updated by the leader)
		if i == 0 {
			lastRound, err = strconv.Atoi(colCell)
			if err != nil {
				log.LLvl2("Panic Raised:\n\n")
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

/* ----------------------------------------------------------------------
-----------------
------------------------------------------------------------------------ */
//updateBC: by leader
func (bz *BaseDFS) updateBCPostLeadership() {
	/* 	var (
		f  *excelize.File
		mu sync.Mutex // guards f,
	) */
	var err error

	//failedAttempts := 0
	//bz.fMu.Lock()

	//f, err = excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
	/* 	file, err := os.Open("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
	   	if err != nil {
	   		log.LLvl2("Panic Raised:\n\n")
	   		panic(err)
	   	}
	   	defer file.Close()
	   	f, err := excelize.OpenReader(file)
	   	if err != nil {
	   		log.LLvl2("Panic Raised:\n\n")
	   		panic(err)
	   	}
	   	f.Path = "/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx"
	   	// ---------- */
	/*
	 */
	/* 	{
		log.LLvl2(bz.TreeNode().Name(), "Can't open the centralbc file: pause 1 sec. for each attempt ---------------- error:", err)
		for err != nil {
			if failedAttempts == 11 {
				log.LLvl2("Can't open the centralbc file: 10 attempts!")
			}
			time.Sleep(1 * time.Second)
			f, err = excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
			/* 			file, err := os.Open("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
			   			if err != nil {
			   				log.LLvl2("Panic Raised:\n\n")
			   				panic(err)
			   			}
			   			defer file.Close()
			   			f, err := excelize.OpenReader(file)
			   			if err != nil {
			   				log.LLvl2("Panic Raised:\n\n")
			   				panic(err)
			   			}
			   			f.Path = "/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx"
			   			// ----------
			failedAttempts = failedAttempts + 1
		}
		log.LLvl2("---------- Opened the centralbc file after ", failedAttempts, " attempts")
	} */
	var rows *excelize.Rows
	var row []string
	var seed string
	//var power *big.Int
	rowNumber := 0
	// looking for last round's seed in the round table sheet in the centralbc file
	if rows, err = bz.f.Rows("RoundTable"); err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	for rows.Next() {
		rowNumber++
		if row, err = rows.Columns(); err != nil {
			log.LLvl2("Panic Raised:\n\n")
			panic(err)
		}
	}
	for i, colCell := range row {
		// --- in RoundTable: i = 0 is (next) round number, i = 1 is (next) round seed, i=2 is blockchain size (empty now, will be updated by the leader)
		//if i == 0 {if bz.roundNumber,err = strconv.Atoi(colCell); err!=nil {log.LLvl2(err)}}  // i dont want to change the round number now, even if it is changed, i will re-check it at the end!
		if i == 1 {
			seed = colCell
		} // next round's seed is the hash of this seed
	}
	// --------------------------------------------------------------------
	// updating the current last row in the "BCsize" column
	currentRow := strconv.Itoa(rowNumber)
	axisBCSize := "C" + currentRow
	err = bz.f.SetCellValue("RoundTable", axisBCSize, bz.BlockSize) //ToDo: measure and update bcsize
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	// --------------------------------------------------------------------
	// updating the current last row in the "miner" column
	axisMiner := "D" + currentRow
	err = bz.f.SetCellValue("RoundTable", axisMiner, bz.TreeNode().Name())
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	// --------------------------------------------------------------------
	// adding one row in round table (round number and seed columns)
	nextRow := strconv.Itoa(rowNumber + 1)
	axisRoundNumber := "A" + nextRow
	err = bz.f.SetCellValue("RoundTable", axisRoundNumber, bz.roundNumber+1)
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	// ---  next round's seed is the hash of current seed
	data := fmt.Sprintf("%v", seed)
	sha := sha256.New()
	if _, err := sha.Write([]byte(data)); err != nil {
		log.Error("Couldn't hash header:", err)
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	hash := sha.Sum(nil)
	axisSeed := "B" + nextRow
	err = bz.f.SetCellValue("RoundTable", axisSeed, hex.EncodeToString(hash))
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}

	/*	each round, adding one row in power table based on the information in market matching sheet,
		assuming that servers are honest  and have honestly publish por for their actice (not expired) contracts,
		  for each storage server and each of their active contracst, add the stored file size to their current power*/
	if rows, err = bz.f.Rows("MarketMatching"); err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	var ContractDuration, ContractStartedRoundNumber, FileSize int
	var MinerServer string
	rowNum := 0
	MinerServers := make(map[string]int)
	for rows.Next() {
		rowNum++
		if rowNum == 1 {
			continue
		}
		row, err = rows.Columns()
		if err != nil {
			log.LLvl2("Panic Raised:\n\n")
			panic(err)
		} else {
			for i, colCell := range row {
				// --- in MarketMatching: i = 0 is Server's Info,  i = 1 is FileSize, i=2 is ContractDuration, i=3 is RoundNumber, i=4 is ContractID, i=5 is Client's PK
				// EXCELIZE KEEPS INITIALIZING ROWS WITH HAVING HEADERS VALUE FOR COLUMNS 1 TO 6, SO 1 STARTS FROM 7!!
				if i == 6 {
					MinerServer = colCell
				}
				if i == 7 {
					FileSize, err = strconv.Atoi(colCell)
					if err != nil {
						log.LLvl2("Panic Raised:\n\n")
						panic(err)
					}
				}
				if i == 8 {
					ContractDuration, err = strconv.Atoi(colCell)
					if err != nil {
						log.LLvl2("Panic Raised:\n\n")
						panic(err)
					}
				}
				if i == 9 {
					ContractStartedRoundNumber, err = strconv.Atoi(colCell)
					if err != nil {
						log.LLvl2("Panic Raised:\n\n")
						panic(err)
					}
				}
			}
		}
		if bz.roundNumber-ContractDuration <= ContractStartedRoundNumber {
			MinerServers[MinerServer] = FileSize //if each server one contract
		} else {
			MinerServers[MinerServer] = 0
		}
	}
	// --- Power Table sheet  ----------------------------------------------
	index := bz.f.GetSheetIndex("PowerTable")
	bz.f.SetActiveSheet(index)
	var PowerInfoRow []int
	for _, a := range bz.Roster().List {
		PowerInfoRow = append(PowerInfoRow, MinerServers[a.Address.String()])
	}
	axisRoundNumber = "B" + currentRow
	err = bz.f.SetSheetRow("PowerTable", axisRoundNumber, &PowerInfoRow)
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	/*
		// add recent power to past accumulated power
		err = f.SetCellFormula("PowerTable", "A3", "=SUM(A1,B1)")
		if err != nil {
			log.LLvl2("Panic Raised:\n\n")
			panic(err)
		}
	*/
	// --------------------------------------------------------------------
	// re-check: being the leader (the earliest leader in this round) before saving
	// --------------------------------------------------------------------
	bz.isLeader = true
}

/* ----------------------------------------------------------------------
-----------------
------------------------------------------------------------------------ */
// startTimer starts the timer to decide whether we should ... or not.
func (bz *BaseDFS) startTimer(millis uint64) {
	log.Lvl3(bz.Name(), "Started timer (", millis, ")...")
	select {
	case <-bz.DoneBaseDFS:
		return
	case <-time.After(time.Millisecond * time.Duration(millis)):
		//bz.sendAndMeasureViewchange()
	}
}

/* ----------------------------------------------------------------------
-----------------
------------------------------------------------------------------------ */
// SetTimeout sets the new timeout
//ToDo: do we need the following functions?
func (bz *BaseDFS) SetTimeout(t time.Duration) {
	bz.timeoutMu.Lock()
	bz.timeout = t
	bz.timeoutMu.Unlock()
}

/* ----------------------------------------------------------------------
-----------------
------------------------------------------------------------------------ */
// Timeout returns the current timeout
func (bz *BaseDFS) Timeout() time.Duration {
	bz.timeoutMu.Lock()
	defer bz.timeoutMu.Unlock()
	return bz.timeout
}

/* ----------------------------------------------------------------------
-----------------
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
		log.LLvl2(err)
	}
	c1, err := f.GetCellValue("Sheet1", "A1")
	if err != nil {
		log.LLvl2(err)
	}
	log.LLvl2(c1)
	c2, err := f.GetCellValue("Sheet1", "A4")
	if err != nil {
		log.LLvl2(err)
	}
	log.LLvl2(c2)
	c3, err := f.GetCellValue("Sheet1", "B2")
	result, err := f.SearchSheet("Sheet1", "100")
	if err != nil {
		log.LLvl2(err)
	}
	log.LLvl2(c3)
	log.LLvl2(result)
	f.SetCellValue("Sheet1", "A3", 42)
	if err := f.SaveAs("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/simple.xlsx"); err != nil {
		log.LLvl2(err)
	}
	log.LLvl2("wait")
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
	log.LLvl2(RoundDuration)
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
// ------------   Sortition Algorithm from ALgorand: ---------------------
// ⟨hash,π⟩←VRFsk(seed||role)
// p←τ/W
// j←0
// while hash/ 2^hashleng </ [ sigma(k=0,j) (B(k,w,p),  sigma(k=0,j+1) (B(k,w,p)] do
//	----	 j++
// return <hash,π, j>
// ----------------------------------------------------------------------
//structures
/*---------------------------------------------------------------------
---------------- these channel were used for communication----------
type ProofOfRetTxChan struct {
	*onet.TreeNode
	por.Por
}

type PreparedBlockChan struct {
	*onet.TreeNode
	PreparedBlock
}
-------------------------------------------------------------------- */
