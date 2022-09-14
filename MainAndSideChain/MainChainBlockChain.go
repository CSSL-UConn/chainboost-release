package MainAndSideChain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chainBoostScale/ChainBoost/MainAndSideChain/blockchain"
	"github.com/chainBoostScale/ChainBoost/onet/log"
	"github.com/xuri/excelize/v2"
)

/* ----------------------------------------------------------------------
	finalMainChainBCInitialization initialize the mainchainbc file based on the config params defined in the config file
	(.toml file of the protocol) the info we hadn't before and we have now is nodes' info that this function add to the mainchainbc file
------------------------------------------------------------------------ */
func (bz *ChainBoost) finalMainChainBCInitialization() {
	var NodeInfoRow []string
	for _, a := range bz.Roster().List {
		NodeInfoRow = append(NodeInfoRow, a.String())
	}
	var err error

	pwd, _ := os.Getwd()
	log.Lvl4("opening bc in:", pwd)
	//bcDirectory := strings.Split(pwd, "/build")
	bcDirectory := strings.Split(pwd, "/build")[0] + "/mainchainbc.xlsx"
	log.Lvl1("opening bc in:", bcDirectory)
	//f, err := excelize.OpenFile("/root/remote/mainchainbc.xlsx")
	f, err := excelize.OpenFile(bcDirectory)
	if err != nil {
		log.Fatal("problem while opening bc: " + err.Error())
	} else {
		log.Lvl2("bc Successfully opened")
	}

	// --- market matching sheet
	index := f.GetSheetIndex("MarketMatching")
	f.SetActiveSheet(index)
	// fill nodes info and get the maximum file size
	maxFileSize := 0
	for i := 2; i <= len(NodeInfoRow)+1; i++ {
		ServAgrRow := strconv.Itoa(i)
		t := "A" + ServAgrRow
		err = f.SetCellValue("MarketMatching", t, NodeInfoRow[i-2])
		k := "B" + ServAgrRow
		if fileSize, err := f.GetCellValue("MarketMatching", k); err != nil {
			log.LLvl1("Panic Raised:\n\n")
			panic(err)
		} else if intFileSize, err := strconv.Atoi(fileSize); err != nil {
			log.LLvl1("Panic Raised:\n\n")
			panic(err)
		} else if maxFileSize < intFileSize {
			maxFileSize = intFileSize
		}
	}
	bz.maxFileSize = maxFileSize
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	// fill server agreement ids
	// later we want to ad market matching transaction and compelete ServAgr info in bc
	err = f.SetCellValue("MarketMatching", "E1", "ServAgrID")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	//r := rand.New(rand.NewSource(int64(bz.SimulationSeed)))
	for i := 2; i <= len(NodeInfoRow)+1; i++ { // len(NodeInfoRow) has been used to reflecvt number of nodes 1<->1 number of agreemenets
		ServAgrRow := strconv.Itoa(i)
		cell := "E" + ServAgrRow
		//todo: use unique increasing int values for each server 1<->1 contract instead of random values
		// this int value as serverAgreementID points to the rownumber in marketmatching sheet that maintain the corresponding contract's info

		//RandomServerAgreementID := r.Int()
		//String_RandomServerAgreementID := strconv.Itoa(serverAgreementID)
		serverAgreementID := i
		String_serverAgreementID := strconv.Itoa(serverAgreementID)
		if err = f.SetCellValue("MarketMatching", cell, String_serverAgreementID); err != nil {
			log.LLvl1("Panic Raised:\n\n")
			panic(err)
		} else {
			bz.SummPoRTxs[serverAgreementID] = 0
		}
	}
	// --- sum of server agreement file size
	_ = f.NewSheet("ExtraInfo")
	if err = f.SetCellValue("ExtraInfo", "B1", "sum of file size"); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	FormulaString := "=SUM(MarketMatching!B2:B" + strconv.Itoa(len(NodeInfoRow)+1) + ")"
	err = f.SetCellFormula("ExtraInfo", "B2", FormulaString)
	if err != nil {
		log.LLvl1(err)
	}
	// --- power table sheet
	index = f.GetSheetIndex("PowerTable")
	f.SetActiveSheet(index)
	err = f.SetSheetRow("PowerTable", "B1", &NodeInfoRow)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	//err = f.SaveAs("/root/remote/mainchainbc.xlsx")
	err = f.SaveAs(bcDirectory)

	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	} else {
		log.Lvl2("bc Successfully closed")
	}
}

/* ----------------------------------------------------------------------
each round THE ROOT NODE send a msg to all nodes,
let other nodes know that the new round has started and the information they need
from blockchain to check if they are next round's leader
------------------------------------------------------------------------ */
func (bz *ChainBoost) readBCAndSendtoOthers() {
	if bz.MCRoundNumber == bz.SimulationRounds {
		log.LLvl1("ChainBoost simulation has passed the number of simulation rounds:", bz.SimulationRounds, "\n returning back to RunSimul")
		bz.DoneChainBoost <- true
		return
	}
	takenTime := time.Now()
	powers, seed := bz.readBCPowersAndSeed()
	bz.MCRoundNumber = bz.MCRoundNumber + 1
	// ---
	//bz.MCLeader.MCPLock.Lock()
	bz.MCLeader.HasLeader = false
	//bz.MCLeader.MCPLock.Unlock()
	// ---
	for _, b := range bz.Tree().List() {
		power, found := powers[b.ServerIdentity.String()]
		//log.LLvl1(power, "::", found)
		if found && !b.IsRoot() {
			err := bz.SendTo(b, &NewRound{
				Seed:        seed,
				Power:       power,
				MaxFileSize: bz.maxFileSize,
			})
			if err != nil {
				log.LLvl1(bz.Info(), "can't send new round msg to", b.Name())
				panic(err)
			} else {
				log.Lvl5(b.Name(), "recieved NewRound from", bz.TreeNode().Name(), "with maxFileSize value of:", bz.maxFileSize)
			}
		}
	}
	// detecting leader-less in next round
	go bz.startTimer(bz.MCRoundNumber)
	log.Lvl1("readBCAndSendtoOthers took:", time.Since(takenTime).String(), "for mc round number", bz.MCRoundNumber)

}

