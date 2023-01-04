package blockchain

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type MarketMatchingRows struct {
	MinerServer          string
	FileSize             int
	ServerAgrDuration    int
	StartedMcRoundNumber int
	Published            bool
	TxIssued             bool
}

type MainChainSecondQueueEntry struct {
	RowId       int
	Time        time.Time
	Size        uint32
	RoundIssued int
}

type MainChainFirstQueueEntry struct {
	RowId       int
	Name        string
	Size        uint32
	Time        time.Time
	RoundIssued int
	ServAgrId   int
}

type SideChainFirstQueueEntry struct {
	RowId               int
	Name                string
	Size                int
	Time                time.Time
	IssuedScRoundNumber int
	ServAgrId           int
	MCRoundNbr          int
}

type SideChainRoundInfo struct {
	RoundNumber    int
	BCSize         int
	RoundLeader    string
	PoRTx          int
	StartTime      time.Time
	AveWait        float64
	TotalNumTx     int
	BlockSpaceFull int
	TimeTaken      int
	McRound        int
}

func GetAbsPath(file string) string {
	pth, _ := os.Getwd()
	return fmt.Sprintf("%s/%s", pth, file)
}

func GetMainChainAbsPath() string {
	return GetAbsPath("mainchain.db")
}

func GetSideChainAbsPath() string {
	return GetAbsPath("sidechain.db")
}

var mainchainpath string = GetMainChainAbsPath()
var sidechainpath string = GetSideChainAbsPath()

func deleteDbIfExists(filename string) {
	if _, err := os.Stat(filename); err == nil {
		os.Remove(filename)
	}
}

func InitalizeMainChainDbTables() error {
	deleteDbIfExists(mainchainpath)
	mainchainDb, err := sql.Open("sqlite", mainchainpath)
	if err != nil {
		return err
	}
	defer mainchainDb.Close()

	_, err = mainchainDb.Query(`
        CREATE TABLE MarketMatching (
            "ServerInfo" text,
            "FileSize" integer,
            "ServAgrDuration" integer,
            "StartedMcRoundNumber" integer DEFAULT 0,
            "ServAgrId" integer,
            "Published" boolean DEFAULT 1,
            "TXIssued" boolean DEFAULT 1
        );
    `)

	if err != nil {
		return err
	}

	_, err = mainchainDb.Query(`
        CREATE TABLE FirstQueue (
            "Name" text,
            "Size" integer,
            "Time" text,
            "IssuedMcRoundNumber" integer,
            "ServAgrId" integer
        );

    `)

	if err != nil {
		return err
	}

	_, err = mainchainDb.Query(`
        CREATE TABLE SecondQueue (
            "Time" text,
            "Size" integer,
            "IssuedMCRoundNumber" integer
        );
    `)

	if err != nil {
		return err
	}

	_, err = mainchainDb.Query(`
        CREATE TABLE RoundTable (
            "RoundNumber" integer key,
            "Seed" text,
            "BCSize" integer,
            "RoundLeader" text,
            "RegPayTx" integer,
            "PoRTx" integer,
            "StorjPayTx" integer,
            "CntPropTx" integer,
            "CntCmtTx" integer,
            "StartTime" text,
            "TotalNumTx" integer,
            "AveWaitOtherTxs" REAL,
            "AveWaitRegPay" REAL,
            "RegSpaceFull" integer,
            "BlockSpaceFull" integer,
            "SyncTx" integer
        );
    `)

	if err != nil {
		return err
	}

	_, err = mainchainDb.Query(`
        CREATE TABLE OverallEvaluation (
            "RoundNbr" integer primary key,
            "BCSize" integer,
            "OverallRegPayTxNbr" integer,
            "OverallPoRTxNbr" integer,
            "OverallStorjPayTxNbr" integer,
            "OverallCntPropTxNbr" integer,
            "OverallCntCmtTxNbr" integer,
            "OverallAveWaitOtherTx" integer,
            "OverallAveWaitRegPay" integer,
            "OverallBlockSpaceFull" integer
        );
    `)

	if err != nil {
		return err
	}

	sqlStmt := `CREATE TABLE PowerTable ("hosts" string primary key, "Power" integer)`

	_, err = mainchainDb.Query(sqlStmt)
	if err != nil {
		return err
	}

	return nil
}

