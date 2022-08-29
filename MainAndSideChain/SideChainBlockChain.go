package MainAndSideChain

import (
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chainBoostScale/ChainBoost/MainAndSideChain/blockchain"
	"github.com/chainBoostScale/ChainBoost/onet/log"
	"github.com/xuri/excelize/v2"
)

/* ----------------------------------------------------------------------
	updateSideChainBC:  when a side chain leader submit a meta block, the side chain blockchain is
	updated by the root node to reflelct an added meta-block
 ----------------------------------------------------------------------*/
func (bz *ChainBoost) updateSideChainBCRound(LeaderName string, blocksize int) {
	//var epochNumber = int(math.Floor(float64(bz.MCRoundNumber) / float64(bz.MCRoundPerEpoch)))
	var err error
	// var rows *excelize.Rows
	// var row []string
	takenTime := time.Now()
	pwd, _ := os.Getwd()
	log.Lvl4("opening sidechainbc in:", pwd)
	//bcDirectory := strings.Split(pwd, "/build")
	bcDirectory := strings.Split(pwd, "/build")[0] + "/sidechainbc.xlsx"
	log.Lvl4("opening sidechainbc in:", bcDirectory)
	//f, err := excelize.OpenFile("/root/remote/sidechainbc.xlsx")
	f, err := excelize.OpenFile(bcDirectory)
	if err != nil {
		log.Fatal("problem while opening sidechainbc: " + err.Error())
	} else {
		log.Lvl2("sidechainbc Successfully opened")
	}

	// --------------------------------------------------------------------
	// looking for last round's number in the round table sheet in the sidechainbc file
	// --------------------------------------------------------------------
	// finding the last row in side chain bc file, in round table sheet
	var rows1 *excelize.Rows
	var row1 []string
	rowNumber := 1
	var RoundIntervalSec int
	var roundNumber int
	if rows1, err = f.Rows("RoundTable"); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	for rows1.Next() {
		rowNumber++
		if row1, err = rows1.Columns(); err != nil {
			log.LLvl1("Panic Raised:\n\n")
			panic(err)
		}
	}
	for i, colCell := range row1 {
		// --- in RoundTable: i = 0 is (next) round number, for i>=2 , the cells are empty now,
		//will be updated by the root node in the rest of this (Take) function
		if i == 0 {
			roundNumber, _ = strconv.Atoi(colCell)
		} else if i == 4 {
			TakeTime, _ := time.Parse(time.RFC3339, colCell)
			RoundIntervalSec = int(time.Now().Sub(TakeTime).Seconds())
		}
	}
	log.Lvl3("Final result SC: round number: ", roundNumber, "took ", RoundIntervalSec, " seconds in total")
	currentRow := strconv.Itoa(rowNumber)
	//nextRow := strconv.Itoa(rowNumber + 1)
	// ---
	// this approach of calculating the current row / side chain round  number should rationally work but it didn't, maybe check later to find a solution
	//CurrentRow := strconv.Itoa(epochNumber*bz.MCRoundDuration*bz.MCRoundPerEpoch/bz.SCRoundDuration + bz.SCRoundNumber + 1)
	//nextRow := strconv.Itoa(epochNumber*bz.MCRoundDuration*bz.MCRoundPerEpoch/bz.SCRoundDuration + bz.SCRoundNumber + 2)
	// --------------------------------------------------------------------
	// --------------------------------------------------------------------

	// --------------------------------------------------------------------
	// updating the current last row in the "BCsize" column
	// axisBCSize := "B" + currentRow
	// err = f.SetCellValue("RoundTable", axisBCSize, bz.SideChainBlockSize)
	// // this value of block size is not correct if the queue gets not full sometimes, but in the evaluation sheet the value is correct
	// if err != nil {
	// 	log.LLvl1("Panic Raised:\n\n")
	// 	panic(err)
	// }
	// --------------------------------------------------------------------
	// --- set starting round time
	// --------------------------------------------------------------------
	cellStartDate := "E" + currentRow
	cellStartInterval := "I" + currentRow
	cellStartMCRN := "J" + currentRow

	err = f.SetCellValue("RoundTable", cellStartDate, time.Now().Format(time.RFC3339))
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	err = f.SetCellValue("RoundTable", cellStartInterval, RoundIntervalSec)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	err = f.SetCellValue("RoundTable", cellStartMCRN, bz.MCRoundNumber)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	// --------------------------------------------------------------------
	// updating the current last row in the "miner" column
	axisMiner := "C" + currentRow
	err = f.SetCellValue("RoundTable", axisMiner, LeaderName)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	// --------------------------------------------------------------------
	axisSCRoundNumber := "A" + currentRow
	err = f.SetCellValue("RoundTable", axisSCRoundNumber, bz.SCRoundNumber)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	// --------------------------------------------------------------------
	if blocksize != 0 { // blocksize is measured before, while issuing the sync transaction and it is passed from func: syncMainChainBCTransactionQueueCollect
		axisBlockSize := "B" + currentRow
		err = f.SetCellValue("RoundTable", axisBlockSize, blocksize)
		if err != nil {
			log.LLvl1("Panic Raised:\n\n")
			panic(err)
		}
	}
	// --------------------------------------------------------------------
	//err = f.SaveAs("/root/remote/sidechainbc.xlsx")
	err = f.SaveAs(bcDirectory)

	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	} else {
		log.Lvl2("sc bc successfully closed")
	}
	updateSideChainBCOverallEvaluation(currentRow, bz.SCRoundNumber)
	log.Lvl1("updateSideChainBCRound took:", time.Since(takenTime).String(), "for sc round number", bz.SCRoundNumber)
}

