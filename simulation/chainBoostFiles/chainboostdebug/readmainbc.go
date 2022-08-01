package main

import (
	"fmt"
	"runtime"
	"time"
	//"net"
	//"time"
)

func main() {

	runtime.GOMAXPROCS(1)

	x := int64(0)
	go cpuIntensive(&x) // This should go into background
	go printVar(&x)

	// This won't get scheduled until everything has finished.
	time.Sleep(1 * time.Nanosecond) // Wait for goroutines to finish
}

func cpuIntensive(p *int64) {
	for i := int64(1); i <= 10000000; i++ {
		runtime.Gosched()
		*p = i
		fmt.Println(" i= ", i)
	}
	fmt.Println("Done intensive thing")
}

func printVar(p *int64) {
	fmt.Printf("print x = %d.\n", *p)
}

// //test tcp connection
// var dialTimeout = 1 * time.Minute
// //var c net.Conn
// netAddr := "192.168.3.212:2000"
// _, err = net.DialTimeout("tcp", netAddr, dialTimeout)
// if err != nil {
// 	fmt.Println("err:", err)
// } else {
// 	fmt.Println("sucees")
// }

// /* ----------------------------------------------------------------------
//     updateBC: this is a connection between first layer of blockchain - ROOT NODE - and the second layer - xlsx file -
// ------------------------------------------------------------------------ */
// //func (bz *ChainBoost) updateMainChainBCTransactionQueueTake() {
// var err error
// var rows [][]string
// // --- reset
// FirstQueueWait := 0
// SecondQueueWait := 0
// MCRoundNumber := 1
// SimState := 2

// // pwd, _ := os.Getwd()
// // fmt.Print("opening bc in:", pwd)
// // //bcDirectory := strings.Split(pwd, "/build")
// // bcDirectory := strings.Split(pwd, "/build")[0] + "/mainchainbc.xlsx"
// // fmt.Print("opening bc in:", bcDirectory)
// f, err := excelize.OpenFile("/root/remote/mainchainbc.xlsx")
// if err != nil {
// 	fmt.Print("problem while opening bc: " + err.Error())
// } else {
// 	fmt.Print("bc Successfully opened\n")
// }

// var accumulatedTxSize, txsize int
// blockIsFull := false
// NextRow := strconv.Itoa(MCRoundNumber + 2)

// axisNumRegPayTx := "E" + NextRow

// axisQueue2IsFull := "N" + NextRow
// axisQueue1IsFull := "O" + NextRow

// numberOfRegPayTx := 0
// //BlockSizeMinusTransactions := blockchain.BlockMeasurement()
// //var TakeTime time.Time

// /* -----------------------------------------------------------------------------
// 	-- take regular payment transactions from sheet: SecondQueue
// ----------------------------------------------------------------------------- */
// if rows, err = f.GetRows("SecondQueue"); err != nil {
// 	fmt.Print("Panic Raised:\n\n")
// 	panic(err)
// }
// PercentageTxPay := 20
// MainChainBlockSize := 2000
// BlockSizeMinusTransactions := 100
// for i := len(rows); i > 1 && !blockIsFull; i-- {
// 	row := rows[i-1][:]
// 	/* each transaction has the following column stored on the Transaction Queue Payment sheet:
// 	0) size
// 	1) time
// 	2) issuedMCRoundNumber */

// 	for j, colCell := range row {
// 		if j == 0 {
// 			if txsize, err = strconv.Atoi(colCell); err != nil {
// 				fmt.Print("Panic Raised:\n\n")
// 				panic(err)
// 			} else if 100*(accumulatedTxSize+txsize) <= (PercentageTxPay)*(MainChainBlockSize-BlockSizeMinusTransactions) {
// 				accumulatedTxSize = accumulatedTxSize + txsize
// 				numberOfRegPayTx++
// 				/* transaction name in transaction queue payment is just "TxPayment"
// 				the transactions are just removed from queue and their size are added to included transactions' size in block */
// 				fmt.Print("a regular payment transaction added to block number", MCRoundNumber, " from the queue")

