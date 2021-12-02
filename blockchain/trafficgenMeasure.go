// note that functions: ScalarLen(), PointLen() and size.Of() return the size in bytes
// ScalarLength, for example is a suit function, which in turn calls a group function with the same name,
// and thenreturns "mod.NewInt64(0, Order).MarshalSize()"

package blockchain

import (
	"math/rand"
	"strconv"
	"time"

	"github.com/DmitriyVTitov/size"
	"github.com/basedfs/log"
	"github.com/basedfs/vrf"

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
	roundNumber   string /* Instead of the "fiel tag" (Tau []byte),
	the round number is sent and stored on chain (later sent to sideChain in summery) and the verifiers
	can reproduce the file tag (as well as random query)
	from that round's seed as a source of randomness */
}

/* TxContractPropose: A client create a contract and add approprite (duration*price) escrow and sign it and issue a contract propose transaction
which has the contract and payment (escrow) info in it */
type TxContractPropose struct {
	tx               *TxPay
	ContractID       *Contract
	clientCommitment string
}

//ToDo: ContractCommit
type TxContractCommit struct {
	serverCommitment string
	ContractID       string
}

/* ---------------- transactions that will be issued until a contract is active ---------------- */

/* por txs are designed in away that the verifier (any miner) has sufficient information to verify it */
type TxPoR struct {
	ContractID  string
	por         *por.Por
	roundNumber string //uint // to determine the random query used for it
}

/* ---------------- transactions that will be issued after a contract is expired ---------------- */