/* ----------------------------------------------------------------------
    each side chain's round, the side chain blockchain is
	updated by the root node to add new proposed PoR tx.s in the queue
	the por tx.s are collected based on the service agreement status read from main chain blockchain
------------------------------------------------------------------------ */
func (bz *ChainBoost) updateSideChainBCTransactionQueueCollect() {

	var err error
	// var rows *excelize.Rows
	// var row []string
	takenTime := time.Now()

	pwd, _ := os.Getwd()
	log.Lvl4("opening mainchainbc in:", pwd)
	//bcDirectory := strings.Split(pwd, "/build")
	bcDirectory := strings.Split(pwd, "/build")[0] + "/mainchainbc.xlsx"
	log.Lvl4("opening mainchainbc in:", bcDirectory)
	//f, err := excelize.OpenFile("/root/remote/mainchainbc.xlsx")
	f, err := excelize.OpenFile(bcDirectory)
	if err != nil {
		log.Fatal("problem while opening mainchainbc: " + err.Error())
	} else {
		log.Lvl2("mainchainbc Successfully opened")
	}

	// -------------------------------------------------------------------------------
	// each round, .... based on the information in market matching sheet,
	// -------------------------------------------------------------------------------
	cols, err := f.GetCols("MarketMatching")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	/* --- in MarketMatching:
	cols[0] is Server's Info,
	cols[1] is FileSize,
	cols[2] is ServAgrDuration,
	cols[3] is ServAgrStartedMCRoundNumber,
	cols[4] is ServAgrID,
	cols[5] is ServAgrPublished */
	ServAgrDurationCol := cols[2]
	ServAgrStartedMCRoundNumberCol := cols[3]
	ServAgrPublishedCol := cols[5]
	f.Close() //todo: does it help?!

	// -------------------------------------------------------------------------------
	pwd, _ = os.Getwd()
	log.Lvl4("opening sidechainbc in:", pwd)
	//bcDirectory := strings.Split(pwd, "/build")
	bcDirectory = strings.Split(pwd, "/build")[0] + "/sidechainbc.xlsx"
	log.Lvl4("opening sidechainbc in:", bcDirectory)
	//f, err := excelize.OpenFile("/root/remote/sidechainbc.xlsx")
	f1, err := excelize.OpenFile(bcDirectory)
	if err != nil {
		log.Fatal("problem while opening sidechainbc: " + err.Error())
	} else {
		log.Lvl2("sidechainbc Successfully opened")
	}

	// ----------------------------------------------------------------------
	// ------ add 5(now it is just 1:por!) types of transactions into transaction queue sheet -----
	// ----------------------------------------------------------------------
	/* each transaction has the following column stored on the transaction queue sheet:
	0) name
	1) size
	2) time
	3) issuedMCRoundNumber
	4) ServAgrId */

	var newTransactionRow [6]string
	s := make([]interface{}, len(newTransactionRow)) //ToDoRaha:  check this out later: https://stackoverflow.com/questions/23148812/whats-the-meaning-of-interface/23148998#23148998

	// this part can be moved to protocol initialization
	var PorTxSize uint32
	PorTxSize, _, _, _, _ = blockchain.TransactionMeasurement(bz.SectorNumber, bz.SimulationSeed)
	// ---
	var ServAgrDuration, ServAgrStartedMCRoundNumber, ServAgrPublished int
	numOfPoRTxs := 1
	colNum := 1
	rowNum := 2
	// ---
	streamWriter, err := f1.NewStreamWriter("FirstQueue")
	if err != nil {
		log.Fatal("problem while opening streamWriter: " + err.Error())
	} else {
		log.Lvl2("sidechainbc streamWriter opened")
	}
	// ------ add Header Row on top of the stream (first queue in side chain bc) ----------
	rows, err := f1.GetRows("FirstQueue")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	} else {
		row := rows[0][:]
		for i, v := range row {
			s[i] = v
		}
		cell, _ := excelize.CoordinatesToCellName(colNum, colNum)
		if err := streamWriter.SetRow(cell, s); err != nil {
			log.LLvl1("Panic Raised:\n\n")
			panic(err)
		}
	}
	// --- check for eligible contracts and add a por tx row on top of the stream ----
	for i := range bz.Roster().List {
		ServAgrDuration, err = strconv.Atoi(ServAgrDurationCol[i+1]) //ServAgrDurationCol is starting from index 0
		if err != nil {
			log.LLvl1("Panic Raised:\n\n", err)
			panic(err)
		}
		ServAgrStartedMCRoundNumber, err = strconv.Atoi(ServAgrStartedMCRoundNumberCol[i+1]) //ServAgrStartedMCRoundNumberCol is starting from index 1
		if err != nil {
			log.LLvl1("Panic Raised:\n\n", err)
			panic(err)
		}
		ServAgrPublished, err = strconv.Atoi(ServAgrPublishedCol[i+1]) //ServAgrPublishedCol is starting from index 1
		if err != nil {
			log.LLvl1("Panic Raised:\n\n", err)
			panic(err)
		}
		// --------------------------------------
		if ServAgrPublished == 1 && bz.MCRoundNumber-ServAgrStartedMCRoundNumber <= ServAgrDuration {
			// ServAgr is not expired => Add TxPor
			newTransactionRow[2] = time.Now().Format(time.RFC3339)
			// 50008 means 50000 + 8 which means epoch number 5 scround number 8 //todo: is it still on?
			newTransactionRow[3] = strconv.Itoa(bz.SCRoundNumber)
			newTransactionRow[0] = "TxPor"
			newTransactionRow[1] = strconv.Itoa(int(PorTxSize))
			newTransactionRow[4] = strconv.Itoa(i)
			newTransactionRow[5] = strconv.Itoa(bz.MCRoundNumber)
			for i, v := range newTransactionRow {
				s[i] = v
			}
			// ------ add a por tx row on top of the stream ----------
			cell, _ := excelize.CoordinatesToCellName(colNum, rowNum)
			if err := streamWriter.SetRow(cell, s); err != nil {
				log.LLvl1("Panic Raised:\n\n")
				panic(err)
			} else {
				numOfPoRTxs++
				rowNum++
			}
		}
	}
	rowNum--
	numOfPoRTxs--
	// ------ add previous por tx row in the first queue to the stream ----------
	for i, row := range rows { //i starts from index 0
		for t, v := range row {
			s[t] = v
		}
		if i != 0 { // index 0 is the header row
			cell, _ := excelize.CoordinatesToCellName(colNum, i+rowNum)
			if err := streamWriter.SetRow(cell, s); err != nil {
				log.LLvl1("Panic Raised:\n\n")
				panic(err)
			}
		}
	}
	/* from the documentation: "after set rows, you must call the Flush method to
	end the streaming writing process and ensure that the order of line numbers is ascending,
	the common API and stream API can't be work mixed to writing data on the worksheets." */
	if err := streamWriter.Flush(); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	if err = f1.SaveAs(bcDirectory); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	} else {
		log.Lvl2("sc bc successfully closed")
		log.Lvl4(bz.Name(), "Final result SC: finished collecting new transactions to side chain queue in sc round number ", bz.SCRoundNumber)
		log.Lvl1("updateSideChainBCTransactionQueueCollect took:", time.Since(takenTime).String())
		log.Lvl1(numOfPoRTxs, "TxPor added to queue in sc round number: ", bz.SCRoundNumber)
	}
}

