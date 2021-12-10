// note that functions: ScalarLen(), PointLen() and size.Of() return the size in bytes
// ScalarLength, for example is a suit function, which in turn calls a group function with the same name,
// and thenreturns "mod.NewInt64(0, Order).MarshalSize()"
// length of uint64: 8 byte
package blockchain

import (
	"crypto/sha256"
	"math/rand"
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
	hash  [32]byte //TXID
	index [4]byte
}
type TxPayIn struct { // Each input spends an outpoint from a previous transaction
	outpoint            *outpoint // 36 bytes
	UnlockingScriptSize [4]byte   // see https://developer.bitcoin.org/reference/transactions.html#compactsize-unsigned-integers
	UnlockinScript      []byte    // signature script // see https://medium.com/coinmonks/on-bitcoin-transaction-sizes-97e31bc9d816 73?
	SequenceNumber      [4]byte
}
type TxPayOut struct {
	Amount            [8]byte
	LockingScript     []byte // pubkey script // see https://medium.com/coinmonks/on-bitcoin-transaction-sizes-97e31bc9d816 35?
	LockingScriptSize [4]byte
}
type TxPay struct {
	LockTime [4]byte
	Version  [4]byte
	TxInCnt  [1]byte // compactSize uint see https://developer.bitcoin.org/reference/transactions.html#compactsize-unsigned-integers
	TxOutCnt [1]byte
	TxIns    []*TxPayIn
	TxOuts   []*TxPayOut
}

/* ---------------- market matching transactions ---------------- */
type Contract struct {
	duration      [2]byte
	fileSize      [4]byte
	startRound    [3]byte
	pricePerRound [3]byte
	Tau           []byte
	//roundNumber   [3]byte
	/* later: Instead of the "file tag" (Tau []byte),
	the round number is sent and stored on chain (later sent to sideChain in summery) and the verifiers
	can reproduce the file tag (as well as random query)
	from that round's seed as a source of randomness */
}

/* TxContractPropose: A client create a contract and add approprite (duration*price) escrow and sign it and issue a contract propose transaction
which has the contract and payment (escrow) info in it */
type TxContractPropose struct {
	tx               *TxPay
	ContractID       *Contract
	clientCommitment [71]byte
}

//ToDo: ContractCommit
type TxContractCommit struct {
	serverCommitment [71]byte
	ContractID       uint64
}

/* ---------------- transactions that will be issued until a contract is active ---------------- */

/* por txs are designed in away that the verifier (any miner) has sufficient information to verify it */
type TxPoR struct {
	ContractID  uint64
	por         *por.Por
	roundNumber [3]byte // to determine the random query used for it
}

/* ---------------- transactions that will be issued after a contract is expired ---------------- */

type TxStoragePay struct {
	ContractID uint64
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
	TxPayCnt [2]byte
	//---
	TxPoRs   []*TxPoR
	TxPoRCnt [2]byte
	//---
	TxContractProposes   []*TxContractPropose
	TxContractProposeCnt [2]byte
	//---
	TxContractCommits   []*TxContractCommit
	TxContractCommitCnt [2]byte
	//---
	TxStoragePay    []*TxStoragePay
	TxStoragePayCnt [2]byte
	Fees            [3]byte
}
type BlockHeader struct {
	RoundNumber [3]byte
	// next round's seed for VRF based leader election which is the output of this round's leader's proof verification: VerifyBytes
	// _, (next round's seed)RoundSeed := (current round's leader)VrfPubkey.VerifyBytes((current round's leader)proof, (current round's seed)t)
	RoundSeed         [64]byte
	LeadershipProof   [80]byte
	PreviousBlockHash [32]byte
	Timestamp         [4]byte
	//--
	MerkleRootHash  [32]byte
	Version         [4]byte
	LeaderPublicKey [33]byte // see https://medium.com/coinmonks/on-bitcoin-transaction-sizes-97e31bc9d816
}
type Block struct {
	BlockSize       [3]byte
	BlockHeader     *BlockHeader
	TransactionList *TransactionList
}