func InitalizeSideChainDbTables() error {
	deleteDbIfExists(sidechainpath)
	sidechainDb, err := sql.Open("sqlite", sidechainpath)
	if err != nil {
		return err
	}
	defer sidechainDb.Close()

	_, err = sidechainDb.Query(`
        CREATE TABLE FirstQueue (
            "Name" text,
            "Size" integer,
            "Time" text,
            "IssuedScRoundNumber" integer,
            "ServAgrId" integer,
            "MCRoundNbr" integer
        );
    `)

	if err != nil {
		return err
	}

	_, err = sidechainDb.Query(`
        CREATE TABLE RoundTable (
            "RoundNumber" integer,
            "BCSize" integer,
            "RoundLeader" text,
            "PoRTx" integer,
            "StartTime" text,
            "AveWait" integer,
            "TotalNumTx" integer,
            "BlockSpaceFull" integer,
            "TimeTaken" integer,
            "McRound" integer
        );
    `)

	if err != nil {
		return err
	}

	_, err = sidechainDb.Query(`
        CREATE TABLE OverallEvaluation (
            "RoundNbr" integer,
            "BCSize" integer,
            "OverallPoRTxNbr" integer,
            "OverallAveWait" integer,
            "OverallBlockSpaceFull" integer
        );
    `)

	if err != nil {
		return err
	}
	return nil
}