/* ----------------------------------------------------------------------
    updateBC: this is a connection between first layer of blockchain - ROOT NODE - on the second layer - xlsx file -
------------------------------------------------------------------------ */
func (bz *ChainBoost) updateSideChainBCTransactionQueueTake() int {
	var err error
	var rows [][]string
	takenTime := time.Now()
	//var epochNumber = int(math.Floor(float64(bz.MCRoundNumber) / float64(bz.MCRoundPerEpoch)))
	// --- reset
	bz.SideChainQueueWait = 0

	pwd, _ := os.Getwd()
	log.Lvl4("opening sidechainbc in:", pwd)
	//bcDirectory := strings.Split(pwd, "/build")
	bcDirectory := strings.Split(pwd, "/build")[0] + "/sidechainbc.xlsx"
	log.Lvl4("opening sidechainbc in:", bcDirectory)
	//f, err := excelize.OpenFile("/root/remote/sidechainbc.xlsx")
	f, err := excelize.OpenFile(bcDirectory)
	if err != nil {
		log.Fatal("problem while opening sidechainbc: " + err.Error())
	} else {
		log.Lvl2("sidechainbc Successfully opened")
	}

	var accumulatedTxSize, txsize int
	blockIsFull := false
	_, MetaBlockSizeMinusTransactions := blockchain.SCBlockMeasurement()
	// --------------- adding bls signature size  -----------------
	log.Lvl4("Size of bls signature:", len(bz.SCSig))
	MetaBlockSizeMinusTransactions = MetaBlockSizeMinusTransactions + len(bz.SCSig)
	// ------------------------------------------------------------
	//var TakeTime time.Time

	/* -----------------------------------------------------------------------------
		 -- take por transactions from sheet: FirstQueue
	----------------------------------------------------------------------------- */
	// --------------------------------------------------------------------
	// looking for last round's number in the round table sheet in the sidechainbc file
	// --------------------------------------------------------------------
	// finding the last row in side chain bc file, in round table sheet
	var rows1 *excelize.Rows
	var row1 []string
	rowNumber := 1
	var roundNumber int
	if rows1, err = f.Rows("RoundTable"); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	for rows1.Next() {
		rowNumber++
		if row1, err = rows1.Columns(); err != nil {
			log.LLvl1("Panic Raised:\n\n")
			panic(err)
		}
	}
	for i, colCell := range row1 {
		// --- in RoundTable: i = 0 is (next) round number, for i>=2 , the cells are empty now,
		//will be updated by the root node in the rest of this (Take) function
		if i == 0 {
			roundNumber, _ = strconv.Atoi(colCell)
		}
	}
	log.Lvlf5("side chain's current round number:", roundNumber)
	CurrentRow := strconv.Itoa(rowNumber - 1) // last row that has some columns filled
	//NextRow := strconv.Itoa(rowNumber + 1)
	// ---
	// this approach of calculating the current row / side chain round  number should rationally work but it didn't, maybe check later to find a solution
	//CurrentRow := strconv.Itoa(epochNumber*bz.MCRoundDuration*bz.MCRoundPerEpoch/bz.SCRoundDuration + bz.SCRoundNumber + 1)
	// --------------------------------------------------------------------
	// --------------------------------------------------------------------

	axisQueue1IsFull := "H" + CurrentRow

	if rows, err = f.GetRows("FirstQueue"); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	// reset variables
	txsize = 0
	blockIsFull = false
	accumulatedTxSize = 0

	numberOfPoRTx := 0
	axisBlockSize := "B" + CurrentRow
	axisNumPoRTx := "D" + CurrentRow
	axisAveFirstQueueWait := "F" + CurrentRow

	// len(rows) gives number of rows - the length of the "external" array
	for i := len(rows); i > 1 && !blockIsFull; i-- {
		row := rows[i-1][:]
		/* each transaction has the following column stored on the transaction queue sheet:
		0) name
		1) size
		2) time
		3) issuedMCRoundNumber
		4) ServAgrId */
		if txsize, err = strconv.Atoi(row[1]); err != nil {
			log.LLvl1("Panic Raised:\n\n")
			panic(err)
		} else if accumulatedTxSize+txsize <= bz.SideChainBlockSize-MetaBlockSizeMinusTransactions {
			accumulatedTxSize = accumulatedTxSize + txsize
			if row[0] == "TxPor" {
				log.Lvl4("a por tx added to block number", bz.MCRoundNumber, " from the queue")
				numberOfPoRTx++
			} else {
				log.LLvl1("Panic Raised:\n\n")
				panic("the type of transaction in the queue is un-defined")
			}
			// row[2] is transaction's collected time
			// if TakeTime, err = time.Parse(time.RFC3339, row[2]); err != nil {
			// 	log.LLvl1("Panic Raised:\n\n")
			// 	panic(err)
			// }
			// bz.SideChainQueueWait = bz.SideChainQueueWait + int(time.Now().Sub(TakeTime).Seconds())

			// row[3] is the SCRound Number that the transaction has been issued
			if x1, err := strconv.Atoi(row[3]); err != nil {
				log.LLvl1("Panic Raised:\n\n")
				panic(err)
			} else {
				if x2, err := strconv.Atoi(row[5]); err != nil {
					log.LLvl1("Panic Raised:\n\n")
					panic(err)
				} else {
					bz.SideChainQueueWait = bz.SideChainQueueWait + int(math.Abs(float64(bz.SCRoundNumber-x1))) + bz.MCRoundDuration/bz.SCRoundDuration*(bz.MCRoundNumber-x2)
				}
			}

			// ---------- keep taken transaction's info summery --------------------
			if serverAgrId, err := strconv.Atoi(row[4]); err != nil {
				log.LLvl1("Panic Raised:\n\n")
				panic(err)
			} else {
				bz.SummPoRTxs[serverAgrId] = bz.SummPoRTxs[serverAgrId] + 1
			}
			// remove the transaction row from the queue
			f.RemoveRow("FirstQueue", i)
		} else {
			blockIsFull = true
			log.Lvl1("final result SC:\n side chain block is full! ")
			f.SetCellValue("RoundTable", axisQueue1IsFull, 1)
			break
		}
	}

	f.SetCellValue("RoundTable", axisNumPoRTx, numberOfPoRTx)

	log.LLvl1("final result SC:\n In total in sc round number ", bz.SCRoundNumber,
		"\n number of published PoR transactions is", numberOfPoRTx)
	TotalNumTxsInFirstQueue := numberOfPoRTx

	//-- accumulated block size
	// --- total throughput
	f.SetCellValue("RoundTable", axisBlockSize, accumulatedTxSize+MetaBlockSizeMinusTransactions)
	if TotalNumTxsInFirstQueue != 0 {
		f.SetCellValue("RoundTable", axisAveFirstQueueWait, float64(bz.SideChainQueueWait)/float64(TotalNumTxsInFirstQueue))
	} else {
		f.SetCellValue("RoundTable", axisAveFirstQueueWait, 0)
	}

	log.Lvl1("final result SC:\n", " this round's block size: ", accumulatedTxSize+MetaBlockSizeMinusTransactions)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	//log.LLvl1("In total in round number ", bz.SCRoundNumber+10000 * epochNumber,
	//	"\n number of all types of submitted txs is: ", TotalNumTxsInFirstQueue)
	// ----
	//err = f.SaveAs("/root/remote/sidechainbc.xlsx")
	err = f.SaveAs(bcDirectory)

	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	} else {
		log.Lvl2("sc bc successfully closed")
	}
	// fill OverallEvaluation Sheet
	updateSideChainBCOverallEvaluation(CurrentRow, bz.SCRoundNumber)
	blocksize := accumulatedTxSize + MetaBlockSizeMinusTransactions
	log.Lvl1("updateSideChainBCTransactionQueueTake took:", time.Since(takenTime).String())
	return blocksize
}