/* -------------------------------------------------------------------- */
//  ----------------  Block and Transactions size measurements -----
/* -------------------------------------------------------------------- */
// BlockMeasurement compute the size of meta data and every thing other than the transactions inside the block
func BlockMeasurement() (BlockSizeMinusTransactions int) {
	// -- Hash Sample ----
	sha := sha256.New()
	if _, err := sha.Write([]byte("a sample seed")); err != nil {
		log.Error("Couldn't hash header:", err)
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}
	hash := sha.Sum(nil)
	var hashSample [32]uint8
	copy(hashSample[:], hash[:])
	// --
	var Version [4]byte
	var cnt [2]byte
	var feeSample, BlockSizeSample [3]byte
	var roundNumberSample [3]byte
	// ---------------- block sample ----------------
	var TxPayArraySample []*TxPay
	var TxPorArraySample []*TxPoR
	var TxContractProposeArraySample []*TxContractPropose
	var TxContractCommitSample []*TxContractCommit
	var TxStoragePaySample []*TxStoragePay

	x9 := &TransactionList{
		//---
		TxPays:   TxPayArraySample,
		TxPayCnt: cnt,
		//---
		TxPoRs:   TxPorArraySample,
		TxPoRCnt: cnt,
		//---
		TxContractProposes:   TxContractProposeArraySample,
		TxContractProposeCnt: cnt,
		//---
		TxContractCommits:   TxContractCommitSample,
		TxContractCommitCnt: cnt,
		//---
		TxStoragePay:    TxStoragePaySample,
		TxStoragePayCnt: cnt,

		Fees: feeSample,
	}
	// real! TransactionListSize = size.Of(x9) + sum of size of included transactions
	// --- VRF
	t := []byte("first round's seed")
	VrfPubkey, VrfPrivkey := vrf.VrfKeygen()
	proof, _ := VrfPrivkey.ProveBytes(t)
	_, vrfOutput := VrfPubkey.VerifyBytes(proof, t)
	var nextroundseed [64]byte = vrfOutput
	var VrfProof [80]byte = proof
	// --- time
	ti := []byte(time.Now().String())
	var timeSample [4]byte
	copy(timeSample[:], ti[:])
	// ---

	x10 := &BlockHeader{
		RoundNumber:       roundNumberSample,
		RoundSeed:         nextroundseed,
		LeadershipProof:   VrfProof,
		PreviousBlockHash: hashSample,
		Timestamp:         timeSample,
		MerkleRootHash:    hashSample,
		Version:           Version,
	}
	x11 := &Block{
		BlockSize:       BlockSizeSample,
		BlockHeader:     x10,
		TransactionList: x9,
	}

	log.Lvl5(x11)

	BlockSizeMinusTransactions = len(BlockSizeSample) + //x11
		len(roundNumberSample) + len(nextroundseed) + len(VrfProof) + len(hashSample) + len(timeSample) + len(hashSample) + len(Version) + //x10
		5*len(cnt) + len(feeSample) //x9
	// ---
	log.Lvl2("Block Size Minus Transactions is: ", BlockSizeMinusTransactions)

	return BlockSizeMinusTransactions
}

