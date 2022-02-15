package BaseDFSProtocol

import (
	"strconv"
	"time"

	"github.com/basedfs/blockchain"
	"github.com/basedfs/log"
	"github.com/xuri/excelize/v2"
)

/* ----------------------------------------------------------------------
	updateSideChainBC:  when a side chain leader submit a meta block, the side chain blockchain is
	updated by the root node to reflelct an added meta-block
 ----------------------------------------------------------------------*/
func (bz *BaseDFS) updateSideChainBCRound(LeaderName string) {
	var err error
	// var rows *excelize.Rows
	// var row []string

	f, err := excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/sidechainbc.xlsx")
	if err != nil {
		log.Lvl2("Raha: ", err)
		panic(err)
	} else {
		log.Lvl3(bz.Name(), "opening side chain bc")
	}
	currentRow := strconv.Itoa(bz.SCRoundNumber + 1)
	nextRow := strconv.Itoa(bz.SCRoundNumber + 2)
	// --------------------------------------------------------------------
	// updating the current last row in the "BCsize" column
	axisBCSize := "C" + currentRow
	err = f.SetCellValue("RoundTable", axisBCSize, bz.BlockSize)
	// ToDo: this value of block size is not correct if the queue gets not full sometimes, but in the evaluation sheet the value is correct
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}
	// --------------------------------------------------------------------
	// --- set starting round time
	// --------------------------------------------------------------------
	cellStartTime := "F" + currentRow
	err = f.SetCellValue("RoundTable", cellStartTime, time.Now().Format(time.RFC3339))

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
	axisSCRoundNumber := "A" + nextRow
	err = f.SetCellValue("RoundTable", axisSCRoundNumber, bz.SCRoundNumber+1+10000*(bz.MCRoundNumber/bz.EpochCount))
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}

	// ----
	err = f.SaveAs("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/sidechainbc.xlsx")
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	} else {
		log.Lvl3("closing side chain  bc")
	}
}

