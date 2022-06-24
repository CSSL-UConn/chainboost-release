// package main

// import (
// 	"fmt"
// 	"net"
// 	"github.com/360EntSecGroup-Skylar/excelize"
// 	"time"
// )

// func main() {

// 	fmt.Print("------------------------------------------------------------- \n ")
//         fmt.Print("------------------------------ MAIN CHAIN -------------------- \n ")
//         fmt.Print("------------------------------------------------------------- \n ")

// 	xlsx, err := excelize.OpenFile("/root/remote/mainchainbc.xlsx")
// 	if err != nil {
// 		fmt.Println(err)
// 		return
// 	}
// 	fmt.Print("----------------------  MarketMatching ------------------- \n ")
// 	rows := xlsx.GetRows("MarketMatching")
// 	for _, row := range rows {
// 		for _, colCell := range row {
// 			fmt.Print(colCell, "\t")
// 		}
// 		fmt.Println()
// 	}

// 	fmt.Print("----------------------  FirstQueue ------------------- \n ")
// 	rows = xlsx.GetRows("FirstQueue")
// 	for _, row := range rows {
// 		for _, colCell := range row {
// 			fmt.Print(colCell, "\t")
// 		}
// 		fmt.Println()
// 	}

// 	fmt.Print("----------------------  SecondQueue ------------------- \n ")
// 	rows = xlsx.GetRows("SecondQueue")
// 	for _, row := range rows {
// 		for _, colCell := range row {
// 			fmt.Print(colCell, "\t")
// 		}
// 		fmt.Println()
// 	}

// 	fmt.Print("----------------------  PowerTable ------------------- \n ")
// 	rows = xlsx.GetRows("PowerTable")
// 	for _, row := range rows {
// 		for _, colCell := range row {
// 			fmt.Print(colCell, "\t")
// 		}
// 		fmt.Println()
// 	}

// 	fmt.Print("----------------------  RoundTable ------------------- \n ")
// 	rows = xlsx.GetRows("RoundTable")
// 	for _, row := range rows {
// 		for _, colCell := range row {
// 			fmt.Print(colCell, "\t")
// 		}
// 		fmt.Println()
// 	}

// 	fmt.Print("----------------------  OverallEvaluation ------------------- \n ")
// 	rows = xlsx.GetRows("OverallEvaluation")
// 	for _, row := range rows {
// 		for _, colCell := range row {
// 			fmt.Print(colCell, "\t")
// 		}
// 		fmt.Println()
// 	}

// 	fmt.Print("------------------------------------------------------------- \n ")
// 	fmt.Print("------------------------------ SIDE CHAIN -------------------- \n ")
// 	fmt.Print("------------------------------------------------------------- \n ")

// 	xlsx, err = excelize.OpenFile("/root/remote/sidechainbc.xlsx")
//         if err != nil {
//                 fmt.Println(err)
//                 return
//         }
//         fmt.Print("----------------------  MarketMatching ------------------- \n ")
//         rows = xlsx.GetRows("MarketMatching")
//         for _, row := range rows {
//                 for _, colCell := range row {
//                         fmt.Print(colCell, "\t")
//                 }
//                 fmt.Println()
//         }

//         fmt.Print("----------------------  FirstQueue ------------------- \n ")
//         rows = xlsx.GetRows("FirstQueue")
//         for _, row := range rows {
//                 for _, colCell := range row {
//                         fmt.Print(colCell, "\t")
//                 }
//                 fmt.Println()
//         }

//         fmt.Print("----------------------  RoundTable ------------------- \n ")
//         rows = xlsx.GetRows("RoundTable")
//         for _, row := range rows {
//                 for _, colCell := range row {
//                         fmt.Print(colCell, "\t")
//                 }
//                 fmt.Println()
//         }

//         fmt.Print("----------------------  OverallEvaluation ------------------- \n ")
//         rows = xlsx.GetRows("OverallEvaluation")
//         for _, row := range rows {
//                 for _, colCell := range row {
//                         fmt.Print(colCell, "\t")
//                 }
//                 fmt.Println()
//         }

// 	//test tcp connection
// 	var dialTimeout = 1 * time.Minute
// 	//var c net.Conn
// 	netAddr := "192.168.3.212:2000"
// 	_, err = net.DialTimeout("tcp", netAddr, dialTimeout)
// 	if err != nil {
// 		fmt.Println("err:", err)
// 	} else {
// 		fmt.Println("sucees")
// 	}
// }


