package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strconv"

	log "github.com/chainBoostScale/ChainBoost/onet/log"
	"github.com/xuri/excelize/v2"
)

// generateNormalValues  generates values that follow a normal distribution with specified variance and mean
func generateNormalValues(variance, mean, nodes, SimulationSeed int) []uint64 {
	var list []float64
	var intlist []uint64
	rand.Seed(int64(SimulationSeed))
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

// InitializeMainChain function is called in simulation level
func InitializeMainChainBC(
	FileSizeDistributionMean, FileSizeDistributionVariance,
	ServAgrDurationDistributionMean, ServAgrDurationDistributionVariance,
	InitialPowerDistributionMean, InitialPowerDistributionVariance,
	Nodes, SimulationSeed string) {
	// --------------------- generating normal distributed number based on config params ---------------------
	intVar, _ := strconv.Atoi(Nodes)
	numberOfNodes := intVar
	// --------------------- distribution of market matching information ---------------------
	intVar, _ = strconv.Atoi(FileSizeDistributionVariance)
	VarianceFileSize := intVar
	intVar, _ = strconv.Atoi(FileSizeDistributionMean)
	MeanFileSize := intVar
	SimulationSeedInt, _ := strconv.Atoi(SimulationSeed)
	FileSizeRow := generateNormalValues(VarianceFileSize, MeanFileSize, numberOfNodes, SimulationSeedInt)
	intVar, _ = strconv.Atoi(ServAgrDurationDistributionVariance)
	VarianceServAgrDuration := intVar
	intVar, _ = strconv.Atoi(ServAgrDurationDistributionMean)
	MeanServAgrDuration := intVar
	ServAgrDurationRow := generateNormalValues(VarianceServAgrDuration, MeanServAgrDuration, numberOfNodes, SimulationSeedInt)
	//--------------------- fill the mainchainbc file with generated numbers  ---------------------
	f := excelize.NewFile()
	var err error
	// ---------------------------------------------------------------------------
	// ------------------------- Market Matching sheet  ------------------
	// ---------------------------------------------------------------------------
	_ = f.NewSheet("MarketMatching")
	if err = f.SetColWidth("MarketMatching", "A", "AAA", 25); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	style, errstyle := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			WrapText: true,
		},
	})
	if errstyle != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	if err := f.SetSheetPrOptions("MarketMatching",
		excelize.FitToPage(true),
		excelize.TabColor("#FFFF00"),
		excelize.AutoPageBreaks(true),
	); err != nil {
		fmt.Println(err)
	}
	err = f.SetRowHeight("MarketMatching", 1, 30)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	f.SetCellStyle("MarketMatching", "A1", "AAA1", style)
	// --------------------------------------------------------------------
	err = f.SetCellValue("MarketMatching", "A1", "Server's Info")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	err = f.SetCellValue("MarketMatching", "B1", "FileSize")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	if err = f.SetColWidth("MarketMatching", "B", "B", 10); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	for i := 2; i <= len(FileSizeRow)+1; i++ {
		ServAgrRow := strconv.Itoa(i)
		t := "B" + ServAgrRow
		err = f.SetCellValue("MarketMatching", t, FileSizeRow[i-2])
	}
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	err = f.SetCellValue("MarketMatching", "C1", "ServAgrDuration")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	if err = f.SetColWidth("MarketMatching", "C", "C", 15); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	for i := 2; i <= len(ServAgrDurationRow)+1; i++ {
		ServAgrRow := strconv.Itoa(i)
		t := "C" + ServAgrRow
		err = f.SetCellValue("MarketMatching", t, ServAgrDurationRow[i-2])
	}
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	err = f.SetCellValue("MarketMatching", "D1", "StartedMCRoundNumber")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	if err = f.SetColWidth("MarketMatching", "D", "D", 15); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	for i := 2; i <= len(ServAgrDurationRow)+1; i++ {
		ServAgrRow := strconv.Itoa(i)
		t := "D" + ServAgrRow
		err = f.SetCellValue("MarketMatching", t, 0)
	}
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	if err = f.SetCellValue("MarketMatching", "F1", "Published"); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	if err = f.SetColWidth("MarketMatching", "F", "F", 10); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	for i := 2; i <= numberOfNodes+1; i++ {
		ServAgrRow := strconv.Itoa(i)
		t := "F" + ServAgrRow
		if err = f.SetCellValue("MarketMatching", t, 0); err != nil {
			log.LLvl1("Panic Raised:\n\n")
			panic(err)
		}
	}
	// ---------------------------------------------------------------------------
	// ------------------------- Transaction Queue sheet  ------------------
	// ---------------------------------------------------------------------------
	_ = f.NewSheet("FirstQueue")
	if err = f.SetColWidth("FirstQueue", "A", "AAA", 25); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	if err := f.SetSheetPrOptions("FirstQueue",
		excelize.FitToPage(true),
		excelize.TabColor("#8B0000"),
		excelize.AutoPageBreaks(true),
	); err != nil {
		fmt.Println(err)
	}
	err = f.SetRowHeight("FirstQueue", 1, 30)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	f.SetCellStyle("FirstQueue", "A1", "AAA1", style)
	// ----------------------- Transaction Queue Header ------------------------------
	err = f.SetCellValue("FirstQueue", "A1", "Name")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	err = f.SetCellValue("FirstQueue", "B1", "Size")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	if err = f.SetColWidth("FirstQueue", "B", "B", 10); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	err = f.SetCellValue("FirstQueue", "C1", "Time")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	err = f.SetCellValue("FirstQueue", "D1", "IssuedMCRoundNumber")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	if err = f.SetColWidth("FirstQueue", "D", "D", 15); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	err = f.SetCellValue("FirstQueue", "E1", "ServAgrId")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	// ---------------------------------------------------------------------------
	// ------------------------- Second Queue sheet  ------------------
	// ---------------------------------------------------------------------------
	_ = f.NewSheet("SecondQueue")
	if err = f.SetColWidth("SecondQueue", "A", "AAA", 25); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	if err := f.SetSheetPrOptions("SecondQueue",
		excelize.FitToPage(true),
		excelize.TabColor("#8B0999"),
		excelize.AutoPageBreaks(true),
	); err != nil {
		fmt.Println(err)
	}
	err = f.SetRowHeight("SecondQueue", 1, 30)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	f.SetCellStyle("SecondQueue", "A1", "AAA1", style)
	// ----------------------- SecondQueue Header ------------------------------
	err = f.SetCellValue("SecondQueue", "A1", "Size")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	if err = f.SetColWidth("SecondQueue", "A", "A", 10); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	err = f.SetCellValue("SecondQueue", "B1", "Time")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	err = f.SetCellValue("SecondQueue", "C1", "IssuedMCRoundNumber")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	if err = f.SetColWidth("SecondQueue", "C", "C", 15); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	// ---------------------------------------------------------------------------
	// ------------------------- power table sheet  -----------------------
	// ---------------------------------------------------------------------------
	_ = f.NewSheet("PowerTable")
	if err = f.SetColWidth("PowerTable", "A", "AAA", 30); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	if err := f.SetSheetPrOptions("PowerTable",
		excelize.FitToPage(true),
		excelize.TabColor("#B2FF66"),
		excelize.AutoPageBreaks(true),
	); err != nil {
		fmt.Println(err)
	}
	err = f.SetRowHeight("PowerTable", 1, 30)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	f.SetCellStyle("PowerTable", "A1", "AAA1", style)
	// --------------------- distribution of initial power -------------------
	intVar, _ = strconv.Atoi(InitialPowerDistributionVariance)
	VarianceInitialPower := intVar
	intVar, _ = strconv.Atoi(InitialPowerDistributionMean)
	MeanInitialPower := intVar
	InitialPowerRow := generateNormalValues(VarianceInitialPower, MeanInitialPower, numberOfNodes, SimulationSeedInt)
	/*var InitialPowerRowString []string
	for i:=0;i<len(InitialPowerRow);i++{
		InitialPowerRowString = append(InitialPowerRowString,InitialPowerRow[i])
	}*/
	// -----------------------    Filling Power Table's Headers   -----------
	err = f.SetCellValue("PowerTable", "A1", "Round#/NodeInfo")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	if err = f.SetColWidth("PowerTable", "A", "A", 15); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	// initial powers
	err = f.SetCellValue("PowerTable", "A2", "1")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	// -----------------------    Filling Power Table's first row ------------
	err = f.SetSheetRow("PowerTable", "B2", &InitialPowerRow)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	// --------------------------------------------------------------------
	// --------------------- Round Table Sheet ---------------------------
	// --------------------------------------------------------------------
	_ = f.NewSheet("RoundTable")
	if err = f.SetColWidth("RoundTable", "A", "AAA", 50); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	if err := f.SetSheetPrOptions("RoundTable",
		excelize.FitToPage(true),
		excelize.TabColor("#FF66FF"),
		excelize.AutoPageBreaks(true),
	); err != nil {
		fmt.Println(err)
	}
	err = f.SetRowHeight("RoundTable", 1, 30)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	f.SetCellStyle("RoundTable", "A1", "AAA1", style)
	// -----------------------    Filling Round Table's Headers ------------
	err = f.SetCellValue("RoundTable", "A1", "Round#")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	if err = f.SetColWidth("RoundTable", "A", "A", 10); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	err = f.SetCellValue("RoundTable", "B1", "Seed")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	err = f.SetCellValue("RoundTable", "C1", "BCSize")
	if err != nil {
		log.LLvl1(err)
	}
	if err = f.SetColWidth("RoundTable", "C", "C", 10); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	err = f.SetCellValue("RoundTable", "D1", "Round Leader")
	if err != nil {
		log.LLvl1(err)
	}
	if err = f.SetColWidth("RoundTable", "D", "D", 20); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	// ---- throughput measurement
	if err = f.SetColWidth("RoundTable", "E", "I", 10); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	err = f.SetCellValue("RoundTable", "E1", "#RegPay-TX")
	if err != nil {
		log.LLvl1(err)
	}
	err = f.SetCellValue("RoundTable", "F1", "#PoR-Tx")
	if err != nil {
		log.LLvl1(err)
	}
	err = f.SetCellValue("RoundTable", "G1", "#StorjPay-Tx")
	if err != nil {
		log.LLvl1(err)
	}
	err = f.SetCellValue("RoundTable", "H1", "#CntProp-Tx")
	if err != nil {
		log.LLvl1(err)
	}
	err = f.SetCellValue("RoundTable", "I1", "#CntCmt-Tx")
	if err != nil {
		log.LLvl1(err)
	}
	err = f.SetCellValue("RoundTable", "P1", "#Sync-Tx")
	if err != nil {
		log.LLvl1(err)
	}

	if err = f.SetColWidth("RoundTable", "J", "J", 25); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	if err = f.SetColWidth("RoundTable", "K", "O", 15); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	err = f.SetCellValue("RoundTable", "J1", "StartTime")
	if err != nil {
		log.LLvl1(err)
	}
	err = f.SetCellValue("RoundTable", "K1", "TotalNumTxs")
	if err != nil {
		log.LLvl1(err)
	}
	err = f.SetCellValue("RoundTable", "L1", "AveWait-OtherTxs")
	if err != nil {
		log.LLvl1(err)
	}
	err = f.SetCellValue("RoundTable", "M1", "AveWait-RegPay")
	if err != nil {
		log.LLvl1(err)
	}
	err = f.SetCellValue("RoundTable", "N1", "RegPaySpaceFull")
	if err != nil {
		log.LLvl1(err)
	}
	err = f.SetCellValue("RoundTable", "O1", "BlockSpaceFull")
	if err != nil {
		log.LLvl1(err)
	}

	// -----------------------    Filling Round Table's first row  -------------------
	// err = f.SetCellValue("RoundTable", "A2", 0)
	// if err != nil {
	// 	log.LLvl1("Panic Raised:\n\n")
	// 	panic(err)
	// }
	t := rand.Intn(SimulationSeedInt)
	err = f.SetCellValue("RoundTable", "B2", t)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	err = f.SetCellValue("RoundTable", "C2", 0)
	if err != nil {
		log.LLvl1(err)
	}
	// -----------------------    Filling Round Table's second row: next round's seed
	err = f.SetCellValue("RoundTable", "A3", 1)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
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
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	// --------------------------------------------------------------------
	// --------------------- Overall Evaluation Sheet ------------------
	// --------------------------------------------------------------------
	_ = f.NewSheet("OverallEvaluation")
	if err = f.SetColWidth("RoundTable", "A", "AAA", 10); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	if err := f.SetSheetPrOptions("OverallEvaluation",
		excelize.FitToPage(true),
		excelize.TabColor("#FE00FF"),
		excelize.AutoPageBreaks(true),
	); err != nil {
		fmt.Println(err)
	}
	err = f.SetRowHeight("OverallEvaluation", 1, 30)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	f.SetCellStyle("OverallEvaluation", "A1", "AAA1", style)
	// -----------------------    Filling Round Table's Headers ------------
	err = f.SetCellValue("OverallEvaluation", "A1", "Round#")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	err = f.SetCellValue("OverallEvaluation", "B1", "BCSize")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	err = f.SetCellValue("OverallEvaluation", "C1", "Overall#RegPay-TX")
	if err != nil {
		log.LLvl1(err)
	}
	err = f.SetCellValue("OverallEvaluation", "D1", "Overall#PoR-TX")
	if err != nil {
		log.LLvl1(err)
	}
	err = f.SetCellValue("OverallEvaluation", "E1", "Overall#StorjPay-TX")
	if err != nil {
		log.LLvl1(err)
	}
	err = f.SetCellValue("OverallEvaluation", "F1", "Overall#CntProp-TX")
	if err != nil {
		log.LLvl1(err)
	}
	err = f.SetCellValue("OverallEvaluation", "G1", "Overall#CntCmt-TX")
	if err != nil {
		log.LLvl1(err)
	}
	err = f.SetCellValue("OverallEvaluation", "H1", "OveralAveWait-OtherTxs")
	if err != nil {
		log.LLvl1(err)
	}
	err = f.SetCellValue("OverallEvaluation", "I1", "OveralAveWait-RegPay")
	if err != nil {
		log.LLvl1(err)
	}
	err = f.SetCellValue("OverallEvaluation", "J1", "OverallBlockSpaceFull")
	if err != nil {
		log.LLvl1(err)
	}
	// --------------------------------------------------------------------
	// if err := f.SaveAs("mainchainbc.xlsx"); err != nil {
	// 	log.LLvl1("Panic Raised:\n\n")
	// 	panic(err)
	// }
	if err := f.SaveAs("mainchainbc.xlsx"); err != nil {
		pwd, _ := os.Getwd()
		log.Fatal("Panic Raised:\n\n", err, "we are in: ", pwd)
		panic(err)
	} else {
		pwd, _ := os.Getwd()
		log.LLvl1("mainchainbc.xlsx created in: ", pwd)
	}

	log.LLvl1("Config params used in initial initialization of main chain:",
		"\n File Size Distribution Mean: ", FileSizeDistributionMean,
		"\n File Size Distribution Variance: ", FileSizeDistributionVariance,
		"\n ServAgr Duration Distribution Mean: ", ServAgrDurationDistributionMean,
		"\n ServAgr Duration Distribution Variance: ", ServAgrDurationDistributionVariance,
		"\n Initial Power Distribution Mean: ", InitialPowerDistributionMean,
		"\n Initial Power Distribution Variance: ", InitialPowerDistributionVariance)
}

