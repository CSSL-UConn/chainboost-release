package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	log "github.com/basedfs/log"
	"github.com/xuri/excelize/v2"
	"math"
	"math/rand"
	"strconv"
	"time"

	//"strings"
)

// generateNormalValues  generates values that follow a normal distribution with specified variance and mean
func generateNormalValues(variance , mean , nodes int) [] int {
	var list []float64
	var intlist [] int
	rand.Seed(time.Now().UnixNano())
	for i := 1; i<=nodes;i++ {
		list = append(list, float64(mean) + float64(variance)*rand.NormFloat64())
	}
	for i := 0; i< len(list);i++{
		t := int(math.Round(list[i]))
		intlist = append(intlist, t )
	}
	return intlist
}

/*func generateRandomValuesBigInt(nodes int) [] *big.Int {
	var bigIntlist [] *big.Int
	//------------------------------  a sample vrfOutput ------------------
	var vrfOutput [64]byte
	var bi *big.Int
	seed := rand.Intn(100) //ToDo: Hash
	toBeHashed:= []byte(strconv.Itoa(seed))
	_, ECPrivateKey := crypto.VrfKeygen()
	proof, ok := ECPrivateKey.ProveBytes(toBeHashed[:])
	if !ok {
		log.LLvl2("error while generating proof")
	}
	_, vrfOutput = ECPrivateKey.Pubkey().VerifyBytes(proof, toBeHashed[:])
	bi = new(big.Int).SetBytes(vrfOutput[:])
	// --------------------------------------------------------------------
	for i := 1; i<= nodes;i++{
		bigIntlist = append(bigIntlist, new(big.Int).Rand(rand.New(rand.NewSource(time.Now().UnixNano())),bi ))
	}
	return bigIntlist
}*/
// InitializeCentralBC function is called in simulation level

