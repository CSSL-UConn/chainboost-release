package blockchain

import (
	"strconv"
	"time"

	"github.com/DmitriyVTitov/size"
	"github.com/basedfs/log"

	"github.com/basedfs/por"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/util/random"
	"go.dedis.ch/kyber/v3/xof/blake2xb"
)

/*  payment transactions are close to byzcoin's and from the structure
provided in: https://developer.bitcoin.org/reference/transactions.html */
type outpoint struct {
	hash  string //[32]string //TXID
	index string //[4]byte
}
type TxPayIn struct {
	outpoint            *outpoint // 36 bytes
	UnlockingScriptSize string    //uint
	UnlockinScript      string    // signature script
	SequenceNumber      string    //uint
}
type TxPayOut struct {
	Amount            string //[8]byte
	LockingScript     string // pubkey script
	LockingScriptSize string //uint
}
type TxPay struct {
	LockTime string //uint
	Version  string //[4]byte
	TxInCnt  string //uint // compactSize uint
	TxOutCnt string //uint
	TxIns    []*TxPayIn
	TxOuts   []*TxPayOut
}

/* ---------------- market matching transactions ---------------- */
type Contract struct {
	duration      time.Time
	fileSize      string //uint
	startRound    string //uint
	pricePerRound string
	// -- ToDo: commitment construiction?
	serverCommitment string
	clientCommitment string
}

/* ---------------- transactions that will be issued until a contract is active ---------------- */

/* after matching is done (contract is created) the client creates an escrow */
type TxEscrow struct {
	tx         *TxPay
	ContractID *Contract
}

/* por txs are designed in away that the verifier (any miner) has sufficient information to verify it */
type TxPoR struct {
	ContractID  *Contract
	por         *por.Por
	Tau         []byte
	roundNumber string //uint // to determine the random query used for it
}

/* ---------------- transactions that will be issued for each round ---------------- */
/* the proof is generated as an output of fucntion call "ProveBytes" in vrf package*/
/* ---------------- block structure and its metadata ----------------
(from algorand) : "Blocks consist of a list of transactions,  along with metadata needed by MainChain miners.
Specifically, the metadata consists of
	- the round number,
	- the proposer’s VRF-based seed,
	- a hash of the previous block in the ledger,and
	- a timestamp indicating when the block was proposed
The list of transactions in a block logically translates to a set of weights for each user’s public key
(based on the balance of currency for that key), along with the total weight of all outstanding currency." //ToDo: compelete this later
*/
type TransactionList struct {
	//---
	TxPays   []*TxPay
	TxPayCnt string //uint
	//---
	TxPoRs   []*TxPoR
	TxPoRCnt string //uint
	//---
	TxEscrows   []*TxEscrow
	TxEscrowCnt string //uint
	//---
	Fees string //float64
}
type BlockHeader struct {
	RoundNumber       string //uint
	RoundSeed         string //proposer’s VRF-based seed
	PreviousBlockHash string
	Timestamp         time.Time
	//--
	MerkleRootHash string
	// -- ToDo: these two are required as well, right?
	Version             string // [4]byte
	SignedBlockByLeader string // []byte //length?
	LeaderAddress       string //ToDo: public key?
}
type Block struct {
	BlockSize       string
	BlockHeader     *BlockHeader
	TransactionList *TransactionList
}