// 				// row[1] is transaction's collected time
// 				// if TakeTime, err = time.Parse(time.RFC3339, row[1]); err != nil {
// 				// 	fmt.Print("Panic Raised:\n\n")
// 				// 	panic(err)
// 				// }
// 				//bz.SecondQueueWait = bz.SecondQueueWait + int(time.Now().Sub(TakeTime).Seconds())

// 				// row[2] is the MCRound Number that the transaction has been issued
// 				if x, err := strconv.Atoi(row[2]); err != nil {
// 					fmt.Print("Panic Raised:\n\n")
// 					panic(err)
// 				} else {
// 					SecondQueueWait = SecondQueueWait + MCRoundNumber - x
// 				}

// 				f.RemoveRow("SecondQueue", i)
// 			} else {
// 				blockIsFull = true
// 				fmt.Print("final result MC: regular  payment share is full!")
// 				f.SetCellValue("RoundTable", axisQueue2IsFull, 1)
// 				break
// 			}
// 		}
// 	}
// }
// allocatedBlockSizeForRegPayTx := accumulatedTxSize
// /* -----------------------------------------------------------------------------
// 	 -- take 5 types of transactions from sheet: FirstQueue
// ----------------------------------------------------------------------------- */

// if rows, err = f.GetRows("FirstQueue"); err != nil {
// 	fmt.Print("Panic Raised:\n\n")
// 	panic(err)
// }
// // reset variables
// txsize = 0
// blockIsFull = false
// accumulatedTxSize = 0
// // new variables for first queue
// var ServAgrIdCellMarketMatching []string

// numberOfPoRTx := 0
// numberOfStoragePayTx := 0
// numberOfServAgrProposeTx := 0
// numberOfServAgrCommitTx := 0
// numberOfSyncTx := 0

// axisBlockSize := "C" + NextRow

// axisNumPoRTx := "F" + NextRow
// axisNumStoragePayTx := "G" + NextRow
// axisNumServAgrProposeTx := "H" + NextRow
// axisNumServAgrCommitTx := "I" + NextRow
// axisNumSyncTx := "P" + NextRow

// axisTotalTxsNum := "K" + NextRow
// axisAveFirstQueueWait := "L" + NextRow
// axisAveSecondQueueWait := "M" + NextRow

// for i := len(rows); i > 1 && !blockIsFull; i-- {
// 	row := rows[i-1][:]
// 	/* each transaction has the following column stored on the transaction queue sheet:
// 	0) name
// 	1) size
// 	2) time
// 	3) issuedMCRoundNumber
// 	4) ServAgrId */
// 	for j, colCell := range row {
// 		if j == 1 {
// 			if txsize, err = strconv.Atoi(colCell); err != nil {
// 				fmt.Print("Panic Raised:\n\n")
// 				panic(err)
// 			} else if accumulatedTxSize+txsize <= MainChainBlockSize-BlockSizeMinusTransactions-allocatedBlockSizeForRegPayTx {
// 				accumulatedTxSize = accumulatedTxSize + txsize
// 				/* transaction name in transaction queue can be "TxServAgrPropose", "TxStoragePayment", or "TxPor"
// 				in case of "TxServAgrCommit":
// 				1) The corresponding ServAgr in marketmatching should be updated to published //ToDoRaha: replace the word "published" with "active"
// 				2) set start round number to current round
// 				other transactions are just removed from queue and their size are added to included transactions' size in block */
// 				if row[0] == "TxServAgrCommit" {
// 					/* when tx TxServAgrCommit left queue:
// 					1) set ServAgrPublished to True
// 					2) set start round number to current round */
// 					cid := row[4]
// 					if ServAgrIdCellMarketMatching, err = f.SearchSheet("MarketMatching", cid); err != nil {
// 						fmt.Print("Panic Raised:\n\n")
// 						panic(err)
// 					} else {
// 						publishedCellMarketMatching := "F" + ServAgrIdCellMarketMatching[0][1:]
// 						err = f.SetCellValue("MarketMatching", publishedCellMarketMatching, 1)
// 						if err != nil {
// 							fmt.Print("Panic Raised:\n\n")
// 							panic(err)
// 						} else {
// 							startRoundCellMarketMatching := "D" + ServAgrIdCellMarketMatching[0][1:]
// 							err = f.SetCellValue("MarketMatching", startRoundCellMarketMatching, MCRoundNumber)
// 							if err != nil {
// 								fmt.Print("Panic Raised:\n\n")
// 								panic(err)
// 							} else {
// 								fmt.Print("a TxServAgrCommit tx added to block number", MCRoundNumber, " from the queue")
// 								numberOfServAgrCommitTx++
// 							}
// 						}
// 					}
// 				} else if row[0] == "TxStoragePayment" {
// 					fmt.Print("a TxStoragePayment tx added to block number", MCRoundNumber, " from the queue")
// 					numberOfStoragePayTx++
// 					// && bz.SimState == 1 is for backup check - the first condition shouldn't be true if the second one isn't
// 				} else if row[0] == "TxPor" && SimState == 1 {
// 					fmt.Print("a por tx added to block number", MCRoundNumber, " from the queue")
// 					numberOfPoRTx++
// 				} else if row[0] == "TxServAgrPropose" {
// 					fmt.Print("a TxServAgrPropose tx added to block number", MCRoundNumber, " from the queue")
// 					numberOfServAgrProposeTx++
// 				} else if row[0] == "TxSync" {
// 					fmt.Print("a sync tx added to block number", MCRoundNumber, " from the queue")
// 					numberOfSyncTx++
// 					numberOfPoRTx, _ = strconv.Atoi(row[4])
// 				} else {
// 					fmt.Print("Panic Raised:\n\n")
// 					panic("the type of transaction in the queue is un-defined")
// 				}