/*	// in case of initializing a column
for i := 1;i<=len(FileSizeColumn);i++ {
		axis = "B" + strconv.Itoa(i+1)
		f.SetCellValue("MarketMatching", axis, FileSizeColumn[i-1])
		axis = "C" + strconv.Itoa(i+1)
		f.SetCellValue("MarketMatching", axis, ServAgrDurationColumn[i-1])
	}
	now := time.Now()
	f2.SetCellValue("Sheet1", "A4", now.Format(time.ANSIC))
*/

// InitializeSideChainBC function is called in simulation level (build.go)
func InitializeSideChainBC() {
	f := excelize.NewFile() //ToDoRaha: we don't need to create a new file here!!! the file is created bt prescript!! Has it?!
	var err error
	style, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			WrapText: true,
		},
	})
	// f, err := excelize.OpenFile("sidechainbc.xlsx")
	// if err != nil {
	// 	log.LLvl1("Raha: ", err)
	// 	panic(err)
	// } else {
	// 	log.LLvl1("opening side chain bc")
	// }
	// ---------------------------------------------------------------------------
	// ------------------------- Transaction Queue sheet  ------------------
	// ---------------------------------------------------------------------------
	_ = f.NewSheet("FirstQueue")
	if err = f.SetColWidth("FirstQueue", "A", "AAA", 25); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	if err := f.SetSheetPrOptions("FirstQueue",
		excelize.FitToPage(true),
		excelize.TabColor("#8B0000"),
		excelize.AutoPageBreaks(true),
	); err != nil {
		fmt.Println(err)
	}
	err = f.SetRowHeight("FirstQueue", 1, 30)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	f.SetCellStyle("FirstQueue", "A1", "AAA1", style)
	// ----------------------- Transaction Queue Header ------------------------------
	err = f.SetCellValue("FirstQueue", "A1", "Name")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	err = f.SetCellValue("FirstQueue", "B1", "Size")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	if err = f.SetColWidth("FirstQueue", "B", "B", 10); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	err = f.SetCellValue("FirstQueue", "C1", "Time")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	err = f.SetCellValue("FirstQueue", "D1", "IssuedSCRoundNumber")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	if err = f.SetColWidth("FirstQueue", "D", "D", 15); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	err = f.SetCellValue("FirstQueue", "E1", "ServAgrId")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	err = f.SetCellValue("FirstQueue", "F1", "MC Round#")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}

	// --------------------------------------------------------------------
	// --------------------- Round Table Sheet ---------------------------
	// --------------------------------------------------------------------
	_ = f.NewSheet("RoundTable")
	if err = f.SetColWidth("RoundTable", "A", "AAA", 15); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	if err := f.SetSheetPrOptions("RoundTable",
		excelize.FitToPage(true),
		excelize.TabColor("#FF66FF"),
		excelize.AutoPageBreaks(true),
	); err != nil {
		fmt.Println(err)
	}
	err = f.SetRowHeight("RoundTable", 1, 30)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	f.SetCellStyle("RoundTable", "A1", "AAA1", style)
	// -----------------------    Filling Round Table's Headers ------------
	err = f.SetCellValue("RoundTable", "A1", "Round#")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	if err = f.SetColWidth("RoundTable", "A", "A", 10); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	err = f.SetCellValue("RoundTable", "B1", "BCSize")
	if err != nil {
		log.LLvl1(err)
	}
	if err = f.SetColWidth("RoundTable", "B", "B", 10); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	err = f.SetCellValue("RoundTable", "C1", "Round Leader")
	if err != nil {
		log.LLvl1(err)
	}
	if err = f.SetColWidth("RoundTable", "C", "C", 40); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	//---- throughput measurement
	if err = f.SetColWidth("RoundTable", "D", "I", 10); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	err = f.SetCellValue("RoundTable", "D1", "#PoR-Tx")
	if err != nil {
		log.LLvl1(err)
	}
	if err = f.SetColWidth("RoundTable", "D", "D", 15); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	if err = f.SetColWidth("RoundTable", "I", "I", 20); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	if err = f.SetColWidth("RoundTable", "E", "E", 60); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	err = f.SetCellValue("RoundTable", "E1", "StartTime")
	if err != nil {
		log.LLvl1(err)
	}

	err = f.SetCellValue("RoundTable", "G1", "TotalNumTxs")
	if err != nil {
		log.LLvl1(err)
	}
	if err = f.SetColWidth("RoundTable", "G", "G", 20); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	err = f.SetCellValue("RoundTable", "F1", "AveWait")
	if err != nil {
		log.LLvl1(err)
	}
	err = f.SetCellValue("RoundTable", "H1", "BlockSpaceFull")
	if err != nil {
		log.LLvl1(err)
	}
	err = f.SetCellValue("RoundTable", "I1", "TimeTaken")
	if err != nil {
		log.LLvl1(err)
	}
	err = f.SetCellValue("RoundTable", "J1", "Mc Round#")
	if err != nil {
		log.LLvl1(err)
	}
	// --------------------------------------------------------------------
	// --------------------- Overall Evaluation Sheet ------------------
	// --------------------------------------------------------------------
	_ = f.NewSheet("OverallEvaluation")
	if err = f.SetColWidth("OverallEvaluation", "A", "AAA", 10); err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	if err := f.SetSheetPrOptions("OverallEvaluation",
		excelize.FitToPage(true),
		excelize.TabColor("#FE00FF"),
		excelize.AutoPageBreaks(true),
	); err != nil {
		fmt.Println(err)
	}
	err = f.SetRowHeight("OverallEvaluation", 1, 30)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	f.SetCellStyle("OverallEvaluation", "A1", "AAA1", style)
	// -----------------------    Filling Round Table's Headers ------------
	err = f.SetCellValue("OverallEvaluation", "A1", "Round#")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	err = f.SetCellValue("OverallEvaluation", "B1", "BCSize")
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		panic(err)
	}
	err = f.SetCellValue("OverallEvaluation", "C1", "Overall#PoR-TX")
	if err != nil {
		log.LLvl1(err)
	}
	err = f.SetCellValue("OverallEvaluation", "D1", "OveralAveWait")
	if err != nil {
		log.LLvl1(err)
	}
	err = f.SetCellValue("OverallEvaluation", "E1", "OverallBlockSpaceFull")
	if err != nil {
		log.LLvl1(err)
	}
	// --------------------------------------------------------------------
	if err := f.SaveAs("sidechainbc.xlsx"); err != nil {
		log.Fatal("Panic Raised:\n\n")
		panic(err)
	} else {
		pwd, _ := os.Getwd()
		log.LLvl1("sidechainbc.xlsx created in: ", pwd)
	}
}