// --------------------------------------------------------------------------------
// ----------------------- OverallEvaluation Sheet --------------------
// --------------------------------------------------------------------------------
func updateSideChainBCOverallEvaluation(CurrentRow string, SCRoundNumber int) {
	var err error
	takenTime := time.Now()
	pwd, _ := os.Getwd()
	log.Lvl4("opening sidechainbc in:", pwd)
	//bcDirectory := strings.Split(pwd, "/build")
	bcDirectory := strings.Split(pwd, "/build")[0] + "/sidechainbc.xlsx"
	log.Lvl4("opening sidechainbc in:", bcDirectory)
	//f, err := excelize.OpenFile("/root/remote/sidechainbc.xlsx")
	f, err := excelize.OpenFile(bcDirectory)
	if err != nil {
		log.Fatal("problem while opening sidechainbc: " + err.Error())
	} else {
		log.Lvl2("sidechainbc Successfully opened")
	}

	// ---- overall results
	axisRound := "A" + CurrentRow
	axisBCSize := "B" + CurrentRow
	axisOverallPoRTX := "C" + CurrentRow
	axisAveWaitOtherTx := "D" + CurrentRow
	axisOverallBlockSpaceFull := "E" + CurrentRow
	var FormulaString string

	err = f.SetCellValue("OverallEvaluation", axisRound, SCRoundNumber)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	FormulaString = "=SUM(RoundTable!B2:B" + CurrentRow + ")"
	err = f.SetCellFormula("OverallEvaluation", axisBCSize, FormulaString)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	FormulaString = "=SUM(RoundTable!D2:D" + CurrentRow + ")"
	err = f.SetCellFormula("OverallEvaluation", axisOverallPoRTX, FormulaString)
	if err != nil {
		log.LLvl1(err)
	}

	FormulaString = "=AVERAGE(RoundTable!F2:F" + CurrentRow + ")"
	err = f.SetCellFormula("OverallEvaluation", axisAveWaitOtherTx, FormulaString)
	if err != nil {
		log.LLvl1(err)
	}
	FormulaString = "=SUM(RoundTable!H2:H" + CurrentRow + ")"
	err = f.SetCellFormula("OverallEvaluation", axisOverallBlockSpaceFull, FormulaString)
	if err != nil {
		log.LLvl1(err)
	}

	// ----
	//err = f.SaveAs("/root/remote/sidechainbc.xlsx")
	err = f.SaveAs(bcDirectory)

	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	} else {
		log.Lvl2("sc bc successfully closed")
		log.Lvl1("updateSideChainBCOverallEvaluation took:", time.Since(takenTime).String())
	}
}