func InitializeCentralBC(RoundDuration, PercentageTxPoR, PercentageTxPay, PercentageTxEscrow,
	DistributionMeanFileSize, DistributionVarianceFileSize,
	DistributionMeanContractDuration, DistributionVarianceContractDuration,
	Nodes,BlockSize string) {
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
	// ---------------------------------------------------------------------------
	// ------------------------- Market Matching sheet  ------------------
	// ---------------------------------------------------------------------------
	index := f.NewSheet("MarketMatching")
	err := f.SetColWidth("MarketMatching", "A", "H", 20)
	// Set active sheet of the workbook.
	f.SetActiveSheet(index)
	// --------------------------------------------------------------------
	err = f.SetCellValue("MarketMatching", "A1", "Server's Info")
	if err != nil {
		return
	}

	err = f.SetCellValue("MarketMatching", "B1", "FileSize")
	if err != nil {
		return
	}
	for i := 2; i <= len(FileSizeRow)+1 ; i++{
		contractRow := strconv.Itoa(i)
		t := "B" + contractRow
		err = f.SetCellValue("MarketMatching", t , FileSizeRow[i-2])
	}
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}

	err = f.SetCellValue("MarketMatching", "C1", "ContractDuration")
	if err != nil {
		return
	}
	for i := 2; i <= len(ContractDurationRow)+1 ; i++{
		contractRow := strconv.Itoa(i)
		t := "C" + contractRow
		err = f.SetCellValue("MarketMatching", t , ContractDurationRow[i-2])
	}
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}

	err = f.SetCellValue("MarketMatching", "D1", "RoundNumber")
	if err != nil {
		return
	}
	for i := 2; i <= len(ContractDurationRow)+2 ; i++{
		contractRow := strconv.Itoa(i)
		t := "D" + contractRow
		err = f.SetCellValue("MarketMatching", t , 1)
	}
	if err != nil {
		log.LLvl2("Panic Raised:\n\n")
		panic(err)
	}

	// later we want to ad market matching transaction and compelete contract info in bc
	err = f.SetCellValue("MarketMatching", "E1", "ContractID")
	if err != nil {
		return
	}
	err = f.SetCellValue("MarketMatching", "F1", "Client's PK")
	if err != nil {
		return
	}
	// ---------------------------------------------------------------------------
	// ------------------------- power table sheet  -----------------------
	// ---------------------------------------------------------------------------
	index = f.NewSheet("PowerTable")
	err = f.SetColWidth("PowerTable", "A", "H", 20)
	// Set active sheet of the workbook.
	f.SetActiveSheet(index)
	// --------------------- distribution of initial power -------------------
	intVar, _ = strconv.Atoi(DistributionVarianceFileSize)
	VarianceFileSize = intVar
	intVar, _ = strconv.Atoi(DistributionMeanFileSize)
	MeanFileSize = intVar
	InitialPowerRow := generateNormalValues(VarianceFileSize , MeanFileSize , numberOfNodes) //ToDo: re-add! var and mean
	/*var InitialPowerRowString []string
	for i:=0;i<len(InitialPowerRow);i++{
		InitialPowerRowString = append(InitialPowerRowString,InitialPowerRow[i])
	}*/
	// -----------------------    Filling Power Table's Headers   -----------
	err = f.SetCellValue("PowerTable", "A1", "Round#/NodeInfo")
	if err != nil {
		return
	}
    // initial powers
	err = f.SetCellValue("PowerTable", "A2", "1")
	if err != nil {
		return
	}
	// -----------------------    Filling Power Table's first row ------------
	err = f.SetSheetRow("PowerTable", "B2", &InitialPowerRow)
	if err != nil {
		log.LLvl2(err)
	}
	// --------------------------------------------------------------------
	// ---------------------Round Table Sheet ---------------------------
	// --------------------------------------------------------------------
	index = f.NewSheet("RoundTable")
	err = f.SetColWidth("RoundTable", "A", "H", 20)
	// Set active sheet of the workbook.
	f.SetActiveSheet(index)
	// -----------------------    Filling Round Table's Headers ------------
	err = f.SetCellValue("RoundTable", "A1", "Round#")
	if err != nil {
		return
	}
	err = f.SetCellValue("RoundTable", "B1", "Seed")
	if err != nil {
		return
	}
	err = f.SetCellValue("RoundTable", "C1", "BCSize")
	if err != nil {
		log.LLvl2(err)
	}
	// -----------------------    Filling Round Table's first row +
	err = f.SetCellValue("RoundTable", "A2", 1)
	if err != nil {
		return
	}
	t := rand.Intn(100)
	err = f.SetCellValue("RoundTable", "B2", t)
	if err != nil {
		return
	}
	err = f.SetCellValue("RoundTable", "C2", 0)
	if err != nil {
		log.LLvl2(err)
	}
	// -----------------------    Filling Round Table's second row: next round's seed
	err = f.SetCellValue("RoundTable", "A3", 2)
	if err != nil {
		return
	}
	// ---  next round's seed is the hash of current seed
	data := fmt.Sprintf("%v", t)
	sha := sha256.New()
	if _, err := sha.Write([]byte(data)); err != nil {
		log.Error("Couldn't hash header:", err)
	}
	hash := sha.Sum(nil)
	err = f.SetCellValue("RoundTable", "B3", hex.EncodeToString(hash))
	// --------------------------------------------------------------------
	if err := f.SaveAs("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx"); err != nil {
		log.LLvl2(err)
	}

	log.LLvl2("Config params: \n RoundDuration: ",RoundDuration ,
		"\n PercentageTxPoR:", PercentageTxPoR,
		"\n PercentageTxPay: ", PercentageTxPay ,
		"\n PercentageTxEscrow: ", PercentageTxEscrow,
		"\n DistributionMeanFileSize: ", DistributionMeanFileSize,
		"\n DistributionVarianceFileSize: ", DistributionVarianceFileSize,
		"\n DistributionMeanContractDuration: ", DistributionMeanContractDuration,
		"\n DistributionVarianceContractDuration: ", DistributionVarianceContractDuration,
		"\n BlockSize: ", BlockSize,)
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