/* ----------------------------------------------------------------------*/
func (bz *ChainBoost) readBCPowersAndSeed() (minerspowers map[string]int, seed string) {
	var err error
	var rows [][]string
	var row []string
	rowNumber := 0
	minerspowers = make(map[string]int) // this convert the declared nil map to an empty map
	takenTime := time.Now()
	pwd, _ := os.Getwd()
	log.Lvl4("opening bc in:", pwd)
	//bcDirectory := strings.Split(pwd, "/build")
	bcDirectory := strings.Split(pwd, "/build")[0] + "/mainchainbc.xlsx"
	log.Lvl4("opening bc in:", bcDirectory)

	//f, err := excelize.OpenFile("/root/remote/mainchainbc.xlsx")
	f, err := excelize.OpenFile(bcDirectory)

	if err != nil {
		log.Lvl1("Panic Raised: ", err)
		panic(err)
	} else {
		log.Lvl2("bc Successfully opened")
	}
	//-------------------------------------------------------------------------------------------
	// looking for last round's seed in the round table sheet in the mainchainbc file

	/* --- RoundTable:
	i = 0 is (next) round number,
	i = 1 is (next) round seed,
	i = 2 is blockchain size (empty now, will be updated by the leader) */

	if rows, err = f.GetRows("RoundTable"); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	} else {
		u := len(rows)
		row = rows[u-1][:]
		seed = row[1]
	}

	//-------------------------------------------------------------------------------------------
	// looking for all nodes' power in the last round in the power table sheet in the mainchainbc file
	if rows, err = f.GetRows("PowerTable"); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	} else {
		u := len(rows)
		rowNumber = u - 1
		row = rows[u-1][:]
		for i, a := range bz.Roster().List { // 0 < i < len(roster)
			powerCell, er := strconv.Atoi(row[i+1])
			if er != nil {
				log.LLvl1("Panic Raised:\n\n", er)
				panic(er)
			}
			minerspowers[a.Address.String()] = powerCell
			log.Lvl4(i+1, "-th miner in roster list:", a.Address.String(), "\ngot power of", powerCell, "\nin row number:", rowNumber, "and mc round number:", bz.MCRoundNumber)
		}
	}

	// ----------------------
	//todod: check if works remove this later
	// ----------------------
	//for _, a := range bz.Tree().List(){
	//var myColumnHeader []string
	// var myColumn string
	// var myCell string
	//myColumnHeader, err = f.SearchSheet("PowerTable", a.Address.String())
	// if err != nil {
	// 	log.LLvl1("Panic Raised:\n\n")
	// 	panic(err)
	// }
	// for _, character := range myColumnHeader[0] {
	// 	if character >= 'A' && character <= 'Z' { // a-z isn't needed! just to make sure
	// 		myColumn = myColumn + string(character)
	// 	}
	// }
	// if rows, err = f.Rows("PowerTable"); err != nil {
	// 	log.LLvl1("Panic Raised:\n\n")
	// 	panic(err)
	// } else if rowNum == rowNumber-1 {
	// 	row, err = rows.Columns()
	// 	for colVal, error := range row {
	// 		if error == "" {
	// 			t, er := strconv.Atoi(colVal)
	// 			if er != nil {
	// 				log.LLvl1("Panic Raised:\n\n")
	// 				panic(er)
	// 			}
	// 			currentPower := uint64(t)
	// 			minerspowers[a.Address.String()] = currentPower
	// 		} else {
	// 			log.LLvl1("Panic Raised:\n\n")
	// 			panic(err)
	// 		}
	// }
	// myCell = myColumn + strconv.Itoa(rowNumber) //such as A2,B3,C3..
	// var p string
	// p, err = f.GetCellValue("PowerTable", myCell)
	// if err != nil {
	// 	log.LLvl1("Panic Raised:\n\n")
	// 	panic(err)
	// }
	// var t int
	// t, err = strconv.Atoi(p)
	// if err != nil {
	// 	log.LLvl1("Panic Raised:\n\n")
	// 	panic(err
	// }
	// minerspowers[a.Address.String()] = uint64(t)
	// --------   add accumulated power to recently added power in last round  ----------
	// for i := 2; i < rowNumber; i++ {
	// 	upperPowerCell := myColumn + strconv.Itoa(i)
	// 	p, err = f.GetCellValue("PowerTable", upperPowerCell)
	// 	if err != nil {
	// 		log.LLvl1("Panic Raised:\n\n")
	// 		panic(err)
	// 	}
	// 	t, er := strconv.Atoi(p)
	// 	if er != nil {
	// 		log.LLvl1("Panic Raised:\n\n")
	// 		panic(er)
	// 	}
	// 	upperPower := uint64(t)
	// 	minerspowers[a.Address.String()] = minerspowers[a.Address.String()] + upperPower
	// }
	log.Lvl1("readBCPowersAndSeed took:", time.Since(takenTime).String(), "for mc round number", bz.MCRoundNumber+1)
	return minerspowers, seed
}

/* ----------------------------------------------------------------------
	updateBC: by leader
	each round, adding one row in power table based on the information in market matching sheet,
	assuming that servers are honest and have honestly publish por for their actice (not expired) ServAgrs,
	for each storage server and each of their active ServAgr, add the stored file size to their current power
 ----------------------------------------------------------------------*/