type TxStoragePay struct {
	ContractID string
	tx         *TxPay
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
	TxContractProposes   []*TxContractPropose
	TxContractProposeCnt string //uint
	//---
	TxContractCommits   []*TxContractCommit
	TxContractCommitCnt string
	//---
	TxStoragePay    []*TxStoragePay
	TxStoragePayCnt string
	Fees            string //float64
}
type BlockHeader struct {
	RoundNumber string //uint
	// next round's seed for VRF based leader election which is the output of this round's leader's proof verification: VerifyBytes
	// _, (next round's seed)RoundSeed := (current round's leader)VrfPubkey.VerifyBytes((current round's leader)proof, (current round's seed)t)
	RoundSeed         [64]byte
	LeadershipProof   [80]byte
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
// BlockMeasurement compute the size of meta data and every thing other than the transactions inside the block
func BlockMeasurement() (BlockSizeMinusTransactions int) {
	hash := "f3f6a909f8521adb57d898d2985834e632374e770fd9e2b98656f1bf1fdfd42701"
	// ---------------- block sample ----------------
	var TxPayArraySample []*TxPay
	var TxPorArraySample []*TxPoR
	var TxContractProposeArraySample []*TxContractPropose
	var TxContractCommitSample []*TxContractCommit
	var TxStoragePaySample []*TxStoragePay

	x9 := &TransactionList{
		//---
		TxPays:   TxPayArraySample,
		TxPayCnt: "010",
		//---
		TxPoRs:   TxPorArraySample,
		TxPoRCnt: "010",
		//---
		TxContractProposes:   TxContractProposeArraySample,
		TxContractProposeCnt: "03",
		//---
		TxContractCommits:   TxContractCommitSample,
		TxContractCommitCnt: "09",
		//---
		TxStoragePay:    TxStoragePaySample,
		TxStoragePayCnt: "02",
		Fees:            "10000000000",
	}
	// real! TransactionListSize = size.Of(x9) + sum of size of included transactions
	// ---
	t := []byte("first round's seed")
	VrfPubkey, VrfPrivkey := vrf.VrfKeygen()
	proof, _ := VrfPrivkey.ProveBytes(t)
	_, vrfOutput := VrfPubkey.VerifyBytes(proof, t)
	var nextroundseed [64]byte = vrfOutput
	var VrfProof [80]byte = proof
	// ---
	x10 := &BlockHeader{
		RoundNumber:         "10000000000",
		RoundSeed:           nextroundseed,
		LeadershipProof:     VrfProof,
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
	BlockSizeMinusTransactions = size.Of(x10) + size.Of(x9) + size.Of(x11)
	// ---
	log.LLvl2("Block Size Minus Transactions is: ", BlockSizeMinusTransactions)

	return BlockSizeMinusTransactions
}

// TransactionMeasurement computes the size of 5 types of transactions we currently have in the system:
// Por, ContractPropose, Pay, StoragePay, ContractCommit
func TransactionMeasurement(SectorNumber int) (PorTxSize int, ContractProposeTxSize int, PayTxSize int, StoragePayTxSize int, ContractCommitTxSize int) {
	// ---------------- payment transaction sample ----------------
	hash := "f3f6a909f8521adb57d898d2985834e632374e770fd9e2b98656f1bf1fdfd42701"
	r := rand.New(rand.NewSource(99))

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

	// ---------------- ContractPropose transaction sample ----------------
	x5 := &Contract{
		duration:      time.Now(),
		fileSize:      "10000000000",
		startRound:    "10000000000",
		pricePerRound: "10000000000",
		roundNumber:   "10000000000",
	}
	x7 := &TxContractPropose{
		tx:               x4,
		ContractID:       x5,
		clientCommitment: hash,
	}
	ContractProposeTxSize = size.Of(x7) + PayTxSize + size.Of(x5)
	log.LLvl2("size of a Contract Propose transaction (including contract creation tx) is: ", ContractProposeTxSize, "bytes \n with ",
		size.Of(x5), " bytes for contract, \n and ", PayTxSize, " bytes for payment")

	// ---------------- por transaction sample ----------------
	var randombyte = make([]byte, 8)
	rand := random.New()
	var muArraySample []kyber.Scalar

	for i := range muArraySample {
		muArraySample[i] = por.Suite.Scalar().Pick(blake2xb.New(randombyte))
	}
	sigmaSample := por.Suite.G1().Point().Mul(por.Suite.G1().Scalar().Pick(rand), nil)

	x8 := &por.Por{
		Mu:    muArraySample,
		Sigma: sigmaSample,
	}

	porSize := SectorNumber*por.Suite.G1().ScalarLen() + por.Suite.G2().PointLen() + size.Of(x8)

	sk, _ := por.RandomizedKeyGeneration()
	//Tau, pf := por.RandomizedFileStoring(sk, por.GenerateFile())
	_, pf := por.RandomizedFileStoring(sk, por.GenerateFile(SectorNumber), SectorNumber)
	p := por.CreatePoR(pf, SectorNumber)

	x6 := &TxPoR{
		ContractID:  strconv.Itoa(r.Int()),
		por:         &p,
		roundNumber: "10000000000",
	}
	PorTxSize = size.Of(x6) + porSize
	log.LLvl2("size of a por transaction is: ", PorTxSize, " bytes \n with ",
		SectorNumber*por.Suite.G1().ScalarLen()+por.Suite.G2().PointLen(), " bytes for pure por")
	// ---------------- TxStoragePay transaction sample ----------------
	x9 := &TxStoragePay{
		ContractID: strconv.Itoa(r.Int()),
		tx:         x4,
	}
	StoragePayTxSize = size.Of(x9) + PayTxSize
	log.LLvl2("size of a StoragePay transaction is: ", StoragePayTxSize)
	// ---------------- TxContractCommit transaction sample ----------------
	x10 := TxContractCommit{
		serverCommitment: hash,
		ContractID:       strconv.Itoa(r.Int()),
	}
	ContractCommitTxSize = size.Of(x10)
	log.LLvl2("size of a ContractCommit transaction is: ", ContractCommitTxSize)

	return PorTxSize, ContractProposeTxSize, PayTxSize, StoragePayTxSize, ContractCommitTxSize
}