// 				// when performance was being measured based on time!
// 				// if TakeTime, err = time.Parse(time.RFC3339, row[2]); err != nil {
// 				// 	fmt.Print("Panic Raised:\n\n")
// 				// 	panic(err)
// 				// }
// 				//bz.FirstQueueWait = bz.FirstQueueWait + int(time.Now().Sub(TakeTime).Seconds())

// 				// row[3] is the MCRound Number that the transaction has been issued
// 				if x, err := strconv.Atoi(row[3]); err != nil {
// 					fmt.Print("Panic Raised:\n\n")
// 					panic(err)
// 				} else {
// 					FirstQueueWait = FirstQueueWait + MCRoundNumber - x
// 				}
// 				// remove transaction from the top of the queue (older ones)
// 				f.RemoveRow("FirstQueue", i)
// 			} else {
// 				blockIsFull = true
// 				fmt.Print("final result MC: block is full! ")
// 				f.SetCellValue("RoundTable", axisQueue1IsFull, 1)
// 				break
// 			}
// 		}
// 	}
// }

// f.SetCellValue("RoundTable", axisNumPoRTx, numberOfPoRTx)
// f.SetCellValue("RoundTable", axisNumStoragePayTx, numberOfStoragePayTx)
// f.SetCellValue("RoundTable", axisNumServAgrProposeTx, numberOfServAgrProposeTx)
// f.SetCellValue("RoundTable", axisNumServAgrCommitTx, numberOfServAgrCommitTx)
// f.SetCellValue("RoundTable", axisNumSyncTx, numberOfSyncTx)

// fmt.Print("In total in round number ", MCRoundNumber,
// 	"\n number of published PoR transactions is", numberOfPoRTx,
// 	"\n number of published Storage payment transactions is", numberOfStoragePayTx,
// 	"\n number of published Propose ServAgr transactions is", numberOfServAgrProposeTx,
// 	"\n number of published Commit ServAgr transactions is", numberOfServAgrCommitTx,
// 	"\n number of published regular payment transactions is", numberOfRegPayTx,
// )
// TotalNumTxsInBothQueue := numberOfPoRTx + numberOfStoragePayTx + numberOfServAgrProposeTx + numberOfServAgrCommitTx + numberOfRegPayTx
// TotalNumTxsInFirstQueue := numberOfPoRTx + numberOfStoragePayTx + numberOfServAgrProposeTx + numberOfServAgrCommitTx