func (bz *ChainBoost) updateBCPowerRound(LeaderName string, leader bool) {
	var err error
	var rowsRoundTable [][]string
	var rowsMarketMatching *excelize.Rows
	var row []string
	var seed string
	takenTime := time.Now()
	//rowNumber := 0

	pwd, _ := os.Getwd()
	log.Lvl4("opening bc in:", pwd)
	//bcDirectory := strings.Split(pwd, "/build")
	bcDirectory := strings.Split(pwd, "/build")[0] + "/mainchainbc.xlsx"
	log.Lvl4("opening bc in:", bcDirectory)
	//f, err := excelize.OpenFile("/root/remote/mainchainbc.xlsx")
	f, err := excelize.OpenFile(bcDirectory)
	if err != nil {
		log.Fatal("problem while opening bc: " + err.Error())
	} else {
		log.Lvl2("bc Successfully opened")
	}

	//-------------------------------------------------------------------------------------------
	// looking for last round's seed in the round table sheet in the mainchainbc file

	/* --- RoundTable:
	i = 0 is (next) round number,
	i = 1 is (next) round seed,
	i = 2 is blockchain size (empty now, will be updated by the leader) */

	if rowsRoundTable, err = f.GetRows("RoundTable"); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	} else {
		u := len(rowsRoundTable)
		row = rowsRoundTable[u-1][:]
		seed = row[1]
	}

	// --------------------------------------------------------------------
	// updating the current last row in the "BCsize" column

	// no longer nodes read from mainchainbc file, instead the round are determined by their local round number variable
	//currentRow := strconv.Itoa(rowNumber)
	//nextRow := strconv.Itoa(rowNumber + 1) //ToDoRaha: remove these, use bz.MCRoundNumber instead!

	// including header row: round 1 is on row number 2
	currentRow := strconv.Itoa(bz.MCRoundNumber + 1)
	nextRow := strconv.Itoa(bz.MCRoundNumber + 2)

	// --- bc size is inserted in take tx function
	// axisBCSize := "C" + currentRow
	// err = f.SetCellValue("RoundTable", axisBCSize, bz.MainChainBlockSize)
	// if err != nil {
	// 	log.LLvl1("Panic Raised:\n\n")
	// 	panic(err)
	// }

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
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	// --------------------------------------------------------------------
	// updating the current last row in the "miner" column
	axisMiner := "D" + currentRow
	err = f.SetCellValue("RoundTable", axisMiner, LeaderName)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	// --------------------------------------------------------------------
	// adding one row in round table (round number and seed columns)
	axisMCRoundNumber := "A" + nextRow
	err = f.SetCellValue("RoundTable", axisMCRoundNumber, bz.MCRoundNumber+1)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	// ---  next round's seed is the hash of current seed
	data := fmt.Sprintf("%v", seed)
	sha := sha256.New()
	if _, err := sha.Write([]byte(data)); err != nil {
		log.Error("Couldn't hash header:", err)
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	hash := sha.Sum(nil)
	axisSeed := "B" + nextRow
	err = f.SetCellValue("RoundTable", axisSeed, hex.EncodeToString(hash))
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	// Each round, adding one row in power table based on the information in market matching sheet,
	// assuming that servers are honest  and have honestly publish por for their actice (not expired)
	// ServAgrs,for each storage server and each of their active contracst,
	// add the stored file size to their current power
	if rowsMarketMatching, err = f.Rows("MarketMatching"); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	var ServAgrDuration, ServAgrStartedMCRoundNumber, FileSize int
	var MinerServer string
	rowNum := 0
	MinerServers := make(map[string]int)
	for rowsMarketMatching.Next() {
		rowNum++
		if rowNum == 1 {
			continue
		} else {
			row, err = rowsMarketMatching.Columns()
			if err != nil {
				log.LLvl1("Panic Raised:\n\n")
				panic(err)
			} else {
				for i, colCell := range row {
					/* --- in MarketMatching: i = 0 is Server's Info,
					i = 1 is FileSize, i=2 is ServAgrDuration,
					i=3 is MCRoundNumber, i=4 is ServAgrID, i=5 is Client's PK,
					i = 6 is ServAgrPublished */
					if i == 0 {
						MinerServer = colCell
					}
					if i == 1 {
						FileSize, err = strconv.Atoi(colCell)
						if err != nil {
							log.LLvl1("Panic Raised:\n\n")
							panic(err)
						}
					}
					if i == 2 {
						ServAgrDuration, err = strconv.Atoi(colCell)
						if err != nil {
							log.LLvl1("Panic Raised:\n\n")
							panic(err)
						}
					}
					if i == 3 {
						ServAgrStartedMCRoundNumber, err = strconv.Atoi(colCell)
						if err != nil {
							log.LLvl1("Panic Raised:\n\n")
							panic(err)
						}
					}
				}
			}
			if bz.MCRoundNumber-ServAgrStartedMCRoundNumber <= ServAgrDuration {
				MinerServers[MinerServer] = FileSize //if each server one ServAgr
			} else {
				MinerServers[MinerServer] = 0
			}
		}
	}
	//
	// ---------------------------------------------------------------------
	// --- Power Table sheet  ----------------------------------------------
	// ---------------------------------------------------------------------
	/* todo: Power has been  added without considering por tx.s not published (waiting in queue yet)
	=> fix it: use TXissued column (first set it to 2 when taken and second change its name to TXonQ), so if
	contract is publlished (1) but TxonQ is taken (2) then add power  */

	index := f.GetSheetIndex("PowerTable")
	f.SetActiveSheet(index)
	var PowerInfoRow []int
	for _, a := range bz.Roster().List {
		PowerInfoRow = append(PowerInfoRow, MinerServers[a.Address.String()])
	}
	axisMCRoundNumber = "B" + currentRow
	err = f.SetSheetRow("PowerTable", axisMCRoundNumber, &PowerInfoRow)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	// --------------------------------------------------------------------
	// adding current round's round number
	axisMCRoundNumber = "A" + currentRow
	err = f.SetCellValue("PowerTable", axisMCRoundNumber, bz.MCRoundNumber)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	// ----
	//err = f.SaveAs("/root/remote/mainchainbc.xlsx")
	err = f.SaveAs(bcDirectory)

	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	} else {
		log.Lvl2("bc Successfully closed")
		log.Lvl1("updateBCPowerRound took:", time.Since(takenTime).String(), "for mc round number", bz.MCRoundNumber)

	}
}

