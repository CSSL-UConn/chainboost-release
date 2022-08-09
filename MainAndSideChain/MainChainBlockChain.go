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
		//todoraha: use unique increasing int values for each server 1<->1 contract instead of random values
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
	log.Lvl1("readBCAndSendtoOthers took:", time.Since(takenTime).String(), "for round number", bz.MCRoundNumber)

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

	// ----------------------
	//tododraha: check if works remove this later
	// ----------------------
	// if rows, err = f.Rows("RoundTable"); err != nil {
	// 	log.LLvl1("Panic Raised:\n\n")
	// 	panic(err)
	// }
	// for rows.Next() {
	// 	rowNumber++
	// 	if row, err = rows.Columns(); err != nil {
	// 		log.LLvl1("Panic Raised:\n\n")
	// 		panic(err)
	// 	}
	// }
	// last row:
	//for i, colCell := range row {
	// if i == 0 {
	// 	if bz.MCRoundNumber, err = strconv.Atoi(colCell); err != nil {
	// 		log.LLvl1("Panic Raised:\n\n")
	// 		panic(err)
	// 	}
	// }
	// 	if i == 1 {
	// 		seed = colCell // last round's seed
	// 	}
	// }
	//-------------------------------------------------------------------------------------------
	// looking for all nodes' power in the last round in the power table sheet in the mainchainbc file
	// rowNumber = 0 //ToDoRaha: later it can go straight to last row based on the round number found in round table
	// rowNum := 0
	// if rows, err = f.Rows("PowerTable"); err != nil {
	// 	log.LLvl1("Panic Raised:\n\n")
	// 	panic(err)
	// }
	// for rows.Next() {
	// 	rowNumber++
	// }
	// last row in power table:
	// if row, err = rows.Columns(); err != nil {
	// 	log.LLvl1("Panic Raised:\n\n")
	// 	panic(err)
	// }
	// ----------------------

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
			log.Lvl4(i+1, "-th miner in roster list:", a.Address.String(), "\ngot power of", powerCell, "\nin row number:", rowNumber, "and round number:", bz.MCRoundNumber)
		}
	}

	// ----------------------
	//tododraha: check if works remove this later
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
	log.Lvl1("readBCPowersAndSeed took:", time.Since(takenTime).String(), "for round number", bz.MCRoundNumber+1)
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
	// ---
	axisBCSize := "C" + currentRow
	err = f.SetCellValue("RoundTable", axisBCSize, bz.MainChainBlockSize)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
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
			row, _ = rowsMarketMatching.Columns()
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

	// ---------------------------------------------------------------------
	// --- Power Table sheet  ----------------------------------------------
	// ---------------------------------------------------------------------
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
		log.Lvl1("updateBCPowerRound took:", time.Since(takenTime).String(), "for round number", bz.MCRoundNumber)

	}
}