/* -------------------------------------------------------------------- */
//  ----------------  Block and Transactions size measurements -----
/* -------------------------------------------------------------------- */
func BlockMeasurement(
	blocksize int,
	nodes int,
	payTxShare int,
	escrowTxShare int,
	porTxShare int) (PorTxSize int,
	EscrowTxSize int,
	PayTxSize int) {
	log.LLvl2("number of nodes: ", nodes,
		"\n payTxShare: ", payTxShare,
		"\n escrowTxShare: ", escrowTxShare,
		"\n porTxShare: ", porTxShare) // todo: remove two last shares

	// ---------------- payment transaction sample ----------------
	hash := "f3f6a909f8521adb57d898d2985834e632374e770fd9e2b98656f1bf1fdfd42701"
	x1 := &outpoint{
		hash:  hash,
		index: "000000",
	}
	x2 := &TxPayIn{
		outpoint:            x1,
		UnlockingScriptSize: "6b",
		UnlockinScript:      "48304502203a776322ebf8eb8b58cc6ced4f2574f4c73aa664edce0b0022690f2f6f47c521022100b82353305988cb0ebd443089a173ceec93fe4dbfe98d74419ecc84a6a698e31d012103c5c1bc61f60ce3d6223a63cedbece03b12ef9f0068f2f3c4a7e7f06c523c3664",
		SequenceNumber:      "ffffffff",
	}
	x3 := &TxPayOut{
		Amount:            "60e3160000000000",
		LockingScript:     "76a914977ae6e32349b99b72196cb62b5ef37329ed81b488ac063d1000000000001976a914f76bc4190f3d8e2315e5c11c59cfc8be9df747e388ac",
		LockingScriptSize: "19",
	}
	xin := []*TxPayIn{x2}
	xout := []*TxPayOut{x3}
	x4 := &TxPay{
		LockTime: "00000000",
		Version:  "01000000",
		TxInCnt:  "01",
		TxOutCnt: "02",
		TxIns:    xin,
		TxOuts:   xout,
	}

	TxInCntInt, _ := strconv.Atoi(x4.TxInCnt)
	TxOutCntInt, _ := strconv.Atoi(x4.TxOutCnt)
	PayTxSize = size.Of(x4) + TxInCntInt*size.Of(x2) + TxOutCntInt*size.Of(x3)
	log.LLvl2("size of a pay transaction is: ", PayTxSize, " bytes \n with ",
		TxInCntInt, " number of input transaction, each ", size.Of(x2),
		" bytes \n and ", TxOutCntInt, " number of output transaction, each ", size.Of(x3),
		"\n plus size of payment transaction itself which is ", size.Of(x4))

	// ---------------- escrow transaction sample ----------------
	x5 := &Contract{
		duration:      time.Now(),
		fileSize:      "10000000000",
		startRound:    "10000000000",
		pricePerRound: "10000000000",

		serverCommitment: hash,
		clientCommitment: hash,
	}
	x7 := &TxEscrow{
		tx:         x4,
		ContractID: x5,
	}
	EscrowTxSize = size.Of(x7) + (size.Of(x4) + TxInCntInt*size.Of(x2) + TxOutCntInt*size.Of(x3)) + size.Of(x5)
	log.LLvl2("size of a escrow transaction (including contract creation tx) is: ", EscrowTxSize, "bytes \n with ",
		size.Of(x5), " bytes for contract, \n and ", PayTxSize, " bytes for payment")

	// ---------------- por transaction sample ----------------
	var randombyte = make([]byte, 8)
	rand := random.New()
	var muArraySample [por.S]kyber.Scalar
	for i := range muArraySample {
		muArraySample[i] = por.Suite.Scalar().Pick(blake2xb.New(randombyte))
	}
	sigmaSample := por.Suite.G1().Point().Mul(por.Suite.G1().Scalar().Pick(rand), nil)
	x8 := &por.Por{
		Mu:    muArraySample,
		Sigma: sigmaSample,
	}
	porSize := por.S*por.Suite.G1().ScalarLen() + por.Suite.G2().PointLen() + size.Of(x8)

	sk, _ := por.RandomizedKeyGeneration()
	Tau, pf := por.RandomizedFileStoring(sk, por.GenerateFile())
	p := por.CreatePoR(pf)

	x6 := &TxPoR{
		ContractID:  x5,
		por:         &p,
		Tau:         Tau,
		roundNumber: "10000000000",
	}
	PorTxSize = size.Of(x6) + size.Of(x8) + porSize
	log.LLvl2("size of a por transaction is: ", PorTxSize, " bytes \n with ",
		size.Of(x8)+porSize, " bytes for pure por")

	// ---------------- block sample ----------------
	var TxPayArraySample []*TxPay
	for i := 1; i <= nodes; i++ {
		TxPayArraySample = append(TxPayArraySample, x4)
	}
	var TxPorArraySample []*TxPoR
	for i := 1; i <= nodes; i++ {
		TxPorArraySample = append(TxPorArraySample, x6)
	}
	var TxEscrowArraySample []*TxEscrow
	for i := 1; i <= nodes; i++ {
		TxEscrowArraySample = append(TxEscrowArraySample, x7)
	}
	x9 := &TransactionList{
		//---
		TxPays:   TxPayArraySample,
		TxPayCnt: "010",
		//---
		TxPoRs:   TxPorArraySample,
		TxPoRCnt: "010",
		//---
		TxEscrows:   TxEscrowArraySample,
		TxEscrowCnt: "03",
		//---
		Fees: "10000000000",
	}
	TransactionListSize := nodes*(PayTxSize+PorTxSize) + size.Of(x9)
	// ---
	x10 := &BlockHeader{
		RoundNumber:         "10000000000",
		RoundSeed:           hash,
		PreviousBlockHash:   "9500c43a25c624520b5100adf82cb9f9da72fd2447a496bc600b000000000000",
		Timestamp:           time.Now(),
		MerkleRootHash:      "6cd862370395dedf1da2841ccda0fc489e3039de5f1ccddef0e834991a65600e",
		Version:             "01000000",
		SignedBlockByLeader: hash,
		LeaderAddress:       hash,
	}
	x11 := &Block{
		BlockSize:       "10000000000",
		BlockHeader:     x10,
		TransactionList: x9,
	}
	BlockSize := size.Of(x10) + TransactionListSize + size.Of(x11)
	// ---

	log.LLvl2("size of a Block Header is: ", size.Of(x10), " bytes, ",
		"\n size of Transaction List is: ", TransactionListSize, " bytes, ",
		"\n overall, size of Block is: ", BlockSize)

	return PorTxSize, EscrowTxSize, PayTxSize
}

