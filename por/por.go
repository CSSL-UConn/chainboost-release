//-----------------------------------------------------------------------
//  --------------- Compact PoR ----------------------------------------
//-----------------------------------------------------------------------
// https://link.springer.com/article/10.1007%2Fs00145-012-9129-2
package por

import (
	"bytes"
	"encoding/binary"
	onet "github.com/basedfs"
	"github.com/basedfs/log"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/kyber/v3/pairing/bn256"
	"go.dedis.ch/kyber/v3/sign/bls"
	"go.dedis.ch/kyber/v3/sign/schnorr"
	"go.dedis.ch/kyber/v3/util/key"
	"go.dedis.ch/kyber/v3/util/random"
	"go.dedis.ch/kyber/v3/xof/blake2xb"
	"math/rand"
	"strconv"
)
const s = 5 // number of sectors in eac block (sys. par.)
//Each sector is one element of Zp,and there are s sectors per block.
//If the processed file is b bits long,then there are n=[b/s lg p] blocks.
const n = 10 // number of blocks
const l = 5  //size of query set (i<n)
//-----------------------------------------------------------------------
type processedFile struct {
	m_ij  initialFile
	sigma [n]kyber.Point
}
type initialFile struct {
	m [n][s]kyber.Scalar
}
type randomQuery struct {
	i   [l]int
	v_i [l]kyber.Scalar
}
type Por struct {
	mu    [s]kyber.Scalar
	sigma kyber.Point
}
type privateKey struct {
	alpha kyber.Scalar
	ssk   kyber.Scalar
}
type publicKey struct {
	v   kyber.Point
	spk kyber.Point
}
// utility functions
func RandomizedKeyGeneration() ( privateKey, publicKey){
	//randomizedKeyGeneration: pubK=(alpha,ssk),prK=(v,spk)
	clientKeyPair := key.NewKeyPair(onet.Suite) //what specific suite is needed here?
	ssk := clientKeyPair.Private
	spk := clientKeyPair.Public
	//BLS keyPair
	//Package bn256: implements the Optimal Ate pairing over a
	//256-bit Barreto-Naehrig curve as described in
	//http://cryptojedi.org/papers/dclxvi-20100714.pdf.
	//claimed 128-bit security level.
	//Package bn256 from kyber library is used in blscosi module for bls scheme
	suite := pairing.NewSuiteBn256()
	private, public := bls.NewKeyPair(suite, random.New())
	alpha := private
	v := public
	return privateKey{
			alpha,
			ssk,
		},publicKey{
			spk: spk,
			v: v,
		}
}
func GenerateFile () (initialFile) {
	suite := pairing.NewSuiteBn256()
	// first apply the erasure code to obtain M′; then split M′
	// into n blocks (for some n), each s sectors long:
	// {mij} 1≤i≤n 1≤j≤s
	var m_ij [n][s]kyber.Scalar
	for i := 0; i < n; i++ {
		for j := 0; j < s; j++ {
			m_ij[i][j] = suite.Scalar().Pick(suite.RandomStream())
		}
	}
	return initialFile{m: m_ij}
}
func randomizedVerifyingQuery() *randomQuery {
	suite := pairing.NewSuiteBn256()
	//--------- randomness: initialize the seed base on a known blockchain related param
	var blockchainRandomSeed = 3 // it needs to be related to some bc params later
	var randombyte = make([]byte, 8)
	binary.LittleEndian.PutUint64(randombyte, uint64(blockchainRandomSeed))
	var rng = blake2xb.New(randombyte)
	rand.Seed(int64(blockchainRandomSeed))
	//-----------------------
	var b[l] int
	var v[l] kyber.Scalar
	for i:=0; i<l; i++{
		b[i] = rand.Intn(n)
		v[i] = suite.Scalar().Pick(rng)
	}
	return &randomQuery{i: b, v_i:    v}
}
// this function is called by the file owner to create file tag - file authentication values - and key-pair
func  RandomizedFileStoring(sk privateKey, initialfile initialFile) ( []byte, processedFile) {
	suite := pairing.NewSuiteBn256()
	m_ij := initialfile.m
	ns := []byte(strconv.Itoa(n))
	//a random file name from some sufficiently large domain (e.g.,Zp)
	aRandomFileName := random.Int(bn256.Order, random.New())
	//u1,..,us random from G
	var u [s]		kyber.Scalar
	var U [s]		kyber.Point
	var st1,st2 bytes.Buffer
	for j := 0; j < s; j++ {
		rand := random.New()
		u[j] = suite.G1().Scalar().Pick(rand)
		U[j] = suite.G1().Point().Mul(u[j], nil)
		t, _ := u[j].MarshalBinary()
		st1.Write(t)
	}
	//Tau0 := "name"||string(n)||u1||...||us
	//Tau=Tau0||Ssig(ssk)(Tau0) "File Tag"
	//Tau0 := aRandomFileName.String() + "||" +  ns + "||" +  st
	st2.Write(aRandomFileName.Bytes())
	st2.Write(ns)
	st2.ReadFrom(&st1)
	Tau0 := st2
	//sg, _ := schnorr.Sign(onet.Suite, sk.ssk, []byte(Tau0))
	//Tau := Tau0 + string(sg)
	sg, _ := schnorr.Sign(onet.Suite, sk.ssk, Tau0.Bytes())
	Tau := append(Tau0.Bytes(), sg...)
	// ----  isn't there another way?---------------------------------------
	type hashablePoint interface {
		Hash([]byte) kyber.Point
	}
	hashable, ok := suite.G1().Point().(hashablePoint)
	if !ok {
		log.LLvl2("err")
	}
	// --------------------------------------------------------------------
	//create "AuthValue" (Sigma_i) for block i
	//Sigma_i = Hash(name||i).P(j=1,..,s)u_j^m_ij
	var b[n] kyber.Point
	for i := 0; i < n; i++ {
		h := hashable.Hash(append(aRandomFileName.Bytes(), byte(i)))
		p := suite.G1().Point()
		for j := 0; j < s; j++ {
			p = p.Add(p, U[j].Mul(m_ij[i][j], U[j]))
		}
		b[i] = p.Mul(sk.alpha, p.Add(p, h))
	}
	return Tau, processedFile{
		m_ij: initialFile{m: m_ij},
		sigma:  b,
	}
}
// this function will be called by the server who wants to create a PoR
// in the paper this function takes 3 inputs: public key , file tag , and processedFile -
// I don't see why the first two parameters are needed!
func CreatePoR(processedfile processedFile) Por {
	// "the query can be generated from a short seed using a random oracle,
	// and this short seed can be transmitted instead of the longer query."
	// note: this function is called by the verifier in the paper but a prover who have access
	// to the random seed (from blockchain) can call this (and get the query) herself.
	rq := randomizedVerifyingQuery()
	suite := pairing.NewSuiteBn256()
	m_ij := processedfile.m_ij.m
	sigma := processedfile.sigma
	var Mu [s] kyber.Scalar
	for j:=0; j<s; j++{
		tv := suite.Scalar().Mul(rq.v_i[0], m_ij[rq.i[0]][j])
		for i:=1; i<l; i++{
			//Mu_j= S(Q)(v_i.m_ij)
			tv = suite.Scalar().Add(suite.Scalar().Mul(rq.v_i[i], m_ij[rq.i[i]][j]), tv)
		}
		Mu [j] = tv
	}
	p := suite.G1().Point()
	for i:=1; i<l; i++{
		//sigma=P(Q)(sigma_i^v_i)
		p = p.Add(p, p.Mul(rq.v_i[i], sigma[i]))
	}
	return Por{
		mu: Mu,
		sigma: p,
	}
}
// verifyPoR: servers will verify por tx.s when they recieve it
// in the paper this function takes 3 inputs: public key , private key, and file tag
// I don't see why the private key is needed!
func VerifyPoR(pk publicKey, Tau []byte, p Por) (bool, error) {
	suite := pairing.NewSuiteBn256()
	rq := randomizedVerifyingQuery()
	//check the file tag (Tau) integrity
	error := schnorr.Verify(onet.Suite, pk.spk, Tau[:32+2+s*32],Tau[32+2+s*32:])
	if error != nil{log.LLvl2("issue in verifying the file tag signature:", error)}
	// ----  isn't there another way?---------------------------------------
	type hashablePoint interface {Hash([]byte) kyber.Point}
	hashable, ok := suite.G1().Point().(hashablePoint)
	if !ok {log.LLvl2("err")}
	// --------------------------------------------------------------------
	//extract the random selected points
	rightTermPoint  := suite.G1().Point() // pairing check: right term of right hand side
	for j:=0; j<s; j++ {
		//These numbers (length of components) can be extracted from calling
		//some utility function in kyber. in case we wanted to customize stuff!
		thisPoint := Tau[32+2+j*32 : 32+2+(j+1)*32]
		u := suite.Scalar().SetBytes(thisPoint)
		U := suite.G1().Point().Mul(u, nil)
		rightTermPoint = rightTermPoint.Add(rightTermPoint, U.Mul(p.mu[j], U))
	}
	leftTermPoint := suite.G1().Point() // pairing check: left term of right hand side
	for i := 0; i < l; i++ {
		t := rq.i[i]
		h := hashable.Hash(append(Tau[:32], byte(t)))
		leftTermPoint = leftTermPoint.Add(leftTermPoint, h.Mul(rq.v_i[i],h))
	}
	//check: e(sigma, g) =? e(PHash(name||i)^v_i.P(j=1,..,s)(u_j^mu_j,v)
	if suite.Pair(leftTermPoint.Add(leftTermPoint, rightTermPoint), pk.v) != suite.Pair(p.sigma, pk.v.Base()){
		log.LLvl2("no")
	}
	var refuse = false
	return refuse, error
}
//-------------------------------------------------------------
// move it to a seperate test file
func Testpor() {
	sk, pk := RandomizedKeyGeneration()
	Tau, pf := RandomizedFileStoring(sk, GenerateFile())
	p := CreatePoR(pf)
	d, _ := VerifyPoR(pk, Tau, p)
	if d==false{log.LLvl2(d)}
}