/* ----------------------------------------------------------------------
    updateBC: this is a connection between first layer of blockchain - ROOT NODE - on the second layer - xlsx file -
------------------------------------------------------------------------ */
func (bz *ChainBoost) updateMainChainBCTransactionQueueCollect() {
	takenTime := time.Now()
	var err error
	pwd, _ := os.Getwd()
	log.Lvl4("opening bc in:", pwd)
	//bcDirectory := strings.Split(pwd, "/build")
	bcDirectoryMC := strings.Split(pwd, "/build")[0] + "/mainchainbc.xlsx"
	log.Lvl4("opening bc in:", bcDirectoryMC)
	//f, err := excelize.OpenFile("/root/remote/mainchainbc.xlsx")
	f, err := excelize.OpenFile(bcDirectoryMC)
	if err != nil {
		log.Fatal("problem while opening bc: " + err.Error())
	} else {
		log.Lvl2("bc Successfully opened")
	}
	cols, err := f.GetCols("MarketMatching")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	/* --- in MarketMatching each row has the following columns:
	cols[0] is Server's Info,
	cols[1] is FileSize,
	cols[2] is ServAgrDuration,
	cols[3] is ServAgrStartedMCRoundNumber,
	cols[4] is ServAgrID,
	cols[5] is ServAgrPublished
	cols[6] is ServAgrTxsIssued */
	ServAgrDurationCol := cols[2][1:]
	ServAgrStartedMCRoundNumberCol := cols[3][1:]
	ServAgrPublishedCol := cols[5][1:]
	ServAgrTxsIssuedCol := cols[6][1:]

	// ----------------------------------------------------------------------
	// ------ add 5 types of transactions into transaction queue sheet -----
	// ----------------------------------------------------------------------
	/* each transaction has the following column stored on the transaction queue sheet:
	0) name
	1) size
	2) time
	3) issuedMCRoundNumber
	4) ServAgrId */
	//-----------------------------------------

	var newTransactionRowMC [5]string
	sMC := make([]interface{}, len(newTransactionRowMC)) //ToDoRaha:  check this out later: https://stackoverflow.com/questions/23148812/whats-the-meaning-of-interface/23148998#23148998
	// this part can be moved to protocol initialization
	var PorTxSize, ServAgrProposeTxSize, PayTxSize, StoragePayTxSize, ServAgrCommitTxSize uint32
	PorTxSize, ServAgrProposeTxSize, PayTxSize, StoragePayTxSize, ServAgrCommitTxSize = blockchain.TransactionMeasurement(bz.SectorNumber, bz.SimulationSeed)
	var ServAgrDuration, ServAgrStartedMCRoundNumber, ServAgrPublished int
	colNum := 1
	rowNumMC := 1
	var numOfPoRTxsMC, numOfServAgrProposeTxs, numOfStoragePaymentTxs, numOfServAgrCommitTxs, numOfRegularPaymentTxs int

	// -------------------------------------------------------------------------------
	//        -------- updateSideChainBCTransactionQueueCollect  --------
	// -------------------------------------------------------------------------------
	var newTransactionRowSC [6]string
	sSC := make([]interface{}, len(newTransactionRowSC))
	numOfPoRTxsSC := 1
	rowNumSC := 1
	// -------------------------------------------------------------------------------
	log.Lvl4("opening sidechainbc in:", pwd)
	bcDirectorySC := strings.Split(pwd, "/build")[0] + "/sidechainbc.xlsx"
	log.Lvl4("opening sidechainbc in:", bcDirectorySC)
	f1, err := excelize.OpenFile(bcDirectorySC)
	if err != nil {
		log.Fatal("problem while opening sidechainbc: " + err.Error())
	} else {
		log.Lvl2("sidechainbc Successfully opened")
	}
	// -------------------------------------------------------------------------------
	// end of side chain operations
	// -------------------------------------------------------------------------------

	streamWriterMC, err := f.NewStreamWriter("FirstQueue")
	if err != nil {
		log.Fatal("problem while opening streamWriter: " + err.Error())
	} else {
		log.Lvl2("mainchainbc streamWriter opened")
	}
	// ------ add Header Row on top of the stream (first queue in main chain bc) ----------
	rowsMCFirstQ, err := f.GetRows("FirstQueue")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	} else {
		row := rowsMCFirstQ[0][:]
		for i, v := range row {
			sMC[i] = v
		}
		cell, _ := excelize.CoordinatesToCellName(colNum, rowNumMC)
		if err := streamWriterMC.SetRow(cell, sMC); err != nil {
			log.LLvl1("Panic Raised:\n\n")
			panic(err)
		} else {
			rowNumMC++
		}
	}

	// -------------------------------------------------------------------------------
	//        -------- updateSideChainBCTransactionQueueCollect  --------
	// -------------------------------------------------------------------------------
	streamWriterSC, err := f1.NewStreamWriter("FirstQueue")
	if err != nil {
		log.Fatal("problem while opening streamWriter: " + err.Error())
	} else {
		log.Lvl2("sidechainbc streamWriter opened")
	}
	// ------ add Header Row on top of the stream (first queue in side chain bc) ----------
	rowsSCFirstQ, err := f1.GetRows("FirstQueue")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	} else {
		row := rowsSCFirstQ[0][:]
		for i, v := range row {
			sSC[i] = v
		}
		cell, _ := excelize.CoordinatesToCellName(colNum, colNum)
		if err := streamWriterSC.SetRow(cell, sSC); err != nil {
			log.LLvl1("Panic Raised:\n\n")
			panic(err)
		} else {
			rowNumSC++
		}
	}
	// -------------------------------------------------------------------------------
	// end of side chain operations
	// -------------------------------------------------------------------------------

	// --- check for contracts states and add appropriate tx row on top of the stream ----
	for i := range bz.Roster().List {
		ServAgrDuration, err = strconv.Atoi(ServAgrDurationCol[i])
		if err != nil {
			log.LLvl1("Panic Raised:\n\n")
			panic(err)
		}
		ServAgrStartedMCRoundNumber, err = strconv.Atoi(ServAgrStartedMCRoundNumberCol[i])
		if err != nil {
			log.LLvl1("Panic Raised:\n\n")
			panic(err)
		}
		ServAgrPublished, err = strconv.Atoi(ServAgrPublishedCol[i]) //ServAgrPublishedCol is starting from index 1
		if err != nil {
			log.LLvl1("Panic Raised:\n\n", err)
			panic(err)
		}
		ServAgrTxsIssued, err := strconv.Atoi(ServAgrTxsIssuedCol[i])
		if err != nil {
			log.LLvl1("Panic Raised:\n\n", err)
			panic(err)
		}
		//-------------------------------------------------------------------
		// when the servic agreement is inactive:
		//-------------------------------------------------------------------
		if ServAgrPublished == 0 && ServAgrTxsIssued == 0 {
			// Add TxServAgrPropose
			newTransactionRowMC[2] = time.Now().Format(time.RFC3339)
			newTransactionRowMC[3] = strconv.Itoa(bz.MCRoundNumber)
			newTransactionRowMC[0] = "TxServAgrPropose"
			newTransactionRowMC[1] = strconv.Itoa(int(ServAgrProposeTxSize))
			newTransactionRowMC[4] = strconv.Itoa(i + 2) // corresponding ServAgr id
			// ------
			for i, v := range newTransactionRowMC {
				sMC[i] = v
			}
			// ------ add a TxServAgrPropose row on top of the stream ----------
			cell, _ := excelize.CoordinatesToCellName(colNum, rowNumMC)
			if err := streamWriterMC.SetRow(cell, sMC); err != nil {
				log.LLvl1("Panic Raised:\n\n")
				panic(err)
			} else {
				numOfServAgrProposeTxs++
				rowNumMC++
			}
			/* second row added in case of having the first row to be ServAgr propose tx which then we will add
			ServAgr commit tx right away.
			warning: Just in one case it may cause irrational statistics which doesn’t worth taking care of!
			when a propose ServAgr tx is added to a block which causes the ServAgr to become active but
			the commit ServAgr transaction is not yet! */

			// Add TxServAgrCommit
			newTransactionRowMC[0] = "TxServAgrCommit"
			newTransactionRowMC[1] = strconv.Itoa(int(ServAgrCommitTxSize))
			newTransactionRowMC[4] = strconv.Itoa(i + 2) // corresponding ServAgr id
			//--
			for i, v := range newTransactionRowMC {
				sMC[i] = v
			}
			// ------ add a TxServAgrCommit row on top of the stream ----------
			cell, _ = excelize.CoordinatesToCellName(colNum, rowNumMC)
			if err := streamWriterMC.SetRow(cell, sMC); err != nil {
				log.LLvl1("Panic Raised:\n\n")
				panic(err)
			} else {
				numOfServAgrCommitTxs++
				rowNumMC++
			}
			// --------------------------------------
			// Set ServAgrTxsIssued to True
			// --------------------------------------
			// i start from index 0, serverAgreementID strats from index 2 which is its rownumber
			issuedCellMarketMatching := "G" + strconv.Itoa(i+2)
			err = f.SetCellValue("MarketMatching", issuedCellMarketMatching, 1)
			if err != nil {
				log.LLvl1("Panic Raised:\n\n")
				panic(err)
			} else {
				log.Lvl3("after issuing contract propose and commit txs, set issued cell to 1")
			}

			//-------------------------------------------------------------------
			// when the servic agreement expires (for now we pay for the service just here):
			//-------------------------------------------------------------------
		} else if bz.MCRoundNumber-ServAgrStartedMCRoundNumber > ServAgrDuration && ServAgrPublished == 1 && ServAgrTxsIssued == 1 {
			// Set ServAgrPublished AND ServAgrTxsIssued to false
			// --------------------------------------
			// i start from index 0, serverAgreementID strats from index 2 which is its rownumber
			publishedCellMarketMatching := "F" + strconv.Itoa(i+2)
			err = f.SetCellValue("MarketMatching", publishedCellMarketMatching, 0)
			if err != nil {
				log.LLvl1("Panic Raised:\n\n")
				panic(err)
			}
			// i start from index 0, serverAgreementID strats from index 2 which is its rownumber
			issuedCellMarketMatching := "G" + strconv.Itoa(i+2)
			err = f.SetCellValue("MarketMatching", issuedCellMarketMatching, 0)
			if err != nil {
				log.LLvl1("Panic Raised:\n\n")
				panic(err)
			}
			// --------------------------------------
			// Add TxStoragePayment
			// --------------------------------------
			newTransactionRowMC[2] = time.Now().Format(time.RFC3339)
			newTransactionRowMC[3] = strconv.Itoa(bz.MCRoundNumber)
			newTransactionRowMC[0] = "TxStoragePayment"
			newTransactionRowMC[1] = strconv.Itoa(int(StoragePayTxSize))
			newTransactionRowMC[4] = strconv.Itoa(i + 2)
			// ------
			for i, v := range newTransactionRowMC {
				sMC[i] = v
			}
			// ------ add a TxStoragePayment row on top of the stream ----------
			cell, _ := excelize.CoordinatesToCellName(colNum, rowNumMC)
			if err := streamWriterMC.SetRow(cell, sMC); err != nil {
				log.LLvl1("Panic Raised:\n\n")
				panic(err)
			} else {
				numOfStoragePaymentTxs++
				rowNumMC++
			}
			//-------------------------------------------------------------------
			// when simStat == 2 it means that side chain is
			// running and por tx.s go to side chain queue
			// when the ServAgr is not expired => Add TxPor
			//-------------------------------------------------------------------
		} else if ServAgrPublished == 1 && bz.MCRoundNumber-ServAgrStartedMCRoundNumber <= ServAgrDuration && bz.SimState == 1 {
			// --- Add TxPor
			newTransactionRowMC[2] = time.Now().Format(time.RFC3339)
			newTransactionRowMC[3] = strconv.Itoa(bz.MCRoundNumber)
			newTransactionRowMC[0] = "TxPor"
			newTransactionRowMC[1] = strconv.Itoa(int(PorTxSize))
			newTransactionRowMC[4] = strconv.Itoa(i + 2)
			// ------
			for i, v := range newTransactionRowMC {
				sMC[i] = v
			}
			// ------ add a TxPor row on top of the stream ----------
			cell, _ := excelize.CoordinatesToCellName(colNum, rowNumMC)
			if err := streamWriterMC.SetRow(cell, sMC); err != nil {
				log.LLvl1("Panic Raised:\n\n")
				panic(err)
			} else {
				numOfPoRTxsMC++
				rowNumMC++
			}
		} else if ServAgrPublished == 1 && bz.MCRoundNumber-ServAgrStartedMCRoundNumber <= ServAgrDuration && bz.SimState == 2 {
			// -------------------------------------------------------------------------------
			//        -------- updateSideChainBCTransactionQueueCollect  --------
			// -------------------------------------------------------------------------------
			// --- check for eligible contracts and add a por tx row on top of the stream ----
			// ServAgr is not expired => Add TxPor
			newTransactionRowSC[2] = time.Now().Format(time.RFC3339)
			// 50008 means 50000 + 8 which means epoch number 5 scround number 8 //todo: is it still on?
			newTransactionRowSC[3] = strconv.Itoa(bz.SCRoundNumber)
			newTransactionRowSC[0] = "TxPor"
			newTransactionRowSC[1] = strconv.Itoa(int(PorTxSize))
			newTransactionRowSC[4] = strconv.Itoa(i)
			newTransactionRowSC[5] = strconv.Itoa(bz.MCRoundNumber)
			for i, v := range newTransactionRowSC {
				sSC[i] = v
			}
			// ------ add a por tx row on top of the stream ----------
			cell, _ := excelize.CoordinatesToCellName(colNum, rowNumSC)
			if err := streamWriterSC.SetRow(cell, sSC); err != nil {
				log.LLvl1("Panic Raised:\n\n")
				panic(err)
			} else {
				numOfPoRTxsSC++
				rowNumSC++
			}
			// -------------------------------------------------------------------------------
			// end of side chain operations
			// -------------------------------------------------------------------------------
		}
	}

	rowNumMC--
	// ------ add previous txs row in the first mc queue to the stream ----------
	for i, row := range rowsMCFirstQ { //i starts from index 0
		if i != 0 { // index 0 is the header row
			for t, v := range row {
				sMC[t] = v
			}
			cell, _ := excelize.CoordinatesToCellName(colNum, i+rowNumMC)
			if err := streamWriterMC.SetRow(cell, sMC); err != nil {
				log.LLvl1("Panic Raised:\n\n")
				panic(err)
			}
		}
	}
	/* from the documentation: "after set rows, you must call the Flush method to
	end the streaming writing process and ensure that the order of line numbers is ascending,
	the common API and stream API can't be work mixed to writing data on the worksheets." */
	if err := streamWriterMC.Flush(); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	// reset rowNumMC
	rowNumMC = 1
	// -------------------------------------------------------------------------------
	// ------ add payment transactions into transaction queue payment sheet
	// -------------------------------------------------------------------------------
	rand.Seed(int64(bz.SimulationSeed))
	// avoid having zero regular payment txs
	var numberOfRegPay int
	for numberOfRegPay == 0 {
		numberOfRegPay = rand.Intn(bz.NumberOfPayTXsUpperBound)
	}
	log.LLvl1("Number of regular payment transactions in mc round number", bz.MCRoundNumber, "is", numberOfRegPay)
	// ----------------------------------------------------------
	streamWriterMC, err = f.NewStreamWriter("SecondQueue")
	if err != nil {
		log.Fatal("problem while opening second queue streamWriter: " + err.Error())
	} else {
		log.Lvl2("sidechainbc streamWriter opened")
	}
	// ------ add Header Row on top of the stream (second queue in main chain bc) ----------
	rowsMCSecondQ, err := f.GetRows("SecondQueue")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	} else {
		row := rowsMCSecondQ[0][:]
		for i, v := range row {
			sMC[i] = v
		}
		cell, _ := excelize.CoordinatesToCellName(colNum, colNum)
		if err := streamWriterMC.SetRow(cell, sMC); err != nil {
			log.LLvl1("Panic Raised:\n\n")
			panic(err)
		} else {
			rowNumMC++
		}
	}
	// -------------------------------------------------------------------
	// ------ add payment transactions into second queue stream writer
	// -------------------------------------------------------------------
	/* each transaction has the following column stored on the transaction queue payment sheet:
	0) size
	1) time
	2) issuedMCRoundNumber */

	newTransactionRowMC[0] = strconv.Itoa(int(PayTxSize))
	newTransactionRowMC[1] = time.Now().Format(time.RFC3339)
	newTransactionRowMC[2] = strconv.Itoa(bz.MCRoundNumber)
	newTransactionRowMC[3] = ""
	newTransactionRowMC[4] = ""
	for i, v := range newTransactionRowMC {
		sMC[i] = v
	}
	for i := 1; i <= numberOfRegPay; i++ {
		cell, _ := excelize.CoordinatesToCellName(colNum, rowNumMC)
		if err := streamWriterMC.SetRow(cell, sMC); err != nil {
			log.LLvl1("Panic Raised:\n\n")
			panic(err)
		} else {
			numOfRegularPaymentTxs++
			rowNumMC++
		}
	}
	rowNumMC--
	// ------ add previous reg pay tx row in the second queue to the stream ----------
	for i, row := range rowsMCSecondQ { //i starts from index 0
		if i != 0 { // index 0 is the header row
			for t, v := range row {
				sMC[t] = v
			}
			cell, _ := excelize.CoordinatesToCellName(colNum, i+rowNumMC)
			if err := streamWriterMC.SetRow(cell, sMC); err != nil {
				log.LLvl1("Panic Raised:\n\n")
				panic(err)
			}
		}
	}
	/* from the documentation: "after set rows, you must call the Flush method to
	end the streaming writing process and ensure that the order of line numbers is ascending,
	the common API and stream API can't be work mixed to writing data on the worksheets." */
	if err := streamWriterMC.Flush(); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	// -------------------------------------------------------------------------------
	//        -------- updateSideChainBCTransactionQueueCollect  --------
	// -------------------------------------------------------------------------------
	rowNumSC--
	numOfPoRTxsSC--
	// ------ add previous por tx row in the first queue to the stream ----------
	for i, row := range rowsSCFirstQ { //i starts from index 0
		for t, v := range row {
			sSC[t] = v
		}
		if i != 0 { // index 0 is the header row
			cell, _ := excelize.CoordinatesToCellName(colNum, i+rowNumSC)
			if err := streamWriterSC.SetRow(cell, sSC); err != nil {
				log.LLvl1("Panic Raised:\n\n")
				panic(err)
			}
		}
	}
	/* from the documentation: "after set rows, you must call the Flush method to
	end the streaming writing process and ensure that the order of line numbers is ascending,
	the common API and stream API can't be work mixed to writing data on the worksheets." */
	if err := streamWriterSC.Flush(); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	// -------------------------------------------------------------------------------
	err = f.SaveAs(bcDirectoryMC)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	} else {
		log.Lvl2("mc bc Successfully closed")
		log.Lvl1(bz.Name(), " finished collecting new transactions to mainchain queues in mc round number ", bz.MCRoundNumber, "in total:\n ",
			numOfPoRTxsMC, "numOfPoRTxsMC\n", numOfServAgrProposeTxs, "numOfServAgrProposeTxs\n",
			numOfStoragePaymentTxs, "numOfStoragePaymentTxs\n", numOfServAgrCommitTxs, "numOfServAgrCommitTxs\n",
			numOfRegularPaymentTxs, "numOfRegularPaymentTxs added to the main chain queus")
		log.Lvl1("Collecting mc tx.s took:", time.Since(takenTime).String())
	}
	// -------------------------------------------------------------------------------
	if err = f1.SaveAs(bcDirectorySC); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	} else {
		log.Lvl2("sc bc successfully closed")
		log.Lvl4(bz.Name(), "Final result SC: finished collecting new transactions to side chain queue in sc round number ", bz.SCRoundNumber)
		//log.Lvl1("updateSideChainBCTransactionQueueCollect took:", time.Since(takenTime).String())
		log.Lvl1(numOfPoRTxsSC, "TxPor added to queue in sc round number: ", bz.SCRoundNumber)
	}
	// -------------------------------------------------------------------------------
	// end of side chain operations
	// -------------------------------------------------------------------------------
}