func TransactionMeasurement() (PorTxSize int, EscrowTxSize int, PayTxSize int) {

	// ---------------- payment transaction sample ----------------
	hash := "f3f6a909f8521adb57d898d2985834e632374e770fd9e2b98656f1bf1fdfd42701"
	x1 := &outpoint{
		hash:  hash,
		index: "000000",
	}
	x2 := &TxPayIn{
		outpoint:            x1,
		UnlockingScriptSize: "6b",
		UnlockinScript:      "48304502203a776322ebf8eb8b58cc6ced4f2574f4c73aa664edce0b0022690f2f6f47c521022100b82353305988cb0ebd443089a173ceec93fe4dbfe98d74419ecc84a6a698e31d012103c5c1bc61f60ce3d6223a63cedbece03b12ef9f0068f2f3c4a7e7f06c523c3664",
		SequenceNumber:      "ffffffff",
	}
	x3 := &TxPayOut{
		Amount:            "60e3160000000000",
		LockingScript:     "76a914977ae6e32349b99b72196cb62b5ef37329ed81b488ac063d1000000000001976a914f76bc4190f3d8e2315e5c11c59cfc8be9df747e388ac",
		LockingScriptSize: "19",
	}
	xin := []*TxPayIn{x2}
	xout := []*TxPayOut{x3}
	x4 := &TxPay{
		LockTime: "00000000",
		Version:  "01000000",
		TxInCnt:  "01",
		TxOutCnt: "01",
		TxIns:    xin,
		TxOuts:   xout,
	}

	TxInCntInt, _ := strconv.Atoi(x4.TxInCnt)
	TxOutCntInt, _ := strconv.Atoi(x4.TxOutCnt)
	PayTxSize = size.Of(x4) + TxInCntInt*size.Of(x2) + TxOutCntInt*size.Of(x3)
	log.LLvl2("size of a pay transaction is: ", PayTxSize, " bytes \n with ",
		TxInCntInt, " number of input transaction, each ", size.Of(x2),
		" bytes \n and ", TxOutCntInt, " number of output transaction, each ", size.Of(x3),
		"\n plus size of payment transaction itself which is ", size.Of(x4))

	// ---------------- escrow transaction sample ----------------
	x5 := &Contract{
		duration:      time.Now(),
		fileSize:      "10000000000",
		startRound:    "10000000000",
		pricePerRound: "10000000000",

		serverCommitment: hash,
		clientCommitment: hash,
	}
	x7 := &TxEscrow{
		tx:         x4,
		ContractID: x5,
	}
	EscrowTxSize = size.Of(x7) + PayTxSize + size.Of(x5)
	log.LLvl2("size of a escrow transaction (including contract creation tx) is: ", EscrowTxSize, "bytes \n with ",
		size.Of(x5), " bytes for contract, \n and ", PayTxSize, " bytes for payment")

	// ---------------- por transaction sample ----------------
	var randombyte = make([]byte, 8)
	rand := random.New()
	var muArraySample [por.S]kyber.Scalar
	for i := range muArraySample {
		muArraySample[i] = por.Suite.Scalar().Pick(blake2xb.New(randombyte))
	}
	sigmaSample := por.Suite.G1().Point().Mul(por.Suite.G1().Scalar().Pick(rand), nil)
	x8 := &por.Por{
		Mu:    muArraySample,
		Sigma: sigmaSample,
	}
	porSize := por.S*por.Suite.G1().ScalarLen() + por.Suite.G2().PointLen() + size.Of(x8)

	sk, _ := por.RandomizedKeyGeneration()
	Tau, pf := por.RandomizedFileStoring(sk, por.GenerateFile())
	p := por.CreatePoR(pf)

	x6 := &TxPoR{
		ContractID:  x5,
		por:         &p,
		Tau:         Tau,
		roundNumber: "10000000000",
	}
	PorTxSize = size.Of(x6) + size.Of(x8) + porSize
	log.LLvl2("size of a por transaction is: ", PorTxSize, " bytes \n with ",
		size.Of(x8)+porSize, " bytes for pure por")

	return PorTxSize, EscrowTxSize, PayTxSize
}