// Insert Functions
func InsertIntoMainChainMarketMatchingTable(serverInfo string,
	fileSize int,
	serverAgrDuration int,
	startedMcRoundNumber int,
	serverAgrId int,
	published bool,
	TXIssued bool) error {
	mainchainDb, err := sql.Open("sqlite", mainchainpath)
	if err != nil {
		return err
	}
	defer mainchainDb.Close()

	_, err = mainchainDb.Exec(`INSERT INTO MarketMatching(ServerInfo, FileSize, ServAgrDuration,
                                StartedMcRoundNumber, ServAgrId, Published, TXIssued)
                                 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		serverInfo, fileSize, serverAgrDuration, startedMcRoundNumber,
		serverAgrId, published, TXIssued)
	return err
}

func InsertIntoMainChainFirstQueue(name string, size uint32, timestamp time.Time, issuedMcRoundNumber int, serverAgrId int) error {
	mainchainDb, err := sql.Open("sqlite", mainchainpath)
	if err != nil {
		return err
	}
	defer mainchainDb.Close()

	_, err = mainchainDb.Exec(`INSERT INTO FirstQueue(Name, Size, Time, IssuedMcRoundNumber, ServAgrId)
                                 VALUES (?, ?, ?, ?, ?)`, name, size, timestamp.Format(time.RFC3339), issuedMcRoundNumber, serverAgrId)
	return err
}

func BulkInsertIntoMainChainFirstQueue(rows []MainChainFirstQueueEntry) error {
	if len(rows) == 0 {
		return nil
	}
	mainchainDb, err := sql.Open("sqlite", mainchainpath)
	if err != nil {
		return err
	}
	valueStrings := make([]string, 0, len(rows))
	valueArgs := make([]interface{}, 0, len(rows)*5)
	for _, row := range rows {
		valueStrings = append(valueStrings, "(?, ?, ?, ?, ?)")
		valueArgs = append(valueArgs, row.Name)
		valueArgs = append(valueArgs, row.Size)
		valueArgs = append(valueArgs, row.Time)
		valueArgs = append(valueArgs, row.RoundIssued)
		valueArgs = append(valueArgs, row.ServAgrId)
	}
	stmt := fmt.Sprintf(`INSERT INTO FirstQueue(Name, Size, Time, IssuedMcRoundNumber, ServAgrId)
                                 VALUES %s`, strings.Join(valueStrings, ","))
	_, err = mainchainDb.Exec(stmt, valueArgs...)
	return err
}

func InsertIntoMainChainSecondQueue(size uint32, timestamp time.Time, issuedMcRoundNumber int) error {
	mainchainDb, err := sql.Open("sqlite", mainchainpath)
	if err != nil {
		return err
	}
	defer mainchainDb.Close()

	_, err = mainchainDb.Exec(`INSERT INTO SecondQueue(Time, Size, IssuedMcRoundNumber)
                                 VALUES (?, ?, ?)`, timestamp.Format(time.RFC3339), size, issuedMcRoundNumber)
	return err
}

func BulkInsertIntoMainChainSecondQueue(rows []MainChainSecondQueueEntry) error {
	if len(rows) == 0 {
		return nil
	}
	mainchainDb, err := sql.Open("sqlite", mainchainpath)
	if err != nil {
		return err
	}
	valueStrings := make([]string, 0, len(rows))
	valueArgs := make([]interface{}, 0, len(rows)*3)
	for _, row := range rows {
		valueStrings = append(valueStrings, "(?, ?, ?)")
		valueArgs = append(valueArgs, row.Time)
		valueArgs = append(valueArgs, row.Size)
		valueArgs = append(valueArgs, row.RoundIssued)
	}
	stmt := fmt.Sprintf(`INSERT INTO SecondQueue(Time, Size, IssuedMcRoundNumber)
                                 VALUES %s`, strings.Join(valueStrings, ","))
	_, err = mainchainDb.Exec(stmt, valueArgs...)
	return err
}

func InsertIntoMainChainRoundTable(roundNbr int,
	Seed string,
	BCSize int,
	roundLeader string,
	regPayTx int,
	PoRTx int,
	StorjPayTx int,
	CntPropTx int,
	CntCmtTx int,
	startTime time.Time,
	TotalNumTx int,
	AveWaitOtherTxs float64,
	AveWaitRegPay float64,
	regSpaceFull bool,
	blockSpaceFull bool,
	SyncTx bool) error {
	mainchainDb, err := sql.Open("sqlite", mainchainpath)
	if err != nil {
		return err
	}
	defer mainchainDb.Close()
	_, err = mainchainDb.Exec(
		`INSERT INTO RoundTable(
                                RoundNumber,
                                Seed,
                                BCSize,
                                RoundLeader,
                                RegPayTx,
                                PoRTx,
                                StorjPayTx,
                                CntPropTx,
                                CntCmtTx,
                                StartTime,
                                TotalNumTx,
                                AveWaitOtherTxs,
                                AveWaitRegPay,
                                RegSpaceFull,
                                BlockSpaceFull,
                                SyncTx
                            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		roundNbr, Seed, BCSize, roundLeader,
		regPayTx, PoRTx, StorjPayTx, CntPropTx,
		CntCmtTx, startTime.Format(time.RFC3339), TotalNumTx, AveWaitOtherTxs,
		AveWaitRegPay, regSpaceFull, blockSpaceFull, SyncTx,
	)
	return err
}

func InsertIntoMainChainOverallEvaluationTable(roundNbr int,
	BCSize float64,
	OverallRegPayTxNbr float64,
	OverallPorTxNbr float64,
	OverallStorjPayTxNbr float64,
	OverallCntPropTxNbr float64,
	OverallCntCmtTxNbr float64,
	OverallAveWaitOtherTx float64,
	OverallAveWaitRegPay float64,
	OverallBlockSpaceFull int) error {
	mainchainDb, err := sql.Open("sqlite", mainchainpath)
	if err != nil {
		return err
	}
	defer mainchainDb.Close()
	_, err = mainchainDb.Exec(
		`INSERT INTO OverallEvaluation(
                                RoundNbr,
                                BCSize,
                                OverallRegPayTxNbr,
                                OverallPoRTxNbr,
                                OverallStorjPayTxNbr,
                                OverallCntPropTxNbr,
                                OverallCntCmtTxNbr,
                                OverallAveWaitOtherTx,
                                OverallAveWaitRegPay,
                                OverallBlockSpaceFull
                             ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		roundNbr, BCSize, OverallRegPayTxNbr, OverallPorTxNbr,
		OverallStorjPayTxNbr, OverallCntPropTxNbr, OverallCntCmtTxNbr,
		OverallAveWaitOtherTx, OverallAveWaitRegPay, OverallBlockSpaceFull,
	)
	return err

}

func InitialInsertValuesIntoMarketMatchingTable(Filesize int,
	ServAgrDuration int,
	StartedMcRoundNumber int,
	ServAgrId int,
	Published bool,
	TxIssued bool) error {
	var err error
	mainchainDb, err := sql.Open("sqlite", mainchainpath)
	if err != nil {
		return err
	}
	defer mainchainDb.Close()
	sqlStmt := "INSERT INTO MarketMatching (FileSize, ServAgrDuration, StartedMcRoundNumber, ServAgrId, Published, TxIssued) VALUES (?, ?, ?, ?, ?, ?)"
	_, err = mainchainDb.Exec(sqlStmt, Filesize, ServAgrDuration, StartedMcRoundNumber, ServAgrId, Published, TxIssued)
	return err
}

func AddMoreFieldsIntoTableInMainChain(table string, column string, values ...interface{}) error {
	var err error
	mainchainDb, err := sql.Open("sqlite", mainchainpath)
	if err != nil {
		return err
	}
	defer mainchainDb.Close()
	for i, value := range values {
		sqlStmt := fmt.Sprintf("UPDATE %s SET %s = ? where rowid = ?", table, column)
		_, err = mainchainDb.Exec(sqlStmt, value, i+1) // SQL starts from 1
		if err != nil {
			return err
		}
	}
	return nil
}

func InitialInsertValuesIntoMainChainPowerTable(values ...interface{}) error {
	var err error
	var placeholders []string
	mainchainDb, err := sql.Open("sqlite", mainchainpath)
	if err != nil {
		return err
	}
	defer mainchainDb.Close()
	for _, _ = range values {
		placeholders = append(placeholders, "(?)")
	}
	sqlStmt := fmt.Sprintf("INSERT INTO PowerTable (hosts) VALUES %s", strings.Join(placeholders, ","))
	_, err = mainchainDb.Exec(sqlStmt, values...)
	return err
}

func InsertIntoSideChainFirstQueue(name string, size uint32, timestamp time.Time, issuedScRoundNumber int, serverAgrId int, MCRoundNbr int) error {
	var err error
	sidechainDb, err := sql.Open("sqlite", sidechainpath)
	if err != nil {
		return err
	}
	defer sidechainDb.Close()
	_, err = sidechainDb.Exec(`INSERT INTO FirstQueue(Name, Size, Time, IssuedScRoundNumber, ServAgrId, MCRoundNbr)
                                 VALUES (?, ?, ?, ?, ?, ?)`, name, size, timestamp.Format(time.RFC3339), issuedScRoundNumber, serverAgrId, MCRoundNbr)
	return err

}

func BulkInsertIntoSideChainFirstQueue(rows []SideChainFirstQueueEntry) error {
	sidechainDb, err := sql.Open("sqlite", sidechainpath)
	if err != nil {
		return err
	}
	valueStrings := make([]string, 0, len(rows))
	valueArgs := make([]interface{}, 0, len(rows)*6)
	for _, row := range rows {
		valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?)")
		valueArgs = append(valueArgs, row.Name)
		valueArgs = append(valueArgs, row.Size)
		valueArgs = append(valueArgs, row.Time)
		valueArgs = append(valueArgs, row.IssuedScRoundNumber)
		valueArgs = append(valueArgs, row.ServAgrId)
		valueArgs = append(valueArgs, row.MCRoundNbr)
	}
	fmt.Printf("len args: %d, len valueArgs: %d", len(rows), len(valueArgs))
	stmt := fmt.Sprintf(`INSERT INTO FirstQueue(Name, Size, Time, IssuedScRoundNumber, ServAgrId, MCRoundNbr)
                                 VALUES %s`, strings.Join(valueStrings, ","))
	_, err = sidechainDb.Exec(stmt, valueArgs...)
	return err
}

func InsertIntoSideChainRoundTable(roundNbr int,
	BCSize int,
	roundLeader string,
	PorTx int,
	startTime time.Time,
	aveWait int,
	TotalNumTx int,
	BlockSpaceFull int,
	TimeTaken int,
	McRound int) error {
	var err error
	sidechainDb, err := sql.Open("sqlite", sidechainpath)
	if err != nil {
		return err
	}
	defer sidechainDb.Close()
	_, err = sidechainDb.Exec(`INSERT INTO RoundTable(
                                        RoundNumber,
                                        BCSize,
                                        RoundLeader,
                                        PoRTx,
                                        StartTime,
                                        AveWait,
                                        TotalNumTx,
                                        BlockSpaceFull,
                                        TimeTaken,
                                        McRound)
                                 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		roundNbr, BCSize, roundLeader, PorTx, startTime.Format(time.RFC3339), aveWait,
		TotalNumTx, BlockSpaceFull, TimeTaken, McRound)

	return err

}

func InsertIntoSideChainOverallEvaluation(roundNbr int,
	BCSize float64,
	OverallPorTxNbr float64,
	OverallAveWait float64,
	OverallBlockSpaceFull int) error {
	var err error
	sidechainDb, err := sql.Open("sqlite", sidechainpath)
	if err != nil {
		return err
	}
	defer sidechainDb.Close()

	stmt := `INSERT INTO OverallEvaluation(
                                        RoundNbr,
                                        BCSize,
                                        OverallPorTxNbr,
                                        OverallAveWait,
                                        OverallBlockSpaceFull)
                                 VALUES (?, ?, ?, ?, ?)`

	_, err = sidechainDb.Exec(stmt, roundNbr, BCSize, OverallPorTxNbr, OverallAveWait, OverallBlockSpaceFull)
	return err
}

// ---------------------------------- Helper Functions

/// Warning: These functions are vulnerable to SQL Injection, and should be used with Care

func GetStatsMainChainImpl(stmt string) (float64, error) {
	var err error
	mainchainDb, err := sql.Open("sqlite", mainchainpath)
	if err != nil {
		return 0, err
	}
	defer mainchainDb.Close()
	row := mainchainDb.QueryRow(stmt)
	var stat float64
	if err := row.Scan(&stat); err != nil {
		return 0, err
	}
	return stat, nil

}

func GetStatsSideChainImpl(stmt string) (float64, error) {
	var err error
	sidechainDb, err := sql.Open("sqlite", sidechainpath)
	if err != nil {
		return 0, err
	}
	defer sidechainDb.Close()
	row := sidechainDb.QueryRow(stmt)

	var stat float64
	if err := row.Scan(&stat); err != nil {
		return 0, err
	}
	return stat, nil
}

func GetAvgMainChain(table string, column string) (float64, error) {
	stmt := fmt.Sprintf("SELECT AVG(%s) FROM %s", column, table)
	return GetStatsMainChainImpl(stmt)
}

func GetMaxMainChain(table string, column string) (float64, error) {
	stmt := fmt.Sprintf("SELECT MAX(%s) FROM %s", column, table)
	return GetStatsMainChainImpl(stmt)
}

func GetAvgMainChainCond(table string, column string, condition string) (float64, error) {
	stmt := fmt.Sprintf("SELECT AVG(%s) FROM %s WHERE %s", column, table, condition)
	return GetStatsMainChainImpl(stmt)
}

func GetSumMainChain(table string, column string) (float64, error) {
	stmt := fmt.Sprintf("SELECT SUM(%s) FROM %s", column, table)
	return GetStatsMainChainImpl(stmt)
}

func GetSumMainChainCond(table string, column string, condition string) (float64, error) {
	stmt := fmt.Sprintf("SELECT SUM(%s) FROM %s WHERE %s", column, table, condition)
	return GetStatsMainChainImpl(stmt)
}

func GetAvgSideChain(table string, column string) (float64, error) {
	stmt := fmt.Sprintf("SELECT AVG(%s) FROM %s", column, table)
	return GetStatsSideChainImpl(stmt)
}

func GetAvgSideChainCond(table string, column string, condition string) (float64, error) {
	stmt := fmt.Sprintf("SELECT AVG(%s) FROM %s WHERE %s", column, table, condition)
	return GetStatsSideChainImpl(stmt)
}

func GetSumSideChain(table string, column string) (float64, error) {
	stmt := fmt.Sprintf("SELECT SUM(%s) FROM %s", column, table)
	return GetStatsSideChainImpl(stmt)
}

func GetSumSideChainCond(table string, column string, condition string) (float64, error) {
	stmt := fmt.Sprintf("SELECT SUM(%s) FROM %s WHERE %s", column, table, condition)
	return GetStatsSideChainImpl(stmt)
}

func MainChainGetLastRoundSeed() (string, error) {
	stmt := "SELECT Seed from RoundTable WHERE RoundNumber =(SELECT max(RoundNumber) FROM RoundTable)"
	var err error
	mainchainDb, err := sql.Open("sqlite", mainchainpath)
	if err != nil {
		return "", err
	}
	defer mainchainDb.Close()
	row := mainchainDb.QueryRow(stmt)

	var stat string
	if err := row.Scan(&stat); err != nil {
		return "", err
	}
	return stat, nil
}

func MainChainGetPowerTable() (map[string]int, error) {
	minerspowers := make(map[string]int)
	stmt := "SELECT hosts, Power From PowerTable"
	mainchainDb, err := sql.Open("sqlite", mainchainpath)
	if err != nil {
		return nil, err
	}
	defer mainchainDb.Close()
	rows, err := mainchainDb.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var host string
		var power int
		if err := rows.Scan(&host, &power); err != nil {
			return nil, err
		}
		minerspowers[host] = power
	}
	return minerspowers, nil
}