/* ----------------------------------------------------------------------
    updateBC: this is a connection between first layer of blockchain - ROOT NODE - and the second layer - xlsx file -
------------------------------------------------------------------------ */
func (bz *ChainBoost) updateMainChainBCTransactionQueueTake() {
	var err error
	var rows [][]string
	// --- reset
	bz.FirstQueueWait = 0
	bz.SecondQueueWait = 0

	pwd, _ := os.Getwd()
	log.Lvl4("opening bc in:", pwd)
	//bcDirectory := strings.Split(pwd, "/build")
	bcDirectory := strings.Split(pwd, "/build")[0] + "/mainchainbc.xlsx"
	log.Lvl4("opening bc in:", bcDirectory)
	//f, err := excelize.OpenFile("/root/remote/mainchainbc.xlsx")
	f, err := excelize.OpenFile(bcDirectory)
	if err != nil {
		log.Fatal("problem while opening bc: " + err.Error())
	} else {
		log.Lvl2("bc Successfully opened")
	}

	var accumulatedTxSize, txsize int
	blockIsFull := false
	regPayshareIsFull := false
	NextRow := strconv.Itoa(bz.MCRoundNumber + 2)

	axisNumRegPayTx := "E" + NextRow

	axisQueue2IsFull := "N" + NextRow
	axisQueue1IsFull := "O" + NextRow

	numberOfRegPayTx := 0
	BlockSizeMinusTransactions := blockchain.BlockMeasurement()
	var takenTime time.Time
	regPayTxShare := (bz.PercentageTxPay) * (bz.MainChainBlockSize - BlockSizeMinusTransactions)
	/* -----------------------------------------------------------------------------
		-- take regular payment transactions from sheet: SecondQueue
	----------------------------------------------------------------------------- */
	if rows, err = f.GetRows("SecondQueue"); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	takenTime = time.Now()
	for i := len(rows); i > 1 && !regPayshareIsFull; i-- {
		row := rows[i-1][:]
		/* each transaction has the following column stored on the Transaction Queue Payment sheet:
		0) size
		1) time
		2) issuedMCRoundNumber */

		// ----------------
		//todo: check if works fine, remove this part later
		// ----------------
		// for j, colCell := range row {
		// 	if j == 0 {
		// ----------------

		if txsize, err = strconv.Atoi(row[0]); err != nil {
			log.LLvl1("Panic Raised:\n\n")
			panic(err)
		} else if 100*(accumulatedTxSize+txsize) <= regPayTxShare {
			accumulatedTxSize = accumulatedTxSize + txsize

			// row[1] is transaction's collected time
			// if TakeTime, err = time.Parse(time.RFC3339, row[1]); err != nil {
			// 	log.LLvl1("Panic Raised:\n\n")
			// 	panic(err)
			// }
			//bz.SecondQueueWait = bz.SecondQueueWait + int(time.Now().Sub(TakeTime).Seconds())

			// x is row[2] which is the MCRound Number that the transaction has been issued
			var x int
			if x, err = strconv.Atoi(row[2]); err != nil {
				log.LLvl1("Panic Raised:\n\n")
				panic(err)
			} else {
				bz.SecondQueueWait = bz.SecondQueueWait + bz.MCRoundNumber - x
			}

			numberOfRegPayTx++
			/* transaction name in transaction queue payment is just "TxPayment"
			the transactions are just removed from queue and their size are added to included transactions' size in block */
			f.RemoveRow("SecondQueue", i)
			log.Lvl2("a regular payment transaction added to block number", bz.MCRoundNumber, " from the queue")
		} else {
			regPayshareIsFull = true
			log.Lvl1("Final result MC: \n regular  payment share is full!")
			f.SetCellValue("RoundTable", axisQueue2IsFull, 1)
			break
		}
		//}
		//}
	}
	log.Lvl1("reg pay tx.s taking took:", time.Since(takenTime).String())
	allocatedBlockSizeForRegPayTx := accumulatedTxSize
	otherTxsShare := bz.MainChainBlockSize - BlockSizeMinusTransactions - allocatedBlockSizeForRegPayTx
	/* -----------------------------------------------------------------------------
		 -- take 5 types of transactions from sheet: FirstQueue
	----------------------------------------------------------------------------- */
	// reset variables
	txsize = 0
	blockIsFull = false
	accumulatedTxSize = 0
	// new variables for first queue
	//var ServAgrIdCellMarketMatching []string

	numberOfPoRTx := 0
	numberOfStoragePayTx := 0
	numberOfServAgrProposeTx := 0
	numberOfServAgrCommitTx := 0
	numberOfSyncTx := 0

	axisBlockSize := "C" + NextRow
	axisNumPoRTx := "F" + NextRow
	axisNumStoragePayTx := "G" + NextRow
	axisNumServAgrProposeTx := "H" + NextRow
	axisNumServAgrCommitTx := "I" + NextRow
	axisNumSyncTx := "P" + NextRow
	axisTotalTxsNum := "K" + NextRow
	axisAveFirstQueueWait := "L" + NextRow
	axisAveSecondQueueWait := "M" + NextRow

	takenTime = time.Now()

	if rows, err = f.GetRows("FirstQueue"); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	for i := len(rows); i > 1 && !blockIsFull; i-- {
		row := rows[i-1][:]
		/* each transaction has the following column stored on the transaction queue sheet:
		0) name
		1) size
		2) time
		3) issuedMCRoundNumber
		4) ServAgrId (In case that the transaction type is a Sync transaction,
		the 4th column will be the number of por transactions summerized in
		the sync tx) */

		if txsize, err = strconv.Atoi(row[1]); err != nil {
			log.LLvl1("Panic Raised:\n\n")
			panic(err)
		} else if accumulatedTxSize+txsize <= otherTxsShare {
			accumulatedTxSize = accumulatedTxSize + txsize
			/* transaction name in transaction queue can be
			"TxServAgrCommit",
			"TxServAgrPropose",
			"TxStoragePayment", or
			"TxPor"
			in case of "TxServAgrCommit":
			1) The corresponding ServAgr in marketmatching should be updated to published //ToDoRaha: replace the word "published" with "active"
			2) set start round number to current round
			other transactions are just removed from queue and their size are added to included transactions' size in block */

			switch row[0] {
			case "TxServAgrCommit":
				/* when tx TxServAgrCommit left queue:
				1) set ServAgrPublished to True
				2) set start round number to current round */
				cid := row[4]

				// ------------------------------
				// todo: check and remove it later if works fine
				// ------------------------------
				// if ServAgrIdCellMarketMatching, err = f.SearchSheet("MarketMatching", cid); err != nil {
				// 	log.LLvl1("Panic Raised:\n\n")
				// 	panic(err)
				// } else {
				//publishedCellMarketMatching := "F" + ServAgrIdCellMarketMatching[0][1:]
				// ------------------------------

				// Raha: cid is the unique increasing int contract id which points to the row number of corresponding contract in the market matching sheet
				publishedCellMarketMatching := "F" + cid
				err = f.SetCellValue("MarketMatching", publishedCellMarketMatching, 1)
				if err != nil {
					log.LLvl1("Panic Raised:\n\n")
					panic(err)
				} else {
					//startRoundCellMarketMatching := "D" + ServAgrIdCellMarketMatching[0][1:]
					startRoundCellMarketMatching := "D" + cid
					err = f.SetCellValue("MarketMatching", startRoundCellMarketMatching, bz.MCRoundNumber)
					if err != nil {
						log.LLvl1("Panic Raised:\n\n")
						panic(err)
					} else {
						log.Lvl4("a TxServAgrCommit tx added to block number", bz.MCRoundNumber, " from the queue")
						numberOfServAgrCommitTx++
					}
				}
				//}
			case "TxStoragePayment":
				log.Lvl4("a TxStoragePayment tx added to block number", bz.MCRoundNumber, " from the queue")
				numberOfStoragePayTx++
				// && bz.SimState == 1 is for backup check - the first condition shouldn't be true if the second one isn't
			case "TxPor":
				if bz.SimState == 1 {
					log.Lvl4("a por tx added to block number", bz.MCRoundNumber, " from the queue")
					numberOfPoRTx++
				}
			case "TxServAgrPropose":
				log.Lvl4("a TxServAgrPropose tx added to block number", bz.MCRoundNumber, " from the queue")
				numberOfServAgrProposeTx++
			case "TxSync":
				log.Lvl4("a sync tx added to block number", bz.MCRoundNumber, " from the queue")
				numberOfSyncTx++
				PoRTx, _ := strconv.Atoi(row[4]) //In case that the transaction type is a Sync transaction,
				// the 4th column will be the number of por transactions summerized in
				// the sync tx
				numberOfPoRTx = numberOfPoRTx + PoRTx
			default:
				log.Lvl1("Panic Raised:\n\n")
				panic("the type of transaction in the queue is un-defined")
			}

			// when performance was being measured based on time!
			// if TakeTime, err = time.Parse(time.RFC3339, row[2]); err != nil {
			// 	log.LLvl1("Panic Raised:\n\n")
			// 	panic(err)
			// }
			//bz.FirstQueueWait = bz.FirstQueueWait + int(time.Now().Sub(TakeTime).Seconds())

			// row[3] is the MCRound Number that the transaction has been issued
			if x, err := strconv.Atoi(row[3]); err != nil {
				log.LLvl1("Panic Raised:\n\n")
				panic(err)
			} else {
				bz.FirstQueueWait = bz.FirstQueueWait + bz.MCRoundNumber - x
			}
			// remove transaction from the top of the queue (older ones)
			f.RemoveRow("FirstQueue", i)
			log.Lvl2("1 other tx is taken.")
		} else {
			blockIsFull = true
			log.Lvl1("Final result MC: \n block is full! ")
			f.SetCellValue("RoundTable", axisQueue1IsFull, 1)
			break
		}
	}
	log.Lvl1("other tx.s taking took:", time.Since(takenTime).String())

	f.SetCellValue("RoundTable", axisNumPoRTx, numberOfPoRTx)
	f.SetCellValue("RoundTable", axisNumStoragePayTx, numberOfStoragePayTx)
	f.SetCellValue("RoundTable", axisNumServAgrProposeTx, numberOfServAgrProposeTx)
	f.SetCellValue("RoundTable", axisNumServAgrCommitTx, numberOfServAgrCommitTx)
	f.SetCellValue("RoundTable", axisNumSyncTx, numberOfSyncTx)

	log.Lvl2("In total in mc round number ", bz.MCRoundNumber,
		"\n number of published PoR transactions is", numberOfPoRTx,
		"\n number of published Storage payment transactions is", numberOfStoragePayTx,
		"\n number of published Propose ServAgr transactions is", numberOfServAgrProposeTx,
		"\n number of published Commit ServAgr transactions is", numberOfServAgrCommitTx,
		"\n number of published regular payment transactions is", numberOfRegPayTx,
	)
	TotalNumTxsInBothQueue := numberOfPoRTx + numberOfStoragePayTx + numberOfServAgrProposeTx + numberOfServAgrCommitTx + numberOfRegPayTx
	TotalNumTxsInFirstQueue := numberOfPoRTx + numberOfStoragePayTx + numberOfServAgrProposeTx + numberOfServAgrCommitTx

	//-- accumulated block size
	// --- total throughput
	f.SetCellValue("RoundTable", axisNumRegPayTx, numberOfRegPayTx)
	f.SetCellValue("RoundTable", axisBlockSize, accumulatedTxSize+allocatedBlockSizeForRegPayTx+BlockSizeMinusTransactions)
	f.SetCellValue("RoundTable", axisTotalTxsNum, TotalNumTxsInBothQueue)
	if TotalNumTxsInFirstQueue != 0 {
		f.SetCellValue("RoundTable", axisAveFirstQueueWait, float64(bz.FirstQueueWait)/float64(TotalNumTxsInFirstQueue))
	}
	if numberOfRegPayTx != 0 {
		f.SetCellValue("RoundTable", axisAveSecondQueueWait, float64(bz.SecondQueueWait)/float64(numberOfRegPayTx))
	}

	log.Lvl2("Final result MC: \n Block size allocation:\n", allocatedBlockSizeForRegPayTx,
		" for regular payment txs,\n and ", accumulatedTxSize, " for other types of txs")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	log.Lvl1("In total in mc round number ", bz.MCRoundNumber,
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
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	FormulaString = "=SUM(RoundTable!C2:C" + NextRow + ")"
	err = f.SetCellFormula("OverallEvaluation", axisBCSize, FormulaString)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	FormulaString = "=SUM(RoundTable!E2:E" + NextRow + ")"
	err = f.SetCellFormula("OverallEvaluation", axisOverallRegPayTX, FormulaString)
	if err != nil {
		log.LLvl1(err)
	}
	FormulaString = "=SUM(RoundTable!F2:F" + NextRow + ")"
	err = f.SetCellFormula("OverallEvaluation", axisOverallPoRTX, FormulaString)
	if err != nil {
		log.LLvl1(err)
	}
	FormulaString = "=SUM(RoundTable!G2:G" + NextRow + ")"
	err = f.SetCellFormula("OverallEvaluation", axisOverallStorjPayTX, FormulaString)
	if err != nil {
		log.LLvl1(err)
	}
	FormulaString = "=SUM(RoundTable!H2:H" + NextRow + ")"
	err = f.SetCellFormula("OverallEvaluation", axisOverallCntPropTX, FormulaString)
	if err != nil {
		log.LLvl1(err)
	}
	FormulaString = "=SUM(RoundTable!I2:I" + NextRow + ")"
	err = f.SetCellFormula("OverallEvaluation", axisOverallCntComTX, FormulaString)
	if err != nil {
		log.LLvl1(err)
	}
	FormulaString = "=AVERAGE(RoundTable!L2:L" + NextRow + ")"
	err = f.SetCellFormula("OverallEvaluation", axisAveWaitOtherTx, FormulaString)
	if err != nil {
		log.LLvl1(err)
	}
	FormulaString = "=AVERAGE(RoundTable!M2:M" + NextRow + ")"
	err = f.SetCellFormula("OverallEvaluation", axisOveralAveWaitRegPay, FormulaString)
	if err != nil {
		log.LLvl1(err)
	}
	FormulaString = "=SUM(RoundTable!O2:O" + NextRow + ")"
	err = f.SetCellFormula("OverallEvaluation", axisOverallBlockSpaceFull, FormulaString)
	if err != nil {
		log.LLvl1(err)
	}
	// ----
	//err = f.SaveAs("/root/remote/mainchainbc.xlsx")
	err = f.SaveAs(bcDirectory)

	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	} else {
		log.Lvl2("bc Successfully closed")
		log.Lvl1(bz.Name(), " Finished taking transactions from queue (FIFO) into new block in mc round number ", bz.MCRoundNumber)
	}
}