// TransactionMeasurement computes the size of 5 types of transactions we currently have in the system:
// Por, ContractPropose, Pay, StoragePay, ContractCommit
func TransactionMeasurement(SectorNumber int) (PorTxSize int, ContractProposeTxSize int, PayTxSize int, StoragePayTxSize int, ContractCommitTxSize int) {
	// -- Hash Sample ----
	sha := sha256.New()
	if _, err := sha.Write([]byte("a sample seed")); err != nil {
		log.Error("Couldn't hash header:", err)
		log.Lvl2("Panic Raised:\n\n")
		panic(err)
	}
	hash := sha.Sum(nil)
	var hashSample [32]byte
	copy(hashSample[:], hash[:])
	// ---------------- payment transaction sample ----------------
	r := rand.New(rand.NewSource(99))
	var Version, SequenceNumber, index, LockingScriptSize, fileSizeSample, UnlockingScriptSize [4]byte
	var Amount [8]byte
	var startRoundSample, roundNumberSample, pricePerRoundSample [3]byte
	var duration [2]byte
	var cmtSample [71]byte // see https://medium.com/coinmonks/on-bitcoin-transaction-sizes-97e31bc9d816
	UnlockinScriptSample := []byte("48304502203a776322ebf8eb8b58cc6ced4f2574f4c73aa664edce0b0022690f2f6f47c521022100b82353305988cb0ebd443089a173ceec93fe4dbfe98d74419ecc84a6a698e31d012103c5c1bc61f60ce3d6223a63cedbece03b12ef9f0068f2f3c4a7e7f06c523c3664")
	LockingScriptSample := []byte("76a914977ae6e32349b99b72196cb62b5ef37329ed81b488ac063d1000000000001976a914f76bc4190f3d8e2315e5c11c59cfc8be9df747e388ac")
	var cnt [1]byte
	// --- time
	t := []byte(time.Now().String())
	var timeSample [4]byte
	copy(timeSample[:], t[:])
	// ---
	x1 := &outpoint{
		hash:  hashSample,
		index: index,
	}
	x2 := &TxPayIn{
		outpoint:            x1,
		UnlockingScriptSize: UnlockingScriptSize,
		UnlockinScript:      UnlockinScriptSample,
		SequenceNumber:      SequenceNumber,
	}
	x3 := &TxPayOut{
		Amount:            Amount,
		LockingScript:     LockingScriptSample,
		LockingScriptSize: LockingScriptSize,
	}
	xin := []*TxPayIn{x2}
	xout := []*TxPayOut{x3}
	x4 := &TxPay{
		LockTime: timeSample,
		Version:  Version,
		TxInCnt:  cnt,
		TxOutCnt: cnt,
		TxIns:    xin,
		TxOuts:   xout,
	}

	PayTxSize = len(hashSample) + len(index) + // outpoint
		len(UnlockingScriptSize) + len(UnlockinScriptSample) + len(SequenceNumber) + //TxPayIn
		len(Amount) + len(LockingScriptSample) + len(LockingScriptSize) + //TxPayOut
		len(timeSample) + len(Version) + len(cnt) + len(cnt) //TxPay
	log.Lvl2("size of a pay transaction is: ", PayTxSize, "bytes")
	// ---------------- por transaction sample  ----------------

	sk, _ := por.RandomizedKeyGeneration()
	Tau, pf := por.RandomizedFileStoring(sk, por.GenerateFile(SectorNumber), SectorNumber)

	// ---------------- ContractPropose transaction sample ----------------
	x5 := &Contract{
		duration:      duration,
		fileSize:      fileSizeSample,
		startRound:    startRoundSample,
		pricePerRound: pricePerRoundSample,
		Tau:           Tau,
		//roundNumber:   roundNumberSample,
	}
	x7 := &TxContractPropose{
		tx:               x4,
		ContractID:       x5,
		clientCommitment: cmtSample,
	}

	log.Lvl5("x5 is:", x5, " and x7 is: ", x7)

	ContractProposeTxSize = PayTxSize + //tx
		len(duration) + len(fileSizeSample) + len(startRoundSample) + len(pricePerRoundSample) + len(Tau) + //contract tx
		len(cmtSample) //clientCommitment

	log.Lvl2("size of a Contract Propose transaction (including contract creation tx) is: ", ContractProposeTxSize,
		"bytes \n with ",
		len(duration)+len(fileSizeSample)+len(startRoundSample)+len(pricePerRoundSample)+len(Tau), " bytes for contract, \n and ",
		PayTxSize, " bytes for payment")

	// ---------------- por transaction sample ----------------
	var randombyte = make([]byte, 8)
	rand := random.New()
	var muArraySample []kyber.Scalar
	var contractIdSample = r.Uint64()

	for i := range muArraySample {
		muArraySample[i] = por.Suite.Scalar().Pick(blake2xb.New(randombyte))
	}
	sigmaSample := por.Suite.G1().Point().Mul(por.Suite.G1().Scalar().Pick(rand), nil)

	x8 := &por.Por{
		Mu:    muArraySample,
		Sigma: sigmaSample,
	}

	porSize := SectorNumber*por.Suite.G1().ScalarLen() + por.Suite.G2().PointLen() + size.Of(x8)

	//sk, _ := por.RandomizedKeyGeneration()
	//_, pf := por.RandomizedFileStoring(sk, por.GenerateFile(SectorNumber), SectorNumber)
	p := por.CreatePoR(pf, SectorNumber)

	x6 := &TxPoR{
		ContractID:  contractIdSample,
		por:         &p,
		roundNumber: roundNumberSample,
	}

	log.Lvl5("tx por is: ", x6)

	PorTxSize = porSize /*size of pur por*/ +
		8 /*len(contractIdSample)*/ + len(roundNumberSample) //TxPoR

	log.Lvl2("size of a por transaction is: ", PorTxSize, " bytes \n with ",
		SectorNumber*por.Suite.G1().ScalarLen()+por.Suite.G2().PointLen(), " bytes for pure por")
	// ---------------- TxStoragePay transaction sample ----------------
	x9 := &TxStoragePay{
		ContractID: contractIdSample,
		tx:         x4,
	}

	log.Lvl5("tx StoragePay is: ", x9)

	StoragePayTxSize = 8 /*len(contractIdSample)*/ + PayTxSize
	log.Lvl2("size of a StoragePay transaction is: ", StoragePayTxSize)
	// ---------------- TxContractCommit transaction sample ----------------
	x10 := TxContractCommit{
		serverCommitment: cmtSample,
		ContractID:       contractIdSample,
	}

	log.Lvl5("tx ContractCommit is: ", x10)

	ContractCommitTxSize = len(cmtSample) + 8 /*len(contractIdSample)*/
	log.Lvl2("size of a ContractCommit transaction is: ", ContractCommitTxSize)

	return PorTxSize, ContractProposeTxSize, PayTxSize, StoragePayTxSize, ContractCommitTxSize
}