func MainChainGetMarketMatchingRows() ([]MarketMatchingRows, error) {
	stmt := `SELECT ServerInfo,
                    FileSize,
                    ServAgrDuration,
                    StartedMcRoundNumber,
                    Published,
                    TxIssued
            FROM MarketMatching`

	var output []MarketMatchingRows
	mainchainDb, err := sql.Open("sqlite", mainchainpath)
	if err != nil {
		return nil, err
	}
	defer mainchainDb.Close()
	rows, err := mainchainDb.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item MarketMatchingRows
		if err := rows.Scan(&item.MinerServer,
			&item.FileSize,
			&item.ServerAgrDuration,
			&item.StartedMcRoundNumber,
			&item.Published,
			&item.TxIssued); err != nil {
			return nil, err
		}
		output = append(output, item)
	}
	return output, nil
}

func MainChainUpdatePowerTable(powermap map[string]int) error {
	mainchainDb, err := sql.Open("sqlite", mainchainpath)
	if err != nil {
		return err
	}

	defer mainchainDb.Close()
	_, err = mainchainDb.Exec("DELETE FROM PowerTable")
	if err != nil {
		return err
	}
	var placeholders []string
	valueArgs := make([]interface{}, 0)
	for key, element := range powermap {
		placeholders = append(placeholders, "(?, ?)")
		valueArgs = append(valueArgs, key)
		valueArgs = append(valueArgs, element)
	}
	sqlStmt := fmt.Sprintf("INSERT INTO PowerTable (hosts, Power) VALUES %s", strings.Join(placeholders, ","))
	_, err = mainchainDb.Exec(sqlStmt, valueArgs...)
	return err

}