/* ----------------------------------------------------------------------
    each side chain's round, the side chain blockchain is
	updated by the root node to add new proposed PoR tx.s in the queue
	the por tx.s are collected based on the service agreement status read from main chain blockchain
------------------------------------------------------------------------ */
func (bz *BaseDFS) updateSideChainBCTransactionQueueCollect() {
	var err error
	var rows *excelize.Rows
	var row []string

	f, err := excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/mainchainbc.xlsx")
	if err != nil {
		log.Lvl2("Raha: ", err)
		panic(err)
	} else {
		log.Lvl3("opening main chain bc")
	}
	// -------------------------------------------------------------------------------
	// each round, adding one row in power table based on the information in market matching sheet,
	// assuming that servers are honest and have honestly publish por for their actice (not expired) ServAgrs,
	// for each storage server and each of their active contracst, add the stored file size to their current power
	// -------------------------------------------------------------------------------
	if rows, err = f.Rows("MarketMatching"); err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}
	var ServAgrDuration, ServAgrStartedMCRoundNumber, FileSize, ServAgrPublished, ServAgrID int
	var MinerServer string

	rowNum := 0
	transactionQueue := make(map[string][5]int)
	// first int: stored file size in this round,
	// second int: corresponding ServAgr id
	// third int: TxServAgrPropose required
	// fourth int: TxStoragePayment required
	// fifth int: TxPor required
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
					i=2 is ServAgrDuration,
					i=3 is MCRoundNumber,
					i=4 is ServAgrID,
					i=5 is Client's PK,
					i = 6 is ServAgrPublished */
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
						ServAgrDuration, err = strconv.Atoi(colCell)
						if err != nil {
							log.Lvl2("Panic Raised:\n\n")
							panic(err)
						}
					}
					if i == 3 {
						ServAgrStartedMCRoundNumber, err = strconv.Atoi(colCell)
						if err != nil {
							log.Lvl2("Panic Raised:\n\n")
							panic(err)
						}
					}
					if i == 4 {
						ServAgrID, err = strconv.Atoi(colCell)
						if err != nil {
							log.Lvl2("Panic Raised:\n\n")
							panic(err)
						}
					}
					if i == 5 {
						ServAgrPublished, err = strconv.Atoi(colCell)
						if err != nil {
							log.Lvl2("Panic Raised:\n\n")
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
			if ServAgrPublished == 0 && bz.MCRoundNumber-ServAgrStartedMCRoundNumber <= ServAgrDuration {
				// ServAgr is not expired
				t[0] = FileSize //if each server one ServAgr
				// Add TxPor
				t[4] = 1
				transactionQueue[MinerServer] = t
			}
		}
	}
	// -------------------------------------------------------------------------------
	f1, err := excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/sidechainbc.xlsx")
	if err != nil {
		log.Lvl2("Raha: ", err)
		panic(err)
	} else {
		log.Lvl3("opening main chain bc")
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
	s := make([]interface{}, len(newTransactionRow)) //todo: raha:  check this out later: https://stackoverflow.com/questions/23148812/whats-the-meaning-of-interface/23148998#23148998

	newTransactionRow[2] = time.Now().Format(time.RFC3339)
	// 50008 means 50000 + 8 which means epoch number 5 scround number 8
	newTransactionRow[3] = strconv.Itoa(bz.SCRoundNumber + 10000*(bz.MCRoundNumber/bz.EpochCount))
	// this part can be moved to protocol initialization
	var PorTxSize int
	PorTxSize, _, _, _, _ = blockchain.TransactionMeasurement(bz.SectorNumber, bz.SimulationSeed)
	// ---
	// map transactionQueue:
	// [0]: stored file size in this round,
	// [1]: corresponding ServAgr id
	// [2]: TxServAgrPropose required
	// [3]: TxStoragePayment required
	// [4]: TxPor required
	for _, a := range bz.Roster().List {
		if transactionQueue[a.Address.String()][4] == 1 { // TxPor required
			newTransactionRow[0] = "TxPor"
			newTransactionRow[1] = strconv.Itoa(PorTxSize)
			newTransactionRow[4] = strconv.Itoa(transactionQueue[a.Address.String()][1])
		}

		for i, v := range newTransactionRow {
			s[i] = v
		}
		// ------ add a por tx row on top of the queue ------
		if err = f1.InsertRow("FirstQueue", 2); err != nil {
			log.Lvl2("Panic Raised:\n\n")
			panic(err)
		} else {
			if err = f1.SetSheetRow("FirstQueue", "A2", &s); err != nil {
				log.Lvl2("Panic Raised:\n\n")
				panic(err)
			} else {
				if newTransactionRow[0] == "TxPor" {
					log.Lvl3("a TxPor added to queue in sc round number: ", bz.SCRoundNumber, "in epoch number: ", bz.MCRoundNumber/bz.EpochCount)
				}
			}
		}
	}
	// -------------------------------------------------------------------------------
	err = f1.SaveAs("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/sidechainbc.xlsx")
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	} else {
		log.Lvl3("closing side bc")
		log.Lvl2(bz.Name(), "Final result SC: finished collecting new transactions to side chain queue in round number ", bz.SCRoundNumber+10000*(bz.MCRoundNumber/bz.EpochCount))
	}
}