func (bz *ChainBoost) syncMainChainBCTransactionQueueCollect() (blocksize int) {
	takenTime := time.Now()
	pwd, _ := os.Getwd()
	log.Lvl4("opening bc in:", pwd)
	//bcDirectory := strings.Split(pwd, "/build")
	bcDirectory := strings.Split(pwd, "/build")[0] + "/mainchainbc.xlsx"
	log.Lvl4("opening bc in:", bcDirectory)
	//f, err := excelize.OpenFile("/root/remote/mainchainbc.xlsx")
	f, err := excelize.OpenFile(bcDirectory)
	if err != nil {
		log.Fatal("problem while opening bc: " + err.Error())
	} else {
		log.Lvl2("bc Successfully opened")
	}

	// ----------------------------------------------------------------------
	// ------ adding sync transaction into transaction queue sheet -----
	// ----------------------------------------------------------------------
	/* each transaction has the following column stored on the transaction queue sheet:
	0) name
	1) size
	2) time
	3) issuedMCRoundNumber
	4) ServAgrId */
	var newTransactionRow [5]string
	s := make([]interface{}, len(newTransactionRow)) //ToDoRaha: :  check this out later: https://stackoverflow.com/questions/23148812/whats-the-meaning-of-interface/23148998#23148998

	newTransactionRow[0] = "TxSync"
	newTransactionRow[2] = time.Now().Format(time.RFC3339)
	newTransactionRow[3] = strconv.Itoa(bz.MCRoundNumber)
	/* In case that the transaction type is a Sync transaction,
	the 4th column will be the number of por transactions summerized in
	the sync tx */

	var SummTxNum int
	totalPoR := 0
	for i, v := range bz.SummPoRTxs {
		totalPoR = totalPoR + v
		if v != 0 {
			SummTxNum = SummTxNum + 1 // counting the number of active contracts to measure the length of summary tx
		}
		bz.SummPoRTxs[i] = 0
	}
	//measuring summary block size
	SummaryBlockSizeMinusTransactions, _ := blockchain.SCBlockMeasurement()
	// --------------- adding bls signature size  -----------------
	log.Lvl4("Size of bls signature:", len(bz.SCSig))
	SummaryBlockSizeMinusTransactions = SummaryBlockSizeMinusTransactions + len(bz.SCSig)
	// ------------------------------------------------------------
	SummTxsSizeInSummBlock := blockchain.SCSummaryTxMeasurement(SummTxNum)
	blocksize = SummaryBlockSizeMinusTransactions + SummTxsSizeInSummBlock
	// ---
	// for now sync transaction and summary transaction are the same, we should change it when  they differ
	SyncTxSize := SummTxsSizeInSummBlock
	// ---
	newTransactionRow[1] = strconv.Itoa(SyncTxSize)
	newTransactionRow[4] = strconv.Itoa(totalPoR)

	for i, v := range newTransactionRow {
		s[i] = v
	}
	if err = f.InsertRow("FirstQueue", 2); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	} else {
		if err = f.SetSheetRow("FirstQueue", "A2", &s); err != nil {
			log.LLvl1("Panic Raised:\n\n")
			panic(err)
		} else {
			log.Lvl4("Final result MC:\n a Sync tx added to queue in mc round number", bz.MCRoundNumber)
		}
	}

	// -------------------------------------------------------------------------------
	//err = f.SaveAs("/root/remote/mainchainbc.xlsx")
	err = f.SaveAs(bcDirectory)

	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	} else {
		log.Lvl2("mc bc Successfully closed")
		log.Lvl1("Final result MC:", bz.Name(), " finished collecting new sync transactions to queue in mc round number ", bz.MCRoundNumber)
		log.Lvl1("syncMainChainBCTransactionQueueCollect took:", time.Since(takenTime).String(), "for mc round number", bz.MCRoundNumber)

	}

	return blocksize
}