func MainChainSetTxIssued(rowId int, value bool) error {
	mainchainDb, err := sql.Open("sqlite", mainchainpath)
	if err != nil {
		return err
	}
	defer mainchainDb.Close()
	_, err = mainchainDb.Exec("UPDATE MarketMatching Set TxIssued=? Where rowid = ?", value, rowId)
	if err != nil {
		return err
	}
	return nil
}

func MainChainSetPublished(rowId int, value bool) error {
	mainchainDb, err := sql.Open("sqlite", mainchainpath)
	if err != nil {
		return err
	}
	defer mainchainDb.Close()
	_, err = mainchainDb.Exec("UPDATE MarketMatching Set Published=? Where rowid = ?", value, rowId)
	if err != nil {
		return err
	}
	return nil
}

func MainChainSetPublishedAndStartRoundOnServAgrId(ServAgrId int, published bool, StartedMcRoundNumber int) error {
	mainchainDb, err := sql.Open("sqlite", mainchainpath)
	if err != nil {
		return err
	}
	defer mainchainDb.Close()
	_, err = mainchainDb.Exec("UPDATE MarketMatching Set Published=?, StartedMcRoundNumber=? Where ServAgrId = ?", published, StartedMcRoundNumber, ServAgrId)
	if err != nil {
		return err
	}
	return nil
}