/* ----------------------------------------------------------------------
    updateBC: this is a connection between first layer of blockchain - ROOT NODE - on the second layer - xlsx file -
------------------------------------------------------------------------ */
func (bz *BaseDFS) updateSideChainBCTransactionQueueTake() {
	var err error
	var rows [][]string
	// --- reset
	bz.FirstSCQueueWait = 0

	f, err := excelize.OpenFile("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/sidechainbc.xlsx")
	if err != nil {
		log.Lvl2("Raha: ", err)
		panic(err)
	} else {
		log.Lvl3("opening side chain bc")
	}

	var accumulatedTxSize, txsize int
	blockIsFull := false
	BlockSizeMinusTransactions := blockchain.BlockMeasurement() // todo
	var TakeTime time.Time

	/* -----------------------------------------------------------------------------
		 -- take por transactions from sheet: FirstQueue
	----------------------------------------------------------------------------- */
	// 50008 means 50000 + 8 which means epoch number 5 scround number 8
	NextRow := strconv.Itoa(bz.SCRoundNumber + 2)
	axisQueue1IsFull := "I" + NextRow

	if rows, err = f.GetRows("FirstQueue"); err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}
	// reset variables
	txsize = 0
	blockIsFull = false
	accumulatedTxSize = 0

	numberOfPoRTx := 0
	axisBlockSize := "C" + NextRow
	axisNumPoRTx := "E" + NextRow
	axisAveFirstQueueWait := "H" + NextRow

	for i := len(rows); i > 1 && !blockIsFull; i-- {
		row := rows[i-1][:]
		/* each transaction has the following column stored on the transaction queue sheet:
		0) name
		1) size
		2) time
		3) issuedMCRoundNumber
		4) ServAgrId */
		for j, colCell := range row {
			if j == 1 {
				if txsize, err = strconv.Atoi(colCell); err != nil {
					log.Lvl2("Panic Raised:\n\n")
					panic(err)
				} else if accumulatedTxSize+txsize <= bz.BlockSize-BlockSizeMinusTransactions {
					accumulatedTxSize = accumulatedTxSize + txsize
					if row[0] == "TxPor" {
						log.Lvl3("a por tx added to block number", bz.MCRoundNumber, " from the queue")
						numberOfPoRTx++
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
					log.Lvl3("final result SC: side chain block is full! ")
					f.SetCellValue("RoundTable", axisQueue1IsFull, 1)
					break
				}
			}
		}
	}

	f.SetCellValue("RoundTable", axisNumPoRTx, numberOfPoRTx)

	log.Lvl2("In total in round number ", bz.SCRoundNumber+10000*(bz.MCRoundNumber/bz.EpochCount),
		"\n number of published PoR transactions is", numberOfPoRTx)
	TotalNumTxsInFirstQueue := numberOfPoRTx

	//-- accumulated block size
	// --- total throughput
	f.SetCellValue("RoundTable", axisBlockSize, accumulatedTxSize)

	f.SetCellValue("RoundTable", axisAveFirstQueueWait, bz.FirstQueueWait/TotalNumTxsInFirstQueue)

	log.Lvl3("final result SC: \n", " this round's block size: ", accumulatedTxSize)
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}

	//log.Lvl3("In total in round number ", bz.SCRoundNumber+10000*(bz.MCRoundNumber/bz.EpochCount),
	//	"\n number of all types of submitted txs is: ", TotalNumTxsInFirstQueue)

	// --------------------------------------------------------------------------------
	// ----------------------- OverallEvaluation Sheet --------------------
	// --------------------------------------------------------------------------------
	// ---- overall results

	axisRound := "A" + NextRow
	axisBCSize := "B" + NextRow
	axisOverallPoRTX := "C" + NextRow
	axisAveWaitOtherTx := "D" + NextRow
	axisOverallBlockSpaceFull := "E" + NextRow
	var FormulaString string

	err = f.SetCellValue("OverallEvaluation", axisRound, bz.SCRoundNumber+10000*(bz.MCRoundNumber/bz.EpochCount))
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}
	FormulaString = "=SUM(RoundTable!B2:B" + NextRow + ")"
	err = f.SetCellFormula("OverallEvaluation", axisBCSize, FormulaString)
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}

	FormulaString = "=SUM(RoundTable!C2:C" + NextRow + ")"
	err = f.SetCellFormula("OverallEvaluation", axisOverallPoRTX, FormulaString)
	if err != nil {
		log.Lvl2(err)
	}

	FormulaString = "=AVERAGE(RoundTable!D2:D" + NextRow + ")"
	err = f.SetCellFormula("OverallEvaluation", axisAveWaitOtherTx, FormulaString)
	if err != nil {
		log.Lvl2(err)
	}
	FormulaString = "=SUM(RoundTable!E2:E" + NextRow + ")"
	err = f.SetCellFormula("OverallEvaluation", axisOverallBlockSpaceFull, FormulaString)
	if err != nil {
		log.Lvl2(err)
	}

	// ----
	err = f.SaveAs("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/sidechainbc.xlsx")
	if err != nil {
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	} else {
		log.Lvl3("closing side bc")
		log.Lvl2(bz.Name(), " Finished taking transactions from side chain queue (FIFO) into new block in round number ", bz.SCRoundNumber+10000*(bz.MCRoundNumber/bz.EpochCount))
	}
}
