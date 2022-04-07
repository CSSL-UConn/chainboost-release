// note that functions: ScalarLen(), PointLen() and size.Of() return the size in bytes
// ScalarLength, for example is a suit function, which in turn calls a group function with the same name,
// and thenreturns "mod.NewInt64(0, Order).MarshalSize()"
// length of uint64: 8 byte

/* Note that all nodes (same in mc. and sc.) can verify each epoch’s leader and committee from the mc.

mc leader: verify the authenticity of: the leader and the involved committee of a proposed sync tx.
leader and committee: verify the correctness of: POR tx.*/

package blockchain

import (
	"crypto/sha256"
	"math/rand"
	"time"

	"github.com/DmitriyVTitov/size"
	"github.com/chainBoostScale/ChainBoost/MainAndSideChain/BLSCoSi"
	"github.com/chainBoostScale/ChainBoost/onet/log"

	//"github.com/chainBoostScale/ChainBoost/vrf"

	// ToDoRaha: later that I brought everything from blscosi package to ChainBoost package, I shoudl add another pacckage with
	// some definitions in it to be imported/used in blockchain(here) and simulation package (instead of using blscosi/protocol)
	"github.com/chainBoostScale/ChainBoost/por"
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

//server agreement
type ServAgr struct {
	duration      [2]byte
	fileSize      [4]byte
	startRound    [3]byte
	pricePerRound [3]byte
	Tau           []byte
	//MCRoundNumber   [3]byte //ToDoRaha: why MCRoundNumber is commented here?
	/* later: Instead of the "file tag" (Tau []byte),
	the round number is sent and stored on chain and the verifiers
	can reproduce the file tag (as well as random query)
	from that round's seed as a source of randomness */
}

/* TxServAgrPropose: A client create a ServAgr and add approprite (duration*price) escrow and sign it and issue a ServAgr propose transaction
which has the ServAgr and payment (escrow) info in it */
type TxServAgrPropose struct {
	tx               *TxPay
	ServAgrID        *ServAgr
	clientCommitment [71]byte
}

type TxServAgrCommit struct {
	serverCommitment [71]byte
	ServAgrID        uint64
}

/* ---------------- transactions that will be issued (with ChainBoost: each side chain's round / Pure MainChain: each main chain's round) until a ServAgr is active ---------------- */

/* por txs are designed in away that the verifier (any miner) has sufficient information to verify it */
type TxPoR struct {
	ServAgrID     uint64
	por           *por.Por
	MCRoundNumber [3]byte // to determine the random query used for it
}

/* ---------------- transactions that will be issued after a ServAgr is expired(?) ---------------- */

type TxStoragePay struct {
	ServAgrID uint64
	tx        *TxPay
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
(based on the balance of currency for that key), along with the total weight of all outstanding currency." //ToDoRaha: compelete this later
*/
type TransactionList struct {
	//---
	TxPays   []*TxPay
	TxPayCnt [2]byte
	//---
	TxPoRs   []*TxPoR
	TxPoRCnt [2]byte
	//---
	TxServAgrProposes   []*TxServAgrPropose
	TxServAgrProposeCnt [2]byte
	//---
	TxServAgrCommits   []*TxServAgrCommit
	TxServAgrCommitCnt [2]byte
	//---
	TxStoragePay    []*TxStoragePay
	TxStoragePayCnt [2]byte
	//--- tx from side chain
	TxSCSync    []*TxSCSync
	TxSCSyncCnt [2]byte
	//---
	Fees [3]byte
}

type BlockHeader struct {
	MCRoundNumber [3]byte
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
	var MCRoundNumberSample [3]byte
	// ---------------- block sample ----------------
	var TxPayArraySample []*TxPay
	var TxPorArraySample []*TxPoR
	var TxServAgrProposeArraySample []*TxServAgrPropose
	var TxServAgrCommitSample []*TxServAgrCommit
	var TxStoragePaySample []*TxStoragePay

	x9 := &TransactionList{
		//---
		TxPays:   TxPayArraySample,
		TxPayCnt: cnt,
		//---
		TxPoRs:   TxPorArraySample,
		TxPoRCnt: cnt,
		//---
		TxServAgrProposes:   TxServAgrProposeArraySample,
		TxServAgrProposeCnt: cnt,
		//---
		TxServAgrCommits:   TxServAgrCommitSample,
		TxServAgrCommitCnt: cnt,
		//---
		TxStoragePay:    TxStoragePaySample,
		TxStoragePayCnt: cnt,

		Fees: feeSample,
	}
	// real! TransactionListSize = size.Of(x9) + sum of size of included transactions
	// --- VRF
	//raha: ToDoRaha: temp comment
	// t := []byte("first round's seed")
	// VrfPubkey, VrfPrivkey := vrf.VrfKeygen()
	// proof, _ := VrfPrivkey.ProveBytes(t)
	// _, vrfOutput := VrfPubkey.VerifyBytes(proof, t)
	// var nextroundseed [64]byte =  // vrfOutput
	// var VrfProof [80]byte = proof
	// --- time
	ti := []byte(time.Now().String())
	var timeSample [4]byte
	copy(timeSample[:], ti[:])
	// ---

	x10 := &BlockHeader{
		MCRoundNumber: MCRoundNumberSample,
		//RoundSeed:         nextroundseed,
		//LeadershipProof:   VrfProof,
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
		len(MCRoundNumberSample) + /*ToDoRaha: temp comment: len(nextroundseed) + len(VrfProof) + */ len(hashSample) + len(timeSample) + len(hashSample) + len(Version) + //x10
		5*len(cnt) + len(feeSample) //x9
	// ---
	log.Lvl3("Block Size Minus Transactions is: ", BlockSizeMinusTransactions)

	return BlockSizeMinusTransactions
}

// TransactionMeasurement computes the size of 5 types of transactions we currently have in the system:
// Por, ServAgrPropose, Pay, StoragePay, ServAgrCommit
func TransactionMeasurement(SectorNumber, SimulationSeed int) (PorTxSize uint32, ServAgrProposeTxSize uint32, PayTxSize uint32, StoragePayTxSize uint32, ServAgrCommitTxSize uint32) {
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
	r := rand.New(rand.NewSource(int64(SimulationSeed)))
	var Version, SequenceNumber, index, LockingScriptSize, fileSizeSample, UnlockingScriptSize [4]byte
	var Amount [8]byte
	var startRoundSample, MCRoundNumberSample, pricePerRoundSample [3]byte
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

	PayTxSize = uint32(len(hashSample) + len(index) + // outpoint
		len(UnlockingScriptSize) + len(UnlockinScriptSample) + len(SequenceNumber) + //TxPayIn
		len(Amount) + len(LockingScriptSample) + len(LockingScriptSize) + //TxPayOut
		len(timeSample) + len(Version) + len(cnt) + len(cnt)) //TxPay
	log.Lvl3("size of a pay transaction is: ", PayTxSize, "bytes")
	// ---------------- por transaction sample  ----------------

	sk, _ := por.RandomizedKeyGeneration()
	Tau, pf := por.RandomizedFileStoring(sk, por.GenerateFile(SectorNumber), SectorNumber)

	// ---------------- ServAgrPropose transaction sample ----------------
	x5 := &ServAgr{
		duration:      duration,
		fileSize:      fileSizeSample,
		startRound:    startRoundSample,
		pricePerRound: pricePerRoundSample,
		Tau:           Tau,
		//MCRoundNumber:   MCRoundNumberSample,
	}
	x7 := &TxServAgrPropose{
		tx:               x4,
		ServAgrID:        x5,
		clientCommitment: cmtSample,
	}

	log.Lvl5("x5 is:", x5, " and x7 is: ", x7)

	ServAgrProposeTxSize = PayTxSize + //tx
		uint32(len(duration)+len(fileSizeSample)+len(startRoundSample)+len(pricePerRoundSample)+len(Tau)+ //ServAgr tx
			len(cmtSample)) //clientCommitment

	log.Lvl3("size of a ServAgr Propose transaction (including ServAgr creation tx) is: ", ServAgrProposeTxSize,
		"bytes \n with ",
		len(duration)+len(fileSizeSample)+len(startRoundSample)+len(pricePerRoundSample)+len(Tau), " bytes for ServAgr, \n and ",
		PayTxSize, " bytes for payment")

	// ---------------- por transaction sample ----------------
	var randombyte = make([]byte, 8)
	rand := random.New()
	var muArraySample []kyber.Scalar
	var ServAgrIdSample = r.Uint64()

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
	p := por.CreatePoR(pf, SectorNumber, SimulationSeed)

	x6 := &TxPoR{
		ServAgrID:     ServAgrIdSample,
		por:           &p,
		MCRoundNumber: MCRoundNumberSample,
	}

	log.Lvl5("tx por is: ", x6)

	PorTxSize = uint32(porSize /*size of pur por*/ +
		8 /*len(ServAgrIdSample)*/ + len(MCRoundNumberSample)) //TxPoR

	log.Lvl3("size of a por transaction is: ", PorTxSize, " bytes \n with ",
		SectorNumber*por.Suite.G1().ScalarLen()+por.Suite.G2().PointLen(), " bytes for pure por")
	// ---------------- TxStoragePay transaction sample ----------------
	x9 := &TxStoragePay{
		ServAgrID: ServAgrIdSample,
		tx:        x4,
	}

	log.Lvl5("tx StoragePay is: ", x9)

	StoragePayTxSize = 8 /*len(ServAgrIdSample)*/ + PayTxSize
	log.Lvl3("size of a StoragePay transaction is: ", StoragePayTxSize)
	// ---------------- TxServAgrCommit transaction sample ----------------
	x10 := TxServAgrCommit{
		serverCommitment: cmtSample,
		ServAgrID:        ServAgrIdSample,
	}

	log.Lvl5("tx ServAgrCommit is: ", x10)

	ServAgrCommitTxSize = uint32(len(cmtSample) + 8) /*len(ServAgrIdSample)*/
	log.Lvl3("size of a ServAgrCommit transaction is: ", ServAgrCommitTxSize)

	return PorTxSize, ServAgrProposeTxSize, PayTxSize, StoragePayTxSize, ServAgrCommitTxSize
}

/* -------------------------------------------------------------------- */
//  ----------------  Side Chain -------------------------------------
/* -------------------------------------------------------------------- */

/* -------------------------------------------------------------------------------------------
---------------- transaction and block types used just in sidechain ----------------
-------------------------------------------------------------------------------------------- */

type SCMetaBlockTransactionList struct {
	//---
	TxPoRs   []*TxPoR
	TxPoRCnt [2]byte
	//---
	Fees [3]byte
}
type SCBlockHeader struct {
	SCRoundNumber [3]byte
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
	// LeaderPublicKey kyber.Point
	// the combined signature of committee members for each meta/summery blocks should be
	// included in their header `SCBlockHeader` which enables future validation
	BlsSignature BLSCoSi.BlsSignature
}
type SCMetaBlock struct {
	BlockSize                  [3]byte
	SCBlockHeader              *SCBlockHeader
	SCMetaBlockTransactionList *SCMetaBlockTransactionList
}
type TxSummery struct {
	//---
	ServAgrID       []uint64
	ConfirmedPoRCnt []uint64
	//---
}
type SCSummeryBlockTransactionList struct {
	//---
	TxSummery    []*TxSummery
	TxSummeryCnt [2]byte
	//---
	Fees [3]byte
}
type SCSummeryBlock struct {
	BlockSize                     [3]byte
	SCBlockHeader                 *SCBlockHeader
	SCSummeryBlockTransactionList *SCSummeryBlockTransactionList
}

// side chain's Sync transaction is the result of summerizing the summery block of each epoch in side chain
type TxSCSync struct {
	//---
	// this information "should be kept in side chain" in `SCSummeryBlock` and
	// ofcourse be sent to mainchain via `TxSCSync` to make it's effect on mainchain
	ServAgrID       []uint64
	ConfirmedPoRCnt []uint64
	//---
}

/* -------------------------------------------------------------------------------------------
    ------------- measuring side chain's sync transaction, summery and meta blocks ------
-------------------------------------------------------------------------------------------- */

// MetaBlockMeasurement compute the size of meta data and every thing other than the transactions inside the meta block
func SCBlockMeasurement() (SummeryBlockSizeMinusTransactions int, MetaBlockSizeMinusTransactions int) {
	// ----- block header sample -----
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
	var SCRoundNumberSample [3]byte
	var samplePublicKey [33]byte
	//var samplePublicKey kyber.Point
	// --- VRF
	// t := []byte("first round's seed")
	// VrfPubkey, VrfPrivkey := vrf.VrfKeygen()
	// proof, _ := VrfPrivkey.ProveBytes(t)
	// _, vrfOutput := VrfPubkey.VerifyBytes(proof, t)
	// var nextroundseed [64]byte = vrfOutput
	// var VrfProof [80]byte = proof
	// --- time
	ti := []byte(time.Now().String())
	var timeSample [4]byte
	copy(timeSample[:], ti[:])
	// ---
	x10 := &SCBlockHeader{
		SCRoundNumber: SCRoundNumberSample,
		//RoundSeed:         nextroundseed,
		//LeadershipProof:   VrfProof,
		PreviousBlockHash: hashSample,
		Timestamp:         timeSample,
		MerkleRootHash:    hashSample,
		Version:           Version,
		LeaderPublicKey:   samplePublicKey,
		//BlsSignature:      sampleBlsSig, // this will be added back in the protocol
	}
	// ---------------- meta block sample ----------------
	var TxPorArraySample []*TxPoR

	x9 := &SCMetaBlockTransactionList{
		TxPoRs:   TxPorArraySample,
		TxPoRCnt: cnt,
		Fees:     feeSample,
	}
	// real! TransactionListSize = size.Of(x9) + sum of size of included transactions

	x11 := &SCMetaBlock{
		BlockSize:                  BlockSizeSample,
		SCBlockHeader:              x10,
		SCMetaBlockTransactionList: x9,
	}

	log.Lvl5(x11)

	MetaBlockSizeMinusTransactions = len(BlockSizeSample) + //x11: SCMetaBlock
		len(SCRoundNumberSample) + /* len(nextroundseed) + len(VrfProof) +*/ len(hashSample) + len(timeSample) +
		len(hashSample) + len(Version) + len(samplePublicKey) + //x10: SCBlockHeader
		len(cnt) + len(feeSample) //x9: SCMetaBlockTransactionList
	// ---
	log.Lvl3("Meta Block Size Minus Transactions is: ", MetaBlockSizeMinusTransactions)

	//------------------------------------- Summery block -----------------------------
	// ---------------- summery block sample ----------------
	var TxSummeryArraySample []*TxSummery

	x12 := &SCSummeryBlockTransactionList{
		//---
		TxSummery:    TxSummeryArraySample,
		TxSummeryCnt: cnt,
		Fees:         feeSample,
	}
	x13 := &SCSummeryBlock{
		BlockSize:                     BlockSizeSample,
		SCBlockHeader:                 x10,
		SCSummeryBlockTransactionList: x12,
	}
	log.LLvl5(x13)
	SummeryBlockSizeMinusTransactions = len(BlockSizeSample) + //x13: SCSummeryBlock
		len(SCRoundNumberSample) + /*len(nextroundseed) + len(VrfProof) +*/ len(hashSample) + len(timeSample) + len(hashSample) +
		len(Version) + len(samplePublicKey) + //x10: SCBlockHeader
		len(cnt) + len(feeSample) //x12: SCSummeryBlockTransactionList
	log.Lvl3("Summery Block Size Minus Transactions is: ", SummeryBlockSizeMinusTransactions)

	return SummeryBlockSizeMinusTransactions, MetaBlockSizeMinusTransactions
}

// SyncTransactionMeasurement computes the size of sync transaction
func SyncTransactionMeasurement() (SyncTxSize int) {
	return
}
func SCSummeryTxMeasurement(SummTxNum int) (SummTxsSizeInSummBlock int) {
	r := rand.New(rand.NewSource(int64(0)))
	var a []uint64
	for i := 0; i < SummTxNum; i++ {
		a = append(a, r.Uint64())
	}
	return 2 * len(a)
}
