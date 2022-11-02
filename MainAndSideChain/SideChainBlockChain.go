package MainAndSideChain

import (
	"math"
	"time"

	"github.com/chainBoostScale/ChainBoost/MainAndSideChain/blockchain"
	"github.com/chainBoostScale/ChainBoost/onet/log"
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

	_, info, err := blockchain.SideChainRoundTableGetLastRow()
	if err != nil{
		panic(err)
	}
	var RoundIntervalSec int
	if (info == nil){
		RoundIntervalSec = int(time.Now().Unix())
		log.Lvl3("Final result SC: round number: ", 0, "took ", RoundIntervalSec, " seconds in total")
	} else {
		RoundIntervalSec = int(time.Now().Sub(info.StartTime).Seconds())
		log.Lvl3("Final result SC: round number: ", info.RoundNumber, "took ", RoundIntervalSec, " seconds in total")
	}
	

	err = blockchain.InsertIntoSideChainRoundTable(bz.SCRoundNumber, blocksize, LeaderName, 0, time.Now(), 0, 0, 0, RoundIntervalSec, bz.MCRoundNumber)
	if err != nil {
		panic(err)
	}
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
	fileinfo, err := blockchain.MainChainGetFileAndRoundInfo()
	if err != nil{
		log.Fatal(err)
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

	// this part can be moved to protocol initialization
	var PorTxSize uint32
	PorTxSize, _, _, _, _ = blockchain.TransactionMeasurement(bz.SectorNumber, bz.SimulationSeed)
	// ---

	var numOfPoRTxs = 0
	// --- check for eligible contracts and add a por tx row on top of the stream ----
	scFirstQueueTxs := make([]blockchain.SideChainFirstQueueEntry, 0)
	for i := range bz.Roster().List {
		// --------------------------------------
		if fileinfo[i].Published && bz.MCRoundNumber - fileinfo[i].StartedMcRoundNumber <= fileinfo[i].ServerAgrDuration {
			tx := blockchain.SideChainFirstQueueEntry{ Name : "TxPor", Size : int(PorTxSize), Time: time.Now(), IssuedScRoundNumber : bz.MCRoundNumber, ServAgrId : i+1, MCRoundNbr: bz.MCRoundNumber}
			scFirstQueueTxs = append(scFirstQueueTxs, tx)
			numOfPoRTxs++
		}
	}

	err = blockchain.BulkInsertIntoSideChainFirstQueue(scFirstQueueTxs)
	if err != nil {
		panic(err)
	}

	log.Lvl4(bz.Name(), "Final result SC: finished collecting new transactions to side chain queue in sc round number ", bz.SCRoundNumber)
	log.Lvl1("updateSideChainBCTransactionQueueCollect took:", time.Since(takenTime).String())
	log.Lvl1(numOfPoRTxs, "TxPor added to queue in sc round number: ", bz.SCRoundNumber)

}

/* ----------------------------------------------------------------------
    updateBC: this is a connection between first layer of blockchain - ROOT NODE - on the second layer - xlsx file -
------------------------------------------------------------------------ */
func (bz *ChainBoost) updateSideChainBCTransactionQueueTake() int {
	var err error
	takenTime := time.Now()
	//var epochNumber = int(math.Floor(float64(bz.MCRoundNumber) / float64(bz.MCRoundPerEpoch)))
	// --- reset
	bz.SideChainQueueWait = 0

	var accumulatedTxSize int
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
	blockIsFull = false
	accumulatedTxSize = 0

	numberOfPoRTx := 0

	rows, err := blockchain.SideChainGetFirstQueue()
	if err != nil{
		panic(err)
	}
	index :=0
	lastRowId :=0
	// len(rows) gives number of rows - the length of the "external" array
	for index=0; index <= len(rows) - 1 && !blockIsFull; index++ {
		row := rows[index]
		/* each transaction has the following column stored on the transaction queue sheet:
		0) name
		1) size
		2) time
		3) issuedMCRoundNumber
		4) ServAgrId */
		if accumulatedTxSize + row.Size <= bz.SideChainBlockSize - MetaBlockSizeMinusTransactions {
			accumulatedTxSize = accumulatedTxSize + row.Size
			if row.Name == "TxPor" {
				log.Lvl4("a por tx added to block number", bz.MCRoundNumber, " from the queue")
				numberOfPoRTx++
			} else {
				log.LLvl1("Panic Raised:\n\n")
				panic("the type of transaction in the queue is un-defined")
			}

			bz.SideChainQueueWait = bz.SideChainQueueWait + int(math.Abs(float64(bz.SCRoundNumber-row.IssuedScRoundNumber))) + bz.MCRoundDuration/bz.SCRoundDuration*(bz.MCRoundNumber-row.MCRoundNbr)
			lastRowId = row.RowId 


			bz.SummPoRTxs[row.ServAgrId] = bz.SummPoRTxs[row.ServAgrId] + 1
		} else {
			blockIsFull = true
			log.Lvl1("final result SC:\n side chain block is full! ")
			err = blockchain.SideChainRoundTableSetBlockSpaceIsFull(bz.SCRoundNumber)
			if err != nil {
				panic(err)
			}
			break
		}
	}
	err = blockchain.SideChainDeleteFromFirstQueue(lastRowId)
	if err != nil {
		panic(err)
	}
	var avgWait float64 = 0
	if numberOfPoRTx != 0 {
		avgWait = float64(bz.SideChainQueueWait)/float64(numberOfPoRTx)
	}
	err = blockchain.SideChainRoundTableSetFinalRoundInfo(accumulatedTxSize + MetaBlockSizeMinusTransactions,
													numberOfPoRTx,
												    avgWait,
													bz.SCRoundNumber)
	if err != nil {
		panic(err)
	}
	log.LLvl1("final result SC:\n In total in sc round number ", bz.SCRoundNumber,
		"\n number of published PoR transactions is", numberOfPoRTx)


	log.Lvl1("final result SC:\n", " this round's block size: ", accumulatedTxSize + MetaBlockSizeMinusTransactions)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	// fill OverallEvaluation Sheet
	updateSideChainBCOverallEvaluation(bz.SCRoundNumber)
	blocksize := accumulatedTxSize + MetaBlockSizeMinusTransactions
	log.Lvl1("updateSideChainBCTransactionQueueTake took:", time.Since(takenTime).String())
	return blocksize
}

// --------------------------------------------------------------------------------
// ----------------------- OverallEvaluation Sheet --------------------
// --------------------------------------------------------------------------------
func updateSideChainBCOverallEvaluation(SCRoundNumber int) {
	var err error
	takenTime := time.Now()

	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	sumBC, err := blockchain.GetSumSideChain("RoundTable", "BCSize")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	sumPoRTx, err := blockchain.GetSumSideChain("RoundTable", "PoRTx")
	if err != nil {
		log.LLvl1(err)
	}

	avgWaitTx, err := blockchain.GetAvgSideChain("RoundTable", "AveWait")
	if err != nil {
		log.LLvl1(err)
	}
	/*
	FormulaString = "=SUM(RoundTable!H2:H" + CurrentRow + ")"
	err = f.SetCellFormula("OverallEvaluation", axisOverallBlockSpaceFull, FormulaString)
	if err != nil {
		log.LLvl1(err)
	}*/

	err = blockchain.InsertIntoSideChainOverallEvaluation(SCRoundNumber, sumBC, sumPoRTx, avgWaitTx,0)
	if err != nil {
		panic(err)
	}
	log.Lvl1("updateSideChainBCOverallEvaluation took:", time.Since(takenTime).String())
}
