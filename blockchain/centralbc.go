package blockchain

import (
	log "github.com/basedfs/log"
	"github.com/xuri/excelize/v2"
	"math/rand"
	"strconv"
	"time"

	//"strings"
)
func generateNormalValues(variance , mean , nodes int) [] float64 {
	var list []float64
	rand.Seed(time.Now().UnixNano())
	for i := 1; i<=nodes;i++ {
		list = append(list, float64(mean) + float64(variance)*rand.NormFloat64())
	}
	return list
}

// InitializeCentralBC function is called in simulation level
func InitializeCentralBC(RoundDuration, PercentageTxPoR, PercentageTxPay, PercentageTxEscrow,
	DistributionMeanFileSize, DistributionVarianceFileSize,
	DistributionMeanContractDuration, DistributionVarianceContractDuration,
	Nodes,
	DistributionMeanInitialPower, DistributionVarianceInitialPower string) {
	/*
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
	*/
	// --------------------- generating normal distributed number based on config params ---------------------
	intVar, _ := strconv.Atoi(Nodes)
	numberOfNodes := intVar
	// --------------------- distribution of market matching information
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
	// ------------------------- Market Matching sheet  ------------------
	index := f.NewSheet("MarketMatching")
	err := f.SetColWidth("MarketMatching", "A", "H", 20)
	// Set active sheet of the workbook.
	f.SetActiveSheet(index)
	// --------------------------------------------------------------------
	err = f.SetCellValue("MarketMatching", "A1", "Node Info")
	if err != nil {
		return
	}

	err = f.SetCellValue("MarketMatching", "A2", "FileSize")
	if err != nil {
		return
	}
	err = f.SetSheetRow("MarketMatching", "B2", &FileSizeRow)
	if err != nil {
		log.LLvl2(err)
	}

	err = f.SetCellValue("MarketMatching", "A3", "ContractDuration")
	if err != nil {
		return
	}
	err = f.SetSheetRow("MarketMatching", "B3", &ContractDurationRow)
	if err != nil {
		log.LLvl2(err)
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
	// ------------------------- power table sheet  -----------------------
	index = f.NewSheet("PowerTable")
	err = f.SetColWidth("PowerTable", "A", "H", 20)
	// Set active sheet of the workbook.
	f.SetActiveSheet(index)
	// --------------------- distribution of initial power -------------------
	intVar, _ = strconv.Atoi(DistributionMeanInitialPower)
	MeanInitialPower := intVar
	intVar, _ = strconv.Atoi(DistributionVarianceInitialPower)
	VarianceInitialPower := intVar
	InitialPowerRow := generateNormalValues(VarianceInitialPower, MeanInitialPower, numberOfNodes)
	// -----------------------    Filling the table   --------------------------
	err = f.SetCellValue("PowerTable", "A1", "Round#/NodeInfo")
	if err != nil {
		return
	}
    // initial powers
	err = f.SetCellValue("PowerTable", "A2", "0")
	if err != nil {
		return
	}
	err = f.SetSheetRow("PowerTable", "B2", &InitialPowerRow)
	if err != nil {
		log.LLvl2(err)
	}
	// --------------------------------------------------------------------
	// --------------------------------------------------------------------
	// ---------------------Round Table Sheet ---------------------------
	index = f.NewSheet("RoundTable")
	err = f.SetColWidth("RoundTable", "A", "H", 20)
	// Set active sheet of the workbook.
	f.SetActiveSheet(index)
	// -----------------------    Filling the table   --------------------------
	err = f.SetCellValue("RoundTable", "A1", "Round#")
	if err != nil {
		return
	}
	err = f.SetCellValue("RoundTable", "A2", 0)
	if err != nil {
		return
	}
	err = f.SetCellValue("RoundTable", "B1", "Seed")
	if err != nil {
		return
	}
	err = f.SetCellValue("RoundTable", "B2", rand.Intn(100))
	if err != nil {
		return
	}
	err = f.SetCellValue("RoundTable", "C1", "BCSize")
	if err != nil {
		log.LLvl2(err)
	}
	err = f.SetCellValue("RoundTable", "C2", 0)
	if err != nil {
		log.LLvl2(err)
	}
	// --------------------------------------------------------------------
	if err := f.SaveAs("/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build/centralbc.xlsx"); err != nil {
		log.LLvl2(err)
	}
	log.LLvl2(RoundDuration, PercentageTxPoR, PercentageTxPay, PercentageTxEscrow, DistributionMeanFileSize, DistributionVarianceFileSize, DistributionMeanContractDuration, DistributionVarianceContractDuration, DistributionMeanInitialPower, DistributionVarianceInitialPower)
}