func MainChainPopFromSecondQueue() (*MainChainSecondQueueEntry, error, bool) {
	mainchainDb, err := sql.Open("sqlite", mainchainpath)
	if err != nil {
		return nil, err, false
	}
	defer mainchainDb.Close()
	row := mainchainDb.QueryRow("SELECT Size, Time, IssuedMcRoundNumber FROM SecondQueue Where rowid in (Select MIN(rowid) FROM SecondQueue)")
	var timeStr string
	var returnValue MainChainSecondQueueEntry
	err = row.Scan(&returnValue.Size, &timeStr, &returnValue.RoundIssued)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, true
		}
		return nil, err, false
	}
	returnValue.Time, err = time.Parse(time.RFC3339, timeStr)
	_, err = mainchainDb.Exec("DELETE FROM SecondQueue WHERE rowid in (Select MIN(rowid) FROM SecondQueue)")
	if err != nil {
		return nil, err, false
	}
	return &returnValue, nil, false
}

func MainChainGetSecondQueue() ([]MainChainSecondQueueEntry, error) {
	mainchainDb, err := sql.Open("sqlite", mainchainpath)
	if err != nil {
		return nil, err
	}
	defer mainchainDb.Close()
	rows, err := mainchainDb.Query("SELECT rowid, Size, Time, IssuedMcRoundNumber FROM SecondQueue")
	var timeStr string
	var retval []MainChainSecondQueueEntry = make([]MainChainSecondQueueEntry, 0)

	for rows.Next() {
		var entry MainChainSecondQueueEntry
		err = rows.Scan(&entry.RowId, &entry.Size, &timeStr, &entry.RoundIssued)
		if err != nil {
			return nil, err
		}
		entry.Time, err = time.Parse(time.RFC3339, timeStr)
		retval = append(retval, entry)
	}
	return retval, nil
}

func MainChainDeleteFromSecondQueue(threshold int) error {
	mainchainDb, err := sql.Open("sqlite", mainchainpath)
	if err != nil {
		return err
	}
	defer mainchainDb.Close()
	_, err = mainchainDb.Exec("DELETE FROM SecondQueue WHERE rowid <= ?", threshold)
	return err
}

func MainChainPopFromFirstQueue() (*MainChainFirstQueueEntry, error, bool) {
	mainchainDb, err := sql.Open("sqlite", mainchainpath)
	if err != nil {
		return nil, err, false
	}
	defer mainchainDb.Close()
	row := mainchainDb.QueryRow("SELECT Size, Time, IssuedMcRoundNumber, Name, ServAgrId  FROM FirstQueue Where rowid in (Select MIN(rowid) FROM FirstQueue)")
	var returnValue MainChainFirstQueueEntry
	var timeStr string
	err = row.Scan(&returnValue.Size, &timeStr, &returnValue.RoundIssued, &returnValue.Name, &returnValue.ServAgrId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, true
		}
		return nil, err, false
	}
	returnValue.Time, err = time.Parse(time.RFC3339, timeStr)
	_, err = mainchainDb.Exec("DELETE FROM FirstQueue WHERE rowid in (Select MIN(rowid) FROM FirstQueue)")
	if err != nil {
		return nil, err, false
	}
	return &returnValue, nil, false
}

