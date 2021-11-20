package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"time"

	log "github.com/basedfs/log"
	"github.com/xuri/excelize/v2"
	//"strings"
)

// generateNormalValues  generates values that follow a normal distribution with specified variance and mean
func generateNormalValues(variance, mean, nodes int) []uint64 {
	var list []float64
	var intlist []uint64
	rand.Seed(time.Now().UnixNano())
	for i := 1; i <= nodes; i++ {
		list = append(list, float64(mean)+float64(variance)*rand.NormFloat64())
	}
	for i := 0; i < len(list); i++ {
		t := uint64(math.Round(list[i]))
		if t == 0 {
			intlist = append(intlist, 1)
		} else {
			intlist = append(intlist, t)
		}
	}
	return intlist
}

// InitializeCentralBC function is called in simulation level
// Nodes: # of nodes
func InitializeCentralBC(RoundDuration, PercentageTxPoR, PercentageTxPay, PercentageTxEscrow,
	DistributionMeanFileSize, DistributionVarianceFileSize,
	DistributionMeanContractDuration, DistributionVarianceContractDuration,
	DistributionMeanInitialPower, DistributionVarianceInitialPower,
	Nodes, BlockSize string) {
	// --------------------- generating normal distributed number based on config params ---------------------
	intVar, _ := strconv.Atoi(Nodes)
	numberOfNodes := intVar
	// --------------------- distribution of market matching information ---------------------
	intVar, _ = strconv.Atoi(DistributionVarianceFileSize)
	VarianceFileSize := intVar
	intVar, _ = strconv.Atoi(DistributionMeanFileSize)
	MeanFileSize := intVar
	FileSizeRow := generateNormalValues(VarianceFileSize, MeanFileSize, numberOfNodes)
	intVar, _ = strconv.Atoi(DistributionVarianceContractDuration)
	VarianceContractDuration := intVar
	intVar, _ = strconv.Atoi(DistributionMeanContractDuration)
	MeanContractDuration := intVar
	ContractDurationRow := generateNormalValues(VarianceContractDuration, MeanContractDuration, numberOfNodes)
	//--------------------- fill the centralbc file with generated numbers  ---------------------
	f := excelize.NewFile()
	var index int
	var err error
	// ---------------------------------------------------------------------------
	// ------------------------- Market Matching sheet  ------------------
	// ---------------------------------------------------------------------------
	index = f.NewSheet("MarketMatching")
	if err = f.SetColWidth("MarketMatching", "A", "AAA", 50); err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	// Set active sheet of the workbook.
	f.SetActiveSheet(index)
	// --------------------------------------------------------------------
	err = f.SetCellValue("MarketMatching", "A1", "Server's Info")
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}

	err = f.SetCellValue("MarketMatching", "B1", "FileSize")
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	for i := 2; i <= len(FileSizeRow)+1; i++ {
		contractRow := strconv.Itoa(i)
		t := "B" + contractRow
		err = f.SetCellValue("MarketMatching", t, FileSizeRow[i-2])
	}
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}

	err = f.SetCellValue("MarketMatching", "C1", "ContractDuration")
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	for i := 2; i <= len(ContractDurationRow)+1; i++ {
		contractRow := strconv.Itoa(i)
		t := "C" + contractRow
		err = f.SetCellValue("MarketMatching", t, ContractDurationRow[i-2])
	}
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}

	err = f.SetCellValue("MarketMatching", "D1", "StartedRoundNumber")
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	for i := 2; i <= len(ContractDurationRow)+1; i++ {
		contractRow := strconv.Itoa(i)
		t := "D" + contractRow
		err = f.SetCellValue("MarketMatching", t, 0)
	}
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}

	// later we want to ad market matching transaction and compelete contract info in bc
	err = f.SetCellValue("MarketMatching", "E1", "ContractID")
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	r := rand.New(rand.NewSource(99))
	for i := 2; i <= numberOfNodes+1; i++ {
		contractRow := strconv.Itoa(i)
		t := "E" + contractRow
		if err = f.SetCellValue("MarketMatching", t, r.Int63()); err != nil {
			log.LLvl2("Panic Raised:\n\n")
			panic(err)
		}
	}
	err = f.SetCellValue("MarketMatching", "F1", "Client'sPK")
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}

	// once the escrow creation transaction leave the transaction queue (the contract get published) the contract will be activated
	// contract start round number will be set to current round
	//todo: what about the first time?
	if err = f.SetCellValue("MarketMatching", "G1", "Published"); err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	for i := 2; i <= numberOfNodes+1; i++ {
		contractRow := strconv.Itoa(i)
		t := "G" + contractRow
		if err = f.SetCellValue("MarketMatching", t, 0); err != nil {
			log.LLvl2("Panic Raised:\n\n")
			panic(err)
		}
	}
	// ---------------------------------------------------------------------------
	// ------------------------- Transaction Queue sheet  ------------------
	// ---------------------------------------------------------------------------
	index = f.NewSheet("TransactionQueue")
	if err = f.SetColWidth("TransactionQueue", "A", "AAA", 50); err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	// Set active sheet of the workbook.
	f.SetActiveSheet(index)
	// --------------------------------------------------------------------
	err = f.SetCellValue("TransactionQueue", "A1", "Name")
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}

	err = f.SetCellValue("TransactionQueue", "B1", "Size")
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}

	err = f.SetCellValue("TransactionQueue", "C1", "Time")
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}

	err = f.SetCellValue("TransactionQueue", "D1", "IssuedRoundNumber")
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	err = f.SetCellValue("TransactionQueue", "E1", "ContractId")
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	// ---------------------------------------------------------------------------
	// ------------------------- power table sheet  -----------------------
	// ---------------------------------------------------------------------------
	index = f.NewSheet("PowerTable")
	if err = f.SetColWidth("PowerTable", "A", "AAA", 50); err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	// Set active sheet of the workbook.
	f.SetActiveSheet(index)
	// --------------------- distribution of initial power -------------------
	intVar, _ = strconv.Atoi(DistributionVarianceInitialPower)
	VarianceInitialPower := intVar
	intVar, _ = strconv.Atoi(DistributionMeanInitialPower)
	MeanInitialPower := intVar
	InitialPowerRow := generateNormalValues(VarianceInitialPower, MeanInitialPower, numberOfNodes)
	/*var InitialPowerRowString []string
	for i:=0;i<len(InitialPowerRow);i++{
		InitialPowerRowString = append(InitialPowerRowString,InitialPowerRow[i])
	}*/
	// -----------------------    Filling Power Table's Headers   -----------
	err = f.SetCellValue("PowerTable", "A1", "Round#/NodeInfo")
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	// initial powers
	err = f.SetCellValue("PowerTable", "A2", "1")
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	// -----------------------    Filling Power Table's first row ------------
	err = f.SetSheetRow("PowerTable", "B2", &InitialPowerRow)
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	// --------------------------------------------------------------------
	// --------------------- Round Table Sheet ---------------------------
	// --------------------------------------------------------------------
	index = f.NewSheet("RoundTable")
	if err = f.SetColWidth("RoundTable", "A", "AAA", 50); err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	// Set active sheet of the workbook.
	f.SetActiveSheet(index)
	// -----------------------    Filling Round Table's Headers ------------
	err = f.SetCellValue("RoundTable", "A1", "Round#")
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	err = f.SetCellValue("RoundTable", "B1", "Seed")
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	err = f.SetCellValue("RoundTable", "C1", "BCSize")
	if err != nil {
		log.LLvl2(err)
	}
	// -----------------------    Filling Round Table's first row +
	err = f.SetCellValue("RoundTable", "A2", 1)
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	t := rand.Intn(100)
	err = f.SetCellValue("RoundTable", "B2", t)
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	err = f.SetCellValue("RoundTable", "C2", 0)
	if err != nil {
		log.LLvl2(err)
	}
	// -----------------------    Filling Round Table's second row: next round's seed
	err = f.SetCellValue("RoundTable", "A3", 2)
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}
	// ---  next round's seed is the hash of current seed
	data := fmt.Sprintf("%v", t)
	sha := sha256.New()
	if _, err := sha.Write([]byte(data)); err != nil {
		log.Error("Couldn't hash header:", err)
	}
	hash := sha.Sum(nil)
	if err = f.SetCellValue("RoundTable", "B3", hex.EncodeToString(hash)); err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}

	// --------------------------------------------------------------------
	if err := f.SaveAs("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx"); err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}

	log.LLvl2("Config params: \n RoundDuration: ", RoundDuration,
		"\n PercentageTxPoR: ", PercentageTxPoR,
		"\n PercentageTxPay: ", PercentageTxPay,
		"\n PercentageTxEscrow: ", PercentageTxEscrow,
		"\n DistributionMeanFileSize: ", DistributionMeanFileSize,
		"\n DistributionVarianceFileSize: ", DistributionVarianceFileSize,
		"\n DistributionMeanContractDuration: ", DistributionMeanContractDuration,
		"\n DistributionVarianceContractDuration: ", DistributionVarianceContractDuration,
		"\n DistributionMeanInitialPower: ", DistributionMeanInitialPower,
		"\n DistributionVarianceInitialPower: ", DistributionVarianceInitialPower,
		"\n BlockSize: ", BlockSize)
}

/*	// in case of initializing a column
for i := 1;i<=len(FileSizeColumn);i++ {
		axis = "B" + strconv.Itoa(i+1)
		f.SetCellValue("MarketMatching", axis, FileSizeColumn[i-1])
		axis = "C" + strconv.Itoa(i+1)
		f.SetCellValue("MarketMatching", axis, ContractDurationColumn[i-1])
	}
	now := time.Now()
	f2.SetCellValue("Sheet1", "A4", now.Format(time.ANSIC))
*/
