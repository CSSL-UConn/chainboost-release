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
		"crypto/sha256"
		"encoding/hex"
		"fmt"
		onet "github.com/basedfs"
		"github.com/basedfs/blockchain"
		"github.com/basedfs/blockchain/blkparser"
		"github.com/basedfs/log"
		"github.com/basedfs/network"
		"github.com/basedfs/simul/monitor"
		crypto "github.com/basedfs/vrf"
		"github.com/xuri/excelize/v2"
		"math/big"
		"strconv"

		"sync"
		"time"
	)

	func init() {
		//network.RegisterMessage(ProofOfRetTxChan{})
		//network.RegisterMessage(PreparedBlockChan{})
		network.RegisterMessage(HelloBaseDFS{})
		onet.GlobalProtocolRegister("BaseDFS", NewBaseDFSProtocol)
	}

	//---------------- these channel were used for communication----------
	/*type ProofOfRetTxChan struct {
		*onet.TreeNode
		por.Por
	}

	type PreparedBlockChan struct {
		*onet.TreeNode
		PreparedBlock
	}*/
	//-----------------------------------------------------------------------
	// Hello is sent down the tree from the root node, every node who gets it starts the protocol and send it to its children
	type HelloBaseDFS struct {
		Timeout time.Duration
		//---- ToDo: Do i need timeout?
		PercentageTxEscrow string
		PercentageTxPoR string
		PercentageTxPay string
		RoundDuration time.Duration
	}

	type HelloChan struct {
		*onet.TreeNode
		HelloBaseDFS
	}
	//---------------- transaction and block structure --------------------
	type PreparedBlock struct {
		blockchain.Block
	}

	type LeadershipProof struct {
		proof crypto.VrfProof
	}

	type tx struct {
		i int8
	} //ToDo: the structure for three types of transactions: payment, escrow creation, and PoR should be finalized
	//---------------------------------------------------------------------
	// baseDFS is the main struct for running the protocol
	type BaseDFS struct {
		// the node we are represented-in
		*onet.TreeNodeInstance
		//ToDo : check if the TreeNodeInstance's private key can be used here
		ECPrivateKey crypto.VrfPrivkey
		// channel used to let all servers that the protocol has started
		HelloChan chan HelloChan
		// transactions is the slice of transactions that contains transactions
		transactions  []blkparser.Tx //ToDo : Do I need it now?
		// finale block that this BaseDFS epoch has produced
		finalBlock       *blockchain.TrBlock //ToDo: finalize block's structure
		// the suite we use
		suite network.Suite //ToDo: check what suit it is
		//  ToDo: I want to sync the nodes' time,it happens when nodes recieve hello msg. So after each roundDuration, nodes go after next round's block
		startBCMeasure *monitor.TimeMeasure
		// onDoneCallback is the callback that will be called at the end of the protocol
		onDoneCallback func() //ToDo: define this function and call it when you want to finish the protocol + check when should it be called
		// channel to notify when we are done -- when a message is sent through this channel a dispatch function in .. file will catch it and finish the protocol.
		DoneBaseDFS chan bool //it is not initiated in new proto!
		// ------------------------------------------------------------------------------------------------------------------
		//  -----  system-wide configurations params from the config file
		// ------------------------------------------------------------------
		//these  params get initialized
		//for the root node: "after" NewBaseDFSProtocol call (in func: Simulate in file: runsimul.go)
		//for the rest of nodes node: while joining protocol by the HelloBaseDFS message
		PercentageTxEscrow string
		PercentageTxPoR string
		PercentageTxPay string
		RoundDuration time.Duration
		// ------------------------------------------------------------------
		roundNumber int
		// ------------------------------------------------------------------------------------------------------------------
		//ToDo: dol I need these items?
		vcMeasure *monitor.TimeMeasure
		// lock associated
		doneLock sync.Mutex
		timeout   time.Duration
		timeoutMu sync.Mutex
	}
	// NewBaseDFSProtocol returns a new BaseDFS struct
	func NewBaseDFSProtocol(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
		bz := &BaseDFS{
			TreeNodeInstance: n,
			suite:            				n.Suite(),
			DoneBaseDFS:      		make(chan bool, 1),
			roundNumber:			0,
		}
		if err := n.RegisterChannel(&bz.HelloChan); err != nil {
			return bz, err
		}
		// bls key pair for each node for VRF
		//ToDo: raha: check how nodes key pair are handeled in network layer
		_, bz.ECPrivateKey = crypto.VrfKeygen()
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
	//Start: starts the protocol by sending hello msg to all nodes //ToDo : could I don't send any mgs?
	func (bz *BaseDFS) Start() error {
		// update the centralbc file with created nodes' information
		bz.finalCentralBCInitialization()
		//por.Testpor()
		//crypto.Testvrf()
		bz.helloBaseDFS()
		log.Lvl2(bz.TreeNode().Name(), "Started the protocol")
		return nil
	}
	//finalCentralBCInitialization initialize the central bc file based on the config params defined in the config file (.toml file of the protocol)
	// the info we hadn't before and we have now is nodes' info that this function add to the centralbc file
	func (bz *BaseDFS) finalCentralBCInitialization() {
		var NodeInfoRow []string
		for _, a := range bz.Roster().List{
			NodeInfoRow = append(NodeInfoRow, a.String())
		}
		f, _ := excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
		// --- market matching sheet
		index := f.GetSheetIndex("MarketMatching")
		f.SetActiveSheet(index)
		err := f.SetSheetRow("MarketMatching", "B1", &NodeInfoRow)
		if err != nil {
			log.LLvl2(err)
		}
		// --- power table sheet
		index = f.GetSheetIndex("PowerTable")
		f.SetActiveSheet(index)
		err = f.SetSheetRow("PowerTable", "B1", &NodeInfoRow)
		if err != nil {
			log.LLvl2(err)
		}

		if err := f.SaveAs("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx"); err != nil {
			log.LLvl2(err)
		}
	}
	// Dispatch listen on the different channels
	func (bz *BaseDFS) Dispatch() error {
		//var timeoutStarted bool
		running := true
		var err error
		for running {
			select {
			case msg := <-bz.HelloChan:
				log.Lvl2(bz.Info(), "received Hello/config params from", msg.TreeNode.ServerIdentity.Address)
				bz.PercentageTxEscrow = msg.PercentageTxEscrow
				bz.PercentageTxPoR= msg.PercentageTxPoR
				bz.PercentageTxPay= msg.PercentageTxPay
				bz.RoundDuration= msg.RoundDuration
				bz.helloBaseDFS()
			// this msg is catched in simulation codes
			case <-bz.DoneBaseDFS:
				running = false
			case <-time.After(5 * time.Second):
				if r := bz.readBCForNewRound(); r == true {
					// a timer should get started now for the next selected leader to use it later
					bz.createEpochBlock(bz.checkLeadership())
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
				panic (err)
			}
		}
		return err
	}
	//helloBaseDFS
	func (bz *BaseDFS) helloBaseDFS() {
		log.Lvl2(bz.TreeNode().Name(), " joined to the protocol")
		//bz.startBCMeasure = monitor.NewTimeMeasure("viewchange") //ToDo: check monitor.measure
		// ----------------------------------------
		if !bz.IsLeaf() {
			for _, child := range bz.Children() {
				go func(c *onet.TreeNode) {
					err := bz.SendTo(c, &HelloBaseDFS{
						Timeout: bz.timeout,
						PercentageTxEscrow: bz.PercentageTxEscrow ,
						PercentageTxPoR: bz.PercentageTxPoR ,
						PercentageTxPay: bz.PercentageTxPay ,
						RoundDuration: bz.RoundDuration ,})
					if err != nil {log.Lvl2(bz.Info(), "couldn't send hello to child", c.Name())}
				}(child)
			}
			log.Lvl2(bz.TreeNode().Name(), " checks if he is a leader...")
			bz.createEpochBlock(bz.checkLeadership())
		} else {
			bz.createEpochBlock(bz.checkLeadership())
		}
	}
	//checkLeadership
	func (bz *BaseDFS) checkLeadership() (bool, crypto.VrfProof) {
		power, seed := bz.readBCPreCheckLeadership()
		var vrfOutput [64]byte
		var bi *big.Int
		toBeHashed:= []byte(seed)
		proof, ok := bz.ECPrivateKey.ProveBytes(toBeHashed[:])
		if !ok {
			log.LLvl2("error while generating proof")
		}
		_, vrfOutput = bz.ECPrivateKey.Pubkey().VerifyBytes(proof, toBeHashed[:])
		// For future refrence: the next commented line is a wrong way to convert a big number to int - it wont overflow but the result is incorrect
		// t := binary.BigEndian.Uint64(vrfOutput[:])
		bi = new(big.Int).SetBytes(vrfOutput[:])
		if r:= bi.Cmp(power); r==-1  {
			return true, proof
		} else {
			return false, proof
		}
	}
	//createEpochBlock: by leader
	func (bz *BaseDFS) createEpochBlock(ok bool, p crypto.VrfProof) {
		if ok == false {
			log.LLvl2(bz.TreeNode().Name(), "is not a leader!")
		} else {
			log.LLvl2(bz.Info(), "is a leader for round number", bz.roundNumber,
				"\n wait for round duration:", bz.RoundDuration,
				"and then check if another leader has already updated bc ...")
			time.Sleep(bz.RoundDuration * time.Second) //ToDo: use timer
			if result := bz.readBCPostLeadership(); result == true {
				bz.updateBCPostLeadership()
			}
		}
	}
	// readBCPostLeadership checks if another leader has updated the centralbc file for this round earlier
	func (bz *BaseDFS) readBCPostLeadership() bool {
		//  readBCForNewRound returns "true" when a new round has been started / another leader has updated the centralbc file
		if bz.readBCForNewRound() == false {
			return true // you are the first leader. go ahead and update the centralbc file
		} else {return true}
	}
	func (bz *BaseDFS) readBCPreCheckLeadership () (power *big.Int, seed string){
		f, _ := excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
		var rows *excelize.Rows
		var row []string
		var err error
		rowNumber := 0
		// looking for last round's seed in the round table sheet in the centralbc file
		if rows, err = f.Rows("RoundTable"); err!=nil {log.LLvl2(err)}
		for rows.Next() {
			rowNumber++
			if row, err = rows.Columns(); err!=nil{log.LLvl2(err)}
		}
		for i, colCell := range row {
			// --- in RoundTable: i = 0 is (next) round number, i = 1 is (next) round seed, i=2 is blockchain size (empty now, will be updated by the leader)
			if i == 0 {if bz.roundNumber,err = strconv.Atoi(colCell); err!=nil {log.LLvl2(err)}}
			if i == 1 {seed = colCell}
		}
		log.Lvl2(seed)
		// looking for my power in the last round in the power table sheet in the centralbc file
		rowNumber = 0 //ToDo: later it can go straight to last row based on the round number found in round table
		if rows, err = f.Rows("PowerTable"); err!=nil {log.LLvl2(err)}
		for rows.Next() {
			rowNumber++
			if row, err = rows.Columns(); err!=nil{log.LLvl2(err)}
		}
		var myColumn []string
		var myCell string
		myColumn, err = f.SearchSheet("PowerTable", bz.ServerIdentity().String())
		myCell = myColumn[0][:1] + strconv.Itoa(rowNumber)
		var p string
		p,err = f.GetCellValue("PowerTable", myCell)
		log.Lvl2(p)
		var success bool
		power, success = new(big.Int).SetString( p,10) //ToDo: check if another base should have been set
		//power, err = strconv.ParseFloat(p, 64)
		if success == true {
			log.LLvl2("blockchain: we have had", rowNumber-2, "rounds intil now")
			log.LLvl2("blockchain:", bz.ServerIdentity().String() ,"'s power is", power, "and current round seed is", seed)
			return power, seed
		} else {
			panic("error in decoding power (string) to big.int from centralbc file")
		}
	}
	func (bz *BaseDFS) readBCForNewRound () bool {
		f, _ := excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
		var rows *excelize.Rows
		var row []string
		var err error
		rowNumber := 0
		var lastRound int
		// looking for last round's seed in the round table sheet in the centralbc file
		if rows, err = f.Rows("RoundTable"); err!=nil {log.LLvl2(err)}
		for rows.Next() {
			rowNumber++
			if row, err = rows.Columns(); err!=nil{log.LLvl2(err)}
		}
		for i, colCell := range row {
			// --- in RoundTable: i = 0 is (next) round number, i = 1 is (next) round seed, i=2 is blockchain size (empty now, will be updated by the leader)
			if i == 0 {lastRound, _ = strconv.Atoi(colCell)}
		}
		if  lastRound != bz.roundNumber {
			return true // a new round has been started
		} else {return false}
	}
	//updateBC: by leader
	func (bz *BaseDFS) updateBCPostLeadership () {
		f, _ := excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx")
		var rows *excelize.Rows
		var row []string
		var err error
		var seed string
		//var power *big.Int
		rowNumber := 0
		// looking for last round's seed in the round table sheet in the centralbc file
		if rows, err = f.Rows("RoundTable"); err!=nil {log.LLvl2(err)}
		for rows.Next() {
			rowNumber++
			if row, err = rows.Columns(); err!=nil{log.LLvl2(err)}
		}
		for i, colCell := range row {
			// --- in RoundTable: i = 0 is (next) round number, i = 1 is (next) round seed, i=2 is blockchain size (empty now, will be updated by the leader)
			//if i == 0 {if bz.roundNumber,err = strconv.Atoi(colCell); err!=nil {log.LLvl2(err)}}  // i dont want to change the round number now, even if it is changed, i will re-check it at the end!
			if i == 1 {seed = colCell} // next round's seed is the hash of this seed
		}
		// --------------------------------------------------------------------
		// updating the current last row in the "BCsize" column
		currentRow := strconv.Itoa(rowNumber)
		axisBCSize := "C" + currentRow
		err = f.SetCellValue("RoundTable", axisBCSize, 10) //ToDo: measure and update bcsize
		if err != nil {
			return
		}
		// --------------------------------------------------------------------
		// adding one row in round table (round number and seed columns)
		nextRow := strconv.Itoa(rowNumber+1)
		axisRoundNumber := "A" + nextRow
		err = f.SetCellValue("RoundTable", axisRoundNumber, bz.roundNumber+1)
		if err != nil {
			return
		}
		// ---  next round's seed is the hash of current seed
		data := fmt.Sprintf("%v", seed)
		sha := sha256.New()
		if _, err := sha.Write([]byte(data)); err != nil {
			log.Error("Couldn't hash header:", err)
		}
		hash := sha.Sum(nil)
		axisSeed := "B" + nextRow
		err = f.SetCellValue("RoundTable", axisSeed, hex.EncodeToString(hash))
		// adding one row in power table and updating new miners' power for each miner
		// based on thier information in market matching sheet

		// looking for my power in the last round in the power table sheet in the centralbc file
/*		rowNumber = 0 //ToDo: later it can go straight to last row based on the round number found in round table
		if rows, err = f.Rows("PowerTable"); err!=nil {log.LLvl2(err)}
		for rows.Next() {
			rowNumber++
			if row, err = rows.Columns(); err!=nil{log.LLvl2(err)}
		}
		var myColumn []string
		var myCell string
		myColumn, err = f.SearchSheet("PowerTable", bz.ServerIdentity().String())
		myCell = myColumn[0][:1] + strconv.Itoa(rowNumber)
		var p string
		p,err = f.GetCellValue("PowerTable", myCell)
		log.Lvl2(p)
		var success bool
		power, success = new(big.Int).SetString( p,10) //ToDo: check if another base should have been set
		//power, err = strconv.ParseFloat(p, 64)
		if success == true {
			log.LLvl2("blockchain: we have had", rowNumber-2, "rounds intil now")
			log.LLvl2("blockchain:", bz.ServerIdentity().String() ,"'s power is", power, "and current round seed is", seed)
		} else {
			panic("error in decoding power (string) to big.int from centralbc file")
		}*/
		// --------------------------------------------------------------------
		// re-check: being the leader (the earliest leader in this round) before saving
		// --------------------------------------------------------------------
		if result := bz.readBCPostLeadership(); result == true {
			if err := f.SaveAs("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx"); err != nil {
				log.LLvl2(err)
			}
		} else {
			log.LLvl2("another leader has already published his block")
		}
	}

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
	// SetTimeout sets the new timeout
	//ToDo: do we need the following functions?
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