func MainChainGetFirstQueue() ([]MainChainFirstQueueEntry, error) {
	mainchainDb, err := sql.Open("sqlite", mainchainpath)
	if err != nil {
		return nil, err
	}
	defer mainchainDb.Close()
	rows, err := mainchainDb.Query("SELECT rowid, Size, Time, IssuedMcRoundNumber, Name, ServAgrId  FROM FirstQueue")
	var timeStr string
	var retval []MainChainFirstQueueEntry = make([]MainChainFirstQueueEntry, 0)

	for rows.Next() {
		var entry MainChainFirstQueueEntry
		err = rows.Scan(&entry.RowId, &entry.Size, &timeStr, &entry.RoundIssued, &entry.Name, &entry.ServAgrId)
		if err != nil {
			return nil, err
		}
		entry.Time, err = time.Parse(time.RFC3339, timeStr)
		retval = append(retval, entry)
	}
	return retval, nil
}

func MainChainDeleteFromFirstQueue(threshold int) error {
	mainchainDb, err := sql.Open("sqlite", mainchainpath)
	if err != nil {
		return err
	}
	defer mainchainDb.Close()
	_, err = mainchainDb.Exec("DELETE FROM FirstQueue WHERE rowid <= ?", threshold)
	return err
}

func AddToRoundTableBasedOnRoundNumber(column string, data interface{}, roundNumber int) error {
	mainchainDb, err := sql.Open("sqlite", mainchainpath)
	if err != nil {
		return err
	}
	defer mainchainDb.Close()
	stmt := fmt.Sprintf("UPDATE RoundTable SET %s = ? where RoundNumber = ?", column)
	_, err = mainchainDb.Exec(stmt, data, roundNumber)
	return err
}

func AddStatsToRoundTableBasedOnRoundNumber(
	BCSize int,
	regPayTx int,
	PoRTx int,
	StorjPayTx int,
	CntPropTx int,
	CntCmtTx int,
	TotalNumTx int,
	AveWaitOtherTxs float64,
	AveWaitRegPay float64,
	SyncTx int,
	RoundNumber int) error {

	mainchainDb, err := sql.Open("sqlite", mainchainpath)
	if err != nil {
		return err
	}
	defer mainchainDb.Close()
	stmt := `UPDATE RoundTable Set BCSize = ?,
                                RegPayTx = ?,
                                PoRTx = ?,
                                StorjPayTx = ?,
                                CntPropTx = ?,
                                CntCmtTx = ?,
                                TotalNumTx = ?,
                                AveWaitOtherTxs = ?,
                                AveWaitRegPay = ?,
                                SyncTx = ?
            WHERE RoundNumber = ?`
	_, err = mainchainDb.Exec(stmt, BCSize, regPayTx, PoRTx, StorjPayTx, CntPropTx, CntCmtTx,
		TotalNumTx, AveWaitOtherTxs, AveWaitRegPay, SyncTx, RoundNumber)
	return err
}

func SideChainRoundTableGetLastRow() (int, *SideChainRoundInfo, error) {
	var err error
	sidechainDb, err := sql.Open("sqlite", sidechainpath)
	if err != nil {
		return 0, nil, err
	}
	defer sidechainDb.Close()
	stmt := "SELECT IFNULL(MAX(rowId),0) FROM RoundTable"
	row := sidechainDb.QueryRow(stmt)
	var lastRow int
	err = row.Scan(&lastRow)
	if err != nil || lastRow == 0 {
		return 0, nil, err
	}
	stmt = `SELECT
                    RoundNumber,
                    BCSize,
                    RoundLeader,
                    PoRTx,
                    StartTime,
                    AveWait,
                    TotalNumTx,
                    BlockSpaceFull,
                    TimeTaken,
                    McRound
            FROM RoundTable Where rowid = ?`
	row = sidechainDb.QueryRow(stmt, lastRow)
	var info SideChainRoundInfo
	var timeStr string
	err = row.Scan(&info.RoundNumber, &info.BCSize, &info.RoundLeader, &info.PoRTx, &timeStr,
		&info.AveWait, &info.TotalNumTx,
		&info.BlockSpaceFull, &info.TimeTaken, &info.McRound)
	if err != nil {
		return 0, nil, err
	}
	info.StartTime, err = time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return 0, nil, err
	}
	return lastRow, &info, nil
}