// //-- accumulated block size
// // --- total throughput
// f.SetCellValue("RoundTable", axisNumRegPayTx, numberOfRegPayTx)
// f.SetCellValue("RoundTable", axisBlockSize, accumulatedTxSize+allocatedBlockSizeForRegPayTx+BlockSizeMinusTransactions)
// f.SetCellValue("RoundTable", axisTotalTxsNum, TotalNumTxsInBothQueue)
// if TotalNumTxsInFirstQueue != 0 {
// 	f.SetCellValue("RoundTable", axisAveFirstQueueWait, float64(FirstQueueWait)/float64(TotalNumTxsInFirstQueue))
// }
// if numberOfRegPayTx != 0 {
// 	f.SetCellValue("RoundTable", axisAveSecondQueueWait, float64(SecondQueueWait)/float64(numberOfRegPayTx))
// }

// fmt.Print("final result MC: \n", allocatedBlockSizeForRegPayTx,
// 	" for regular payment txs,\n and ", accumulatedTxSize, " for other types of txs")
// if err != nil {
// 	fmt.Print("Panic Raised:\n\n")
// 	panic(err)
// }

// fmt.Print("In total in round number ", MCRoundNumber,
// 	"\n number of all types of submitted txs is", TotalNumTxsInBothQueue)

// // ---- overall results
// axisRound := "A" + NextRow
// axisBCSize := "B" + NextRow
// axisOverallRegPayTX := "C" + NextRow
// axisOverallPoRTX := "D" + NextRow
// axisOverallStorjPayTX := "E" + NextRow
// axisOverallCntPropTX := "F" + NextRow
// axisOverallCntComTX := "G" + NextRow
// axisAveWaitOtherTx := "H" + NextRow
// axisOveralAveWaitRegPay := "I" + NextRow
// axisOverallBlockSpaceFull := "J" + NextRow
// var FormulaString string

// err = f.SetCellValue("OverallEvaluation", axisRound, MCRoundNumber)
// if err != nil {
// 	fmt.Print("Panic Raised:\n\n")
// 	panic(err)
// }
// FormulaString = "=SUM(RoundTable!C2:C" + NextRow + ")"
// err = f.SetCellFormula("OverallEvaluation", axisBCSize, FormulaString)
// if err != nil {
// 	fmt.Print("Panic Raised:\n\n")
// 	panic(err)
// }
// FormulaString = "=SUM(RoundTable!E2:E" + NextRow + ")"
// err = f.SetCellFormula("OverallEvaluation", axisOverallRegPayTX, FormulaString)
// if err != nil {
// 	fmt.Print(err)
// }
// FormulaString = "=SUM(RoundTable!F2:F" + NextRow + ")"
// err = f.SetCellFormula("OverallEvaluation", axisOverallPoRTX, FormulaString)
// if err != nil {
// 	fmt.Print(err)
// }
// FormulaString = "=SUM(RoundTable!G2:G" + NextRow + ")"
// err = f.SetCellFormula("OverallEvaluation", axisOverallStorjPayTX, FormulaString)
// if err != nil {
// 	fmt.Print(err)
// }
// FormulaString = "=SUM(RoundTable!H2:H" + NextRow + ")"
// err = f.SetCellFormula("OverallEvaluation", axisOverallCntPropTX, FormulaString)
// if err != nil {
// 	fmt.Print(err)
// }
// FormulaString = "=SUM(RoundTable!I2:I" + NextRow + ")"
// err = f.SetCellFormula("OverallEvaluation", axisOverallCntComTX, FormulaString)
// if err != nil {
// 	fmt.Print(err)
// }
// FormulaString = "=AVERAGE(RoundTable!L2:L" + NextRow + ")"
// err = f.SetCellFormula("OverallEvaluation", axisAveWaitOtherTx, FormulaString)
// if err != nil {
// 	fmt.Print(err)
// }
// FormulaString = "=AVERAGE(RoundTable!M2:M" + NextRow + ")"
// err = f.SetCellFormula("OverallEvaluation", axisOveralAveWaitRegPay, FormulaString)
// if err != nil {
// 	fmt.Print(err)
// }
// FormulaString = "=SUM(RoundTable!O2:O" + NextRow + ")"
// err = f.SetCellFormula("OverallEvaluation", axisOverallBlockSpaceFull, FormulaString)
// if err != nil {
// 	fmt.Print(err)
// }
// // ----
// //err = f.SaveAs("/root/remote/mainchainbc.xlsx")
// err = f.SaveAs("/root/remote/mainchainbc.xlsx")