/* ----------------------------------------------------------------------
    updateBC: this is a connection between first layer of blockchain - ROOT NODE - on the second layer - xlsx file -
------------------------------------------------------------------------ */
func (bz *ChainBoost) updateMainChainBCTransactionQueueCollect() {
	var err error
	var rows *excelize.Rows
	var row []string

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
	// -------------------------------------------------------------------------------
	var takenTime time.Time
	takenTime = time.Now()
	// -------------------------------------------------------------------------------
	// each round, adding one row in power table based on the information in market matching sheet,
	// assuming that servers are honest and have honestly publish por for their actice (not expired) ServAgrs,
	// for each storage server and each of their active contracst, add the stored file size to their current power
	// -------------------------------------------------------------------------------
	if rows, err = f.Rows("MarketMatching"); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	var ServAgrDuration, ServAgrStartedMCRoundNumber, FileSize, ServAgrPublished, ServAgrID int
	var MinerServer, ServAgrIDString string

	rowNum := 0
	transactionQueue := make(map[string][5]int)
	// first int:  stored file size in this round,
	// second int: corresponding ServAgr id
	// third int:  TxServAgrPropose required
	// fourth int: TxStoragePayment required
	for rows.Next() {
		rowNum++
		if rowNum == 1 { // first row is header
			_, _ = rows.Columns()
		} else {
			row, err = rows.Columns()
			if err != nil {
				log.LLvl1("Panic Raised:\n\n")
				panic(err)
			} else {
				for i, colCell := range row {

					/* --- in MarketMatching:
					i = 0 is Server's Info,
					i = 1 is FileSize,
					i = 2 is ServAgrDuration,
					i = 3 is MCRoundNumber,
					i = 4 is ServAgrID,
					i = 5 is Client's PK,
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
					if i == 4 {
						ServAgrIDString = colCell
						ServAgrID, err = strconv.Atoi(colCell)
						if err != nil {
							log.LLvl1("bad colCell is:", colCell,
								" cell row num is: ", rowNum, " and the rest of row is:", row)
							log.LLvl1("Panic Raised:\n\n")
							panic(err)
						}
					}
					if i == 5 {
						ServAgrPublished, err = strconv.Atoi(colCell)
						if err != nil {
							log.LLvl1("Panic Raised:\n\n")
							panic(err)
						}
					}
				}
			}
			t := [5]int{0, ServAgrID, 0, 0, 0}
			// map transactionQueue:
			// t[0]: stored file size in this round,
			// t[1]: corresponding ServAgr id
			// t[2]: TxServAgrPropose required
			// t[3]: TxStoragePayment required
			// t[4]: TxPor required
			if ServAgrPublished == 0 {
				// Add TxServAgrPropose
				t[2] = 1
				transactionQueue[MinerServer] = t
				/* ------------------------------------------------------------
				// when simStat == 2 it means that side chain is
				running and por tx.s go to side chain queue
				------------------------------------------------------------ */
			} else if bz.MCRoundNumber-ServAgrStartedMCRoundNumber <= ServAgrDuration && bz.SimState == 1 { // ServAgr is not expired
				t[0] = FileSize //if each server one ServAgr
				// Add TxPor
				t[4] = 1
				transactionQueue[MinerServer] = t
			} else if bz.MCRoundNumber-ServAgrStartedMCRoundNumber > ServAgrDuration {
				// Set ServAgrPublished to false
				// ----
				// todoraha: check if works file, remove it later
				// ----
				// if ServAgrIdCellMarketMatching, err := f.SearchSheet("MarketMatching", ServAgrIDString); err != nil {
				// 	log.LLvl1("Panic Raised:\n\n")
				// 	panic(err)
				// } else {
				//publishedCellMarketMatching := "F" + ServAgrIdCellMarketMatching[0][1:]
				// ----
				publishedCellMarketMatching := "F" + ServAgrIDString
				err = f.SetCellValue("MarketMatching", publishedCellMarketMatching, 0)
				if err != nil {
					log.LLvl1("Panic Raised:\n\n")
					panic(err)
				}
				//}
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
	4) ServAgrId */
	var newTransactionRow [5]string
	s := make([]interface{}, len(newTransactionRow)) //ToDoRaha:  check this out later: https://stackoverflow.com/questions/23148812/whats-the-meaning-of-interface/23148998#23148998

	// this part can be moved to protocol initialization
	var PorTxSize, ServAgrProposeTxSize, PayTxSize, StoragePayTxSize, ServAgrCommitTxSize uint32
	PorTxSize, ServAgrProposeTxSize, PayTxSize, StoragePayTxSize, ServAgrCommitTxSize = blockchain.TransactionMeasurement(bz.SectorNumber, bz.SimulationSeed)
	// ---
	addCommitTx := false
	// map transactionQueue:
	// [0]: stored file size in this round,
	// [1]: corresponding ServAgr id
	// [2]: TxServAgrPropose required
	// [3]: TxStoragePayment required
	// [4]: TxPor required
	for _, a := range bz.Roster().List {
		if transactionQueue[a.Address.String()][2] == 1 { //TxServAgrPropose required
			newTransactionRow[2] = time.Now().Format(time.RFC3339)
			newTransactionRow[3] = strconv.Itoa(bz.MCRoundNumber)
			newTransactionRow[0] = "TxServAgrPropose"
			newTransactionRow[1] = strconv.Itoa(int(ServAgrProposeTxSize))
			newTransactionRow[4] = strconv.Itoa(transactionQueue[a.Address.String()][1]) // corresponding ServAgr id
			addCommitTx = true                                                           // another row will be added containing "TxServAgrCommit"
		} else if transactionQueue[a.Address.String()][3] == 1 { // TxStoragePayment required
			newTransactionRow[2] = time.Now().Format(time.RFC3339)
			newTransactionRow[3] = strconv.Itoa(bz.MCRoundNumber)
			newTransactionRow[0] = "TxStoragePayment"
			newTransactionRow[1] = strconv.Itoa(int(StoragePayTxSize))
			newTransactionRow[4] = strconv.Itoa(transactionQueue[a.Address.String()][1])
			// && bz.SimState == 1 is  for backup check-  the first condition should never be true when the second condition isn't!
		} else if transactionQueue[a.Address.String()][4] == 1 && bz.SimState == 1 { // TxPor required
			newTransactionRow[2] = time.Now().Format(time.RFC3339)
			newTransactionRow[3] = strconv.Itoa(bz.MCRoundNumber)
			newTransactionRow[0] = "TxPor"
			newTransactionRow[1] = strconv.Itoa(int(PorTxSize))
			newTransactionRow[4] = strconv.Itoa(transactionQueue[a.Address.String()][1])
		} else {
			continue
		}

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
				if newTransactionRow[0] == "TxPor" {
					log.Lvl4("a TxPor added to queue in round number", bz.MCRoundNumber)
				} else if newTransactionRow[0] == "TxStoragePayment" {
					log.Lvl4("a TxStoragePayment added to queue in round number", bz.MCRoundNumber)
				} else if newTransactionRow[0] == "TxServAgrPropose" {
					log.Lvl4("a TxServAgrPropose added to queue in round number", bz.MCRoundNumber)
				}
			}
		}
		/* second row added in case of having the first row to be ServAgr propose tx which then we will add
		ServAgr commit tx right away.
		warning: Just in one case it may cause irrational statistics which doesn’t worth taking care of!
		when a propose ServAgr tx is added to a block which causes the ServAgr to become active but
		the commit ServAgr transaction is not yet! */
		if addCommitTx {
			newTransactionRow[0] = "TxServAgrCommit"
			newTransactionRow[1] = strconv.Itoa(int(ServAgrCommitTxSize))
			newTransactionRow[4] = strconv.Itoa(transactionQueue[a.Address.String()][1]) // corresponding ServAgr id
			//--
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
					addCommitTx = false
					log.Lvl4("a TxServAgrCommit added to queue in round number", bz.MCRoundNumber)
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

	newTransactionRow[0] = strconv.Itoa(int(PayTxSize))
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
	log.LLvl1("Number of regular payment transactions in round number", bz.MCRoundNumber, "is", numberOfRegPay)
	for i := 1; i <= numberOfRegPay; i++ {
		if err = f.InsertRow("SecondQueue", 2); err != nil {
			log.LLvl1("Panic Raised:\n\n")
			panic(err)
		} else {
			if err = f.SetSheetRow("SecondQueue", "A2", &s); err != nil {
				log.LLvl1("Panic Raised:\n\n")
				panic(err)
			} else {
				log.Lvl4("a regular payment transaction added to queue in round number", bz.MCRoundNumber)
			}
		}
	}
	// -------------------------------------------------------------------------------

	// ---
	//err = f.SaveAs("/root/remote/mainchainbc.xlsx")
	err = f.SaveAs(bcDirectory)

	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	} else {
		log.Lvl2("bc Successfully closed")
		log.Lvl1(bz.Name(), " finished collecting new transactions to queue in round number ", bz.MCRoundNumber)
		log.Lvl1("Collecting tx.s took:", time.Since(takenTime).String())
	}
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
		//todoraha: check if works fine, remove this part later
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

		// ----------------
		//todoraha: check if works fine, remove this part later
		// ----------------
		//for j, colCell := range row {
		//if j == 1 {
		// ----------------

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
				// todoraha: check and remove it later if works fine
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
				numberOfPoRTx, _ = strconv.Atoi(row[4]) //In case that the transaction type is a Sync transaction,
				// the 4th column will be the number of por transactions summerized in
				// the sync tx
			default:
				log.Lvl4("Panic Raised:\n\n")
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
		//}
		//}
	}
	log.Lvl1("other tx.s taking took:", time.Since(takenTime).String())

	f.SetCellValue("RoundTable", axisNumPoRTx, numberOfPoRTx)
	f.SetCellValue("RoundTable", axisNumStoragePayTx, numberOfStoragePayTx)
	f.SetCellValue("RoundTable", axisNumServAgrProposeTx, numberOfServAgrProposeTx)
	f.SetCellValue("RoundTable", axisNumServAgrCommitTx, numberOfServAgrCommitTx)
	f.SetCellValue("RoundTable", axisNumSyncTx, numberOfSyncTx)

	log.Lvl2("In total in round number ", bz.MCRoundNumber,
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

	log.Lvl1("In total in round number ", bz.MCRoundNumber,
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
		log.Lvl1(bz.Name(), " Finished taking transactions from queue (FIFO) into new block in round number ", bz.MCRoundNumber)
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
	s := make([]interface{}, len(newTransactionRow)) //ToDoRaha: raha:  check this out later: https://stackoverflow.com/questions/23148812/whats-the-meaning-of-interface/23148998#23148998

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
			SummTxNum = SummTxNum + 1 // counting the number of active contracts to measure the length of summery tx
		}
		bz.SummPoRTxs[i] = 0
	}
	//measuring summery block size
	SummeryBlockSizeMinusTransactions, _ := blockchain.SCBlockMeasurement()
	// --------------- adding bls signature size  -----------------
	log.Lvl4("Size of bls signature:", len(bz.SCSig))
	SummeryBlockSizeMinusTransactions = SummeryBlockSizeMinusTransactions + len(bz.SCSig)
	// ------------------------------------------------------------
	SummTxsSizeInSummBlock := blockchain.SCSummeryTxMeasurement(SummTxNum)
	blocksize = SummeryBlockSizeMinusTransactions + SummTxsSizeInSummBlock
	// ---
	// for now sync transaction and summery transaction are the same, we should change it when  they differ
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
			log.Lvl1("Final result MC:\n a Sync tx added to queue in round number", bz.MCRoundNumber)
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
		log.Lvl1("Final result MC:", bz.Name(), " finished collecting new sync transactions to queue in round number ", bz.MCRoundNumber)
		log.Lvl1("syncMainChainBCTransactionQueueCollect took:", time.Since(takenTime).String(), "for round number", bz.MCRoundNumber)

	}

	return blocksize
}