func UpdateRowSideChainRoundTable(roundNbr int,
	BCSize int,
	roundLeader string,
	PorTx int,
	startTime time.Time,
	aveWait int,
	TotalNumTx int,
	BlockSpaceFull int,
	TimeTaken int,
	McRound int,
	rowid int) error {
	var err error
	sidechainDb, err := sql.Open("sqlite", sidechainpath)
	if err != nil {
		return err
	}
	defer sidechainDb.Close()
	_, err = sidechainDb.Exec(`Update RoundTable SET
                                        RoundNumber = ?
                                        BCSize = ?,
                                        RoundLeader =?,
                                        PoRTx =?,
                                        StartTime =?,
                                        AveWait =?,
                                        TotalNumTx =?,
                                        BlockSpaceFull =?,
                                        TimeTaken =?,
                                        McRound = ?)
                                 Where RowId = ?`,
		roundNbr, BCSize, roundLeader, PorTx, startTime.Format(time.RFC3339), aveWait,
		TotalNumTx, BlockSpaceFull, TimeTaken, McRound, rowid)
	return err
}

func SideChainPopFromFirstQueue() (*SideChainFirstQueueEntry, error, bool) {
	sidechainDb, err := sql.Open("sqlite", sidechainpath)
	if err != nil {
		return nil, err, false
	}
	defer sidechainDb.Close()
	row := sidechainDb.QueryRow(`SELECT Name,
                                        Size,
                                        Time,
                                        IssuedScRoundNumber,
                                        ServAgrId,
                                        MCRoundNbr
                                 FROM FirstQueue Where rowid in (Select MIN(rowid) FROM FirstQueue)`)
	var returnValue SideChainFirstQueueEntry
	var timeStr string
	err = row.Scan(&returnValue.Name, &returnValue.Size, &timeStr, &returnValue.IssuedScRoundNumber, &returnValue.ServAgrId, &returnValue.MCRoundNbr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, true
		}
		return nil, err, false
	}
	returnValue.Time, err = time.Parse(time.RFC3339, timeStr)
	_, err = sidechainDb.Exec("DELETE FROM FirstQueue WHERE rowid in (Select MIN(rowid) FROM FirstQueue)")
	if err != nil {
		return nil, err, false
	}
	return &returnValue, nil, false
}

func SideChainGetFirstQueue() ([]SideChainFirstQueueEntry, error) {
	sidechainDb, err := sql.Open("sqlite", sidechainpath)
	if err != nil {
		return nil, err
	}
	defer sidechainDb.Close()
	rows, err := sidechainDb.Query(`SELECT rowid,
                                        Name,
                                        Size,
                                        Time,
                                        IssuedScRoundNumber,
                                        ServAgrId,
                                        MCRoundNbr FROM FirstQueue`)
	var timeStr string
	var retval []SideChainFirstQueueEntry = make([]SideChainFirstQueueEntry, 0)

	for rows.Next() {
		var entry SideChainFirstQueueEntry
		err = rows.Scan(&entry.RowId, &entry.Name, &entry.Size, &timeStr, &entry.IssuedScRoundNumber, &entry.ServAgrId, &entry.MCRoundNbr)
		if err != nil {
			return nil, err
		}
		entry.Time, err = time.Parse(time.RFC3339, timeStr)
		retval = append(retval, entry)
	}
	return retval, nil
}

func SideChainDeleteFromFirstQueue(threshold int) error {
	sidechainDb, err := sql.Open("sqlite", sidechainpath)
	if err != nil {
		return err
	}
	defer sidechainDb.Close()
	_, err = sidechainDb.Exec("DELETE FROM FirstQueue WHERE rowid <= ?", threshold)
	return err
}

func SideChainRoundTableSetBlockSpaceIsFull(roundnbr int) error {
	stmt := "Update RoundTable SET BlockSpaceFull=True Where RoundNumber = ?"
	sidechainDb, err := sql.Open("sqlite", sidechainpath)
	if err != nil {
		return err
	}
	defer sidechainDb.Close()
	_, err = sidechainDb.Exec(stmt, roundnbr)
	return err
}

func SideChainRoundTableSetFinalRoundInfo(blocksize int, PorTx int, avewait float64, roundnbr int) error {
	stmt := "Update RoundTable SET BCSize=?, PorTx=?, AveWait=?  Where RowId = (Select Max(RowId) From RoundTable Where RoundNumber = ?)"
	sidechainDb, err := sql.Open("sqlite", sidechainpath)
	if err != nil {
		return err
	}
	defer sidechainDb.Close()
	_, err = sidechainDb.Exec(stmt, blocksize, PorTx, avewait, roundnbr)
	return err
}