// if err != nil {
// 	fmt.Print("Panic Raised:\n\n")
// 	panic(err)
// } else {
// 	fmt.Print("bc Successfully closed")
// 	fmt.Print("Raha:::", " Finished taking transactions from queue (FIFO) into new block in round number ", MCRoundNumber)
// }

// fmt.Print("------------------------------------------------------------- \n ")
// fmt.Print("------------------------------ MAIN CHAIN -------------------- \n ")
// fmt.Print("------------------------------------------------------------- \n ")

// var rows [][]string

// xlsx, err := excelize.OpenFile("/root/remote/mainchainbc.xlsx")
// if err != nil {
// 	fmt.Println(err)
// 	return
// }

// fmt.Print("----------------------  MarketMatching ------------------- \n ")
// rows, _ = xlsx.GetRows("MarketMatching")
// for _, row := range rows {
// 	for _, colCell := range row {
// 		fmt.Print(colCell, "\t")
// 	}
// 	fmt.Println()
// }

// fmt.Print("----------------------  FirstQueue ------------------- \n ")
// rows, err = xlsx.GetRows("FirstQueue")
// for _, row := range rows {
// 	for _, colCell := range row {
// 		fmt.Print(colCell, "\t")
// 	}
// 	fmt.Println()
// }

// fmt.Print("----------------------  SecondQueue ------------------- \n ")
// rows, _ = xlsx.GetRows("SecondQueue")
// for _, row := range rows {
// 	for _, colCell := range row {
// 		fmt.Print(colCell, "\t")
// 	}
// 	fmt.Println()
// }

// fmt.Print("----------------------  PowerTable ------------------- \n ")
// rows, _ = xlsx.GetRows("PowerTable")
// for _, row := range rows {
// 	for _, colCell := range row {
// 		fmt.Print(colCell, "\t")
// 	}
// 	fmt.Println()
// }

// fmt.Print("----------------------  RoundTable ------------------- \n ")
// rows, _ = xlsx.GetRows("RoundTable")
// for _, row := range rows {
// 	for _, colCell := range row {
// 		fmt.Print(colCell, "\t")
// 	}
// 	fmt.Println()
// }

// fmt.Print("----------------------  OverallEvaluation ------------------- \n ")
// rows, _ = xlsx.GetRows("OverallEvaluation")
// for _, row := range rows {
// 	for _, colCell := range row {
// 		fmt.Print(colCell, "\t")
// 	}
// 	fmt.Println()
// }

// fmt.Print("------------------------------------------------------------- \n ")
// fmt.Print("------------------------------ SIDE CHAIN -------------------- \n ")
// fmt.Print("------------------------------------------------------------- \n ")

// xlsx, _ = excelize.OpenFile("/root/remote/sidechainbc.xlsx")
// if err != nil {
// 	fmt.Println(err)
// 	return
// }
// fmt.Print("----------------------  MarketMatching ------------------- \n ")
// rows, _ = xlsx.GetRows("MarketMatching")
// for _, row := range rows {
// 	for _, colCell := range row {
// 		fmt.Print(colCell, "\t")
// 	}
// 	fmt.Println()
// }

// fmt.Print("----------------------  FirstQueue ------------------- \n ")
// rows, _ = xlsx.GetRows("FirstQueue")
// for _, row := range rows {
// 	for _, colCell := range row {
// 		fmt.Print(colCell, "\t")
// 	}
// 	fmt.Println()
// }

// fmt.Print("----------------------  RoundTable ------------------- \n ")
// rows, _ = xlsx.GetRows("RoundTable")
// for _, row := range rows {
// 	for _, colCell := range row {
// 		fmt.Print(colCell, "\t")
// 	}
// 	fmt.Println()
// }

// fmt.Print("----------------------  OverallEvaluation ------------------- \n ")
// rows, _ = xlsx.GetRows("OverallEvaluation")
// for _, row := range rows {
// 	for _, colCell := range row {
// 		fmt.Print(colCell, "\t")
// 	}
// 	fmt.Println()
// }
