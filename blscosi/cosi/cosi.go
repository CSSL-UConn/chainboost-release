// --------------------------------------------------------------------------
//  cosi from:  https://github.com/dedis/cothority/tree/main/cosi/crypto
// --------------------------------------------------------------------------

/*
Package crypto is a temporary copy of gopkg.in/dedis/crypto.v0/cosi.

Package cosi is the Collective Signing implementation according to the paper of
Bryan Ford: http://arxiv.org/pdf/1503.08768v1.pdf .

Stages of CoSi

The CoSi-protocol has 4 stages:

1. Announcement: The leader multicasts an announcement
of the start of this round down through the spanning tree,
optionally including the statement S to be signed.

2. Commitment: Each node i picks a random scalar vi and
computes its individual commit Vi = Gvi . In a bottom-up
process, each node i waits for an aggregate commit Vˆj from
each immediate child j, if any. Node i then computes its
own aggregate commit Vˆi = Vi \prod{j ∈ Cj}{Vˆj}, where Ci is the
set of i’s immediate children. Finally, i passes Vi up to its
parent, unless i is the leader (node 0).

3. Challenge: The leader computes a collective challenge
c = H( Aggregate Commit ∥ Aggregate Public key || Message ),
then multicasts c down through the tree, along
with the statement S to be signed if it was not already
announced in phase 1.

4. Response: In a final bottom-up phase, each node i waits
to receive a partial aggregate response rˆj from each of
its immediate children j ∈ Ci. Node i now computes its
individual response ri = vi + cxi, and its partial aggregate
response rˆi = ri + \sum{j ∈ Cj}{rˆj} . Node i finally passes rˆi
up to its parent, unless i is the root.
*/
package cosi

import (
	//"crypto/cipher"

	"crypto/sha512"
	"errors"
	"fmt"
	"time"

	"github.com/ChainBoost/onet/network"
	//"github.com/dedis/crypto/config"
	"go.dedis.ch/kyber/v3/util/key"
	//"github.com/dedis/crypto/abstract"
	"go.dedis.ch/kyber/v3"
	//"github.com/ChainBoost/MainAndSideChain"
	"github.com/ChainBoost/onet"
)

// CoSi is the struct that implements one round of a CoSi protocol.
// It's important to only use this struct *once per round*, and if you  try to
// use it twice, it will try to alert you if it can.
// You create a CoSi struct by giving your secret key you wish to pariticipate
// with during the CoSi protocol, and the list of public keys representing the
// list of all co-signer's public keys involved in the round.
// To use CoSi, call three different functions on it which corresponds to the last
// three phases of the protocols:
//  - (Create)Commitment: creates a new secret and its commitment. The output has to
//  be passed up to the parent in the tree.
//  - CreateChallenge: the root creates the challenge from receiving all the
//  commitments. This output must be sent down the tree using Challenge()
//  function.
//  - (Create)Response: creates and possibly aggregates all responses and the
//  output must be sent up into the tree.
// The root can then issue `Signature()` to get the final signature that can be
// verified using `VerifySignature()`.
// To handle missing signers, the signature generation will append a bitmask at
// the end of the signature with each bit index set corresponding to a missing
// cosigner. If you need to specify a missing signer, you can call
// SetMaskBit(i int, enabled bool) which will set the signer i disabled in the
// mask. The index comes from the list of public keys you give when creating the
// CoSi struct. You can also give the full mask directly with SetMask().
type CoSi struct {
	// Suite used
	suite kyber.Group
	// mask is the mask used to select which signers participated in this round
	// or not. All code regarding the mask is directly inspired from
	// github.com/bford/golang-x-crypto/ed25519/cosi code.
	*mask
	// the message being co-signed
	message []byte
	// V_hat is the aggregated commit (our own + the children's)
	aggregateCommitment kyber.Point
	// challenge holds the challenge for this round
	challenge kyber.Scalar

	// the longterm private key CoSi will use during the response phase.
	// The private key must have its public version in the list of publics keys
	// given to CoSi.
	private kyber.Scalar
	// random is our own secret that we wish to commit during the commitment phase.
	random kyber.Scalar
	// commitment is our own commitment
	commitment kyber.Point
	// response is our own computed response
	response kyber.Scalar
	// aggregateResponses is the aggregated response from the children + our own
	aggregateResponse kyber.Scalar

	// timestamp of when the announcement is done (i.e. timestamp of the four
	// phases)
	timestamp int64
}

// NewCosi returns a new Cosi struct given the suite, the longterm secret, and
// the list of public keys. If some signers were not to be participating, you
// have to set the mask using `SetMask` method. By default, all participants are
// designated as participating. If you wish to specify which co-signers are
// participating, use NewCosiWithMask
func NewCosi(suite kyber.Group, private kyber.Scalar, publics []kyber.Point) *CoSi {
	cosi := &CoSi{
		suite:   suite,
		private: private,
	}
	// Start with an all-disabled participation mask, then set it correctly
	cosi.mask = newMask(suite, publics)
	return cosi
}

// CreateCommitment creates the commitment of a random secret generated from the
// given s stream. It returns the message to pass up in the tree. This is
// typically called by the leaves.
// func (c *CoSi) CreateCommitment(s cipher.Stream) kyber.Point {
// 	c.genCommit(s)
// 	return c.commitment
// }

// Commit creates the commitment / secret as in CreateCommitment and it also
// aggregate children commitments from the children's messages.
// func (c *CoSi) Commit(s cipher.Stream, subComms []kyber.Point) kyber.Point {
// 	// generate our own commit
// 	c.genCommit(s)

// 	// add our own commitment to the aggregate commitment
// 	c.aggregateCommitment = c.suite.Point().Add(c.suite.Point().Null(), c.commitment)
// 	// take the children commitments
// 	for _, com := range subComms {
// 		c.aggregateCommitment.Add(c.aggregateCommitment, com)
// 	}
// 	return c.aggregateCommitment

// }

// CreateChallenge creates the challenge out of the message it has been given.
// This is typically called by Root.
// func (c *CoSi) CreateChallenge(msg []byte) (kyber.Scalar, error) {
// 	// H( Commit || AggPublic || M)
// 	hash := sha512.New()
// 	if _, err := c.aggregateCommitment.MarshalTo(hash); err != nil {
// 		return nil, err
// 	}
// 	if _, err := c.mask.Aggregate().MarshalTo(hash); err != nil {
// 		return nil, err
// 	}
// 	hash.Write(msg)
// 	chalBuff := hash.Sum(nil)
// 	// reducing the challenge
// 	c.challenge = c.suite.Scalar().SetBytes(chalBuff)
// 	c.message = msg
// 	return c.challenge, nil
// }

// Challenge keeps in memory the Challenge from the message.
// func (c *CoSi) Challenge(challenge kyber.Scalar) {
// 	c.challenge = challenge
// }

// CreateResponse is called by a leaf to create its own response from the
// challenge + commitment + private key. It returns the response to send up to
// the tree.
// func (c *CoSi) CreateResponse() (kyber.Scalar, error) {
// 	err := c.genResponse()
// 	return c.response, err
// }

// Response generates the response from the commitment, challenge and the
// responses of its children.
// func (c *CoSi) Response(responses []kyber.Scalar) (kyber.Scalar, error) {
// 	//create your own response
// 	if err := c.genResponse(); err != nil {
// 		return nil, err
// 	}
// 	// Add our own
// 	c.aggregateResponse = c.suite.Scalar().Set(c.response)
// 	for _, resp := range responses {
// 		// add responses of child
// 		c.aggregateResponse.Add(c.aggregateResponse, resp)
// 	}
// 	return c.aggregateResponse, nil
// }

// Signature returns a signature using the same format as EdDSA signature
// AggregateCommit || AggregateResponse || Mask
// *NOTE*: Signature() is only intended to be called by the root since only the
// root knows the aggregate response.
// func (c *CoSi) Signature() []byte {
// 	// Sig = C || R || bitmask
// 	lenC := c.suite.PointLen()
// 	lenSig := lenC + c.suite.ScalarLen()
// 	sigC, err := c.aggregateCommitment.MarshalBinary()
// 	if err != nil {
// 		panic("Can't marshal Commitment")
// 	}
// 	sigR, err := c.aggregateResponse.MarshalBinary()
// 	if err != nil {
// 		panic("Can't generate signature !")
// 	}
// 	final := make([]byte, lenSig+c.mask.MaskLen())
// 	copy(final[:], sigC)
// 	copy(final[lenC:lenSig], sigR)
// 	copy(final[lenSig:], c.mask.mask)
// 	return final
// }

// VerifyResponses verifies the response this CoSi has against the aggregated
// public key the tree is using. This is callable by any nodes in the tree,
// after it has aggregated its responses. You can enforce verification at each
// level of the tree for faster reactivity.
func (c *CoSi) VerifyResponses(aggregatedPublic kyber.Point) error {
	k := c.challenge

	// k * -aggPublic + s * B = k*-A + s*B
	// from s = k * a + r => s * B = k * a * B + r * B <=> s*B = k*A + r*B
	// <=> s*B + k*-A = r*B
	minusPublic := c.suite.Point().Neg(aggregatedPublic)
	kA := c.suite.Point().Mul(k, minusPublic)
	sB := c.suite.Point().Mul(c.aggregateResponse, nil)
	left := c.suite.Point().Add(kA, sB)

	if !left.Equal(c.aggregateCommitment) {
		return errors.New("recreated commitment is not equal to one given")
	}

	return nil
}

// VerifySignature is the method to call to verify a signature issued by a Cosi
// struct. Publics is the WHOLE list of publics keys, the mask at the end of the
// signature will take care of removing the indivual public keys that did not
// participate
func VerifySignature(suite kyber.Group, publics []kyber.Point, message, sig []byte) error {
	lenC := suite.PointLen()
	lenSig := lenC + suite.ScalarLen()
	aggCommitBuff := sig[:lenC]
	aggCommit := suite.Point()
	if err := aggCommit.UnmarshalBinary(aggCommitBuff); err != nil {
		panic(err)
	}
	sigBuff := sig[lenC:lenSig]
	sigInt := suite.Scalar().SetBytes(sigBuff)
	maskBuff := sig[lenSig:]
	mask := newMask(suite, publics)
	mask.SetMask(maskBuff)
	aggPublic := mask.Aggregate()
	aggPublicMarshal, err := aggPublic.MarshalBinary()
	if err != nil {
		return err
	}

	hash := sha512.New()
	hash.Write(aggCommitBuff)
	hash.Write(aggPublicMarshal)
	hash.Write(message)
	buff := hash.Sum(nil)
	k := suite.Scalar().SetBytes(buff)

	// k * -aggPublic + s * B = k*-A + s*B
	// from s = k * a + r => s * B = k * a * B + r * B <=> s*B = k*A + r*B
	// <=> s*B + k*-A = r*B
	minusPublic := suite.Point().Neg(aggPublic)
	kA := suite.Point().Mul(k, minusPublic)
	sB := suite.Point().Mul(sigInt, nil)
	left := suite.Point().Add(kA, sB)

	if !left.Equal(aggCommit) {
		return errors.New("Signature invalid")
	}

	return nil
}

// AggregateResponse returns the aggregated response that this cosi has
// accumulated.
func (c *CoSi) AggregateResponse() kyber.Scalar {
	return c.aggregateResponse
}

// GetChallenge returns the challenge that were passed down to this cosi.
func (c *CoSi) GetChallenge() kyber.Scalar {
	return c.challenge
}

// GetCommitment returns the commitment generated by this CoSi (not aggregated).
func (c *CoSi) GetCommitment() kyber.Point {
	return c.commitment
}

// GetResponse returns the individual response generated by this CoSi
func (c *CoSi) GetResponse() kyber.Scalar {
	return c.response
}

// genCommit generates a random scalar vi and computes its individual commit
// Vi = G^vi
// func (c *CoSi) genCommit(s cipher.Stream) {
// 	if s == nil {
// 		panic("s is required")
// 	}
// 	c.random = c.suite.Scalar().Pick(s)
// 	c.commitment = c.suite.Point().Mul(c.random, nil)
// 	c.aggregateCommitment = c.commitment
// }

// genResponse creates the response
func (c *CoSi) genResponse() error {
	if c.private == nil {
		return errors.New("No private key given in this cosi")
	}
	if c.random == nil {
		return errors.New("No random scalar computed in this cosi")
	}
	if c.challenge == nil {
		return errors.New("No challenge computed in this cosi")
	}

	// resp = random - challenge * privatekey
	// i.e. ri = vi + c * xi
	resp := c.suite.Scalar().Mul(c.private, c.challenge)
	c.response = resp.Add(c.random, resp)
	// no aggregation here
	c.aggregateResponse = c.response
	// paranoid protection: delete the random
	c.random = nil
	return nil
}

// mask holds the mask utilities
type mask struct {
	mask      []byte
	publics   []kyber.Point
	aggPublic kyber.Point
	suite     kyber.Group
}

// newMask returns a new mask to use with the cosigning with all cosigners enabled
func newMask(suite kyber.Group, publics []kyber.Point) *mask {
	// Start with an all-disabled participation mask, then set it correctly
	cm := &mask{
		publics: publics,
		suite:   suite,
	}
	cm.mask = make([]byte, cm.MaskLen())
	cm.aggPublic = cm.suite.Point().Null()
	cm.allEnabled()
	return cm

}

// AllEnabled sets the pariticipation bit mask accordingly to make all
// signers participating.
func (cm *mask) allEnabled() {
	for i := range cm.mask {
		cm.mask[i] = 0xff // all disabled
	}
	cm.SetMask(make([]byte, len(cm.mask)))
}

// Set the entire participation bitmask according to the provided
// packed byte-slice interpreted in little-endian byte-order.
// That is, bits 0-7 of the first byte correspond to cosigners 0-7,
// bits 0-7 of the next byte correspond to cosigners 8-15, etc.
// Each bit is set to indicate the corresponding cosigner is disabled,
// or cleared to indicate the cosigner is enabled.
//
// If the mask provided is too short (or nil),
// SetMask conservatively interprets the bits of the missing bytes
// to be 0, or Enabled.
func (cm *mask) SetMask(mask []byte) error {
	if cm.MaskLen() != len(mask) {
		err := fmt.Errorf("CosiMask.MaskLen() is %d but is given %d bytes)", cm.MaskLen(), len(mask))
		return err
	}
	masklen := len(mask)
	for i := range cm.publics {
		byt := i >> 3
		bit := byte(1) << uint(i&7)
		if (byt < masklen) && (mask[byt]&bit != 0) {
			// Participant i disabled in new mask.
			if cm.mask[byt]&bit == 0 {
				cm.mask[byt] |= bit // disable it
				cm.aggPublic.Sub(cm.aggPublic, cm.publics[i])
			}
		} else {
			// Participant i enabled in new mask.
			if cm.mask[byt]&bit != 0 {
				cm.mask[byt] &^= bit // enable it
				cm.aggPublic.Add(cm.aggPublic, cm.publics[i])
			}
		}
	}
	return nil
}

// MaskLen returns the length in bytes
// of a complete disable-mask for this cosigner list.
func (cm *mask) MaskLen() int {
	return (len(cm.publics) + 7) >> 3
}

// SetMaskBit enables or disables the mask bit for an individual cosigner.
func (cm *mask) SetMaskBit(signer int, enabled bool) {
	if signer > len(cm.publics) {
		panic("SetMaskBit range out of index")
	}
	byt := signer >> 3
	bit := byte(1) << uint(signer&7)
	if !enabled {
		if cm.mask[byt]&bit == 0 { // was enabled
			cm.mask[byt] |= bit // disable it
			cm.aggPublic.Sub(cm.aggPublic, cm.publics[signer])
		}
	} else { // enable
		if cm.mask[byt]&bit != 0 { // was disabled
			cm.mask[byt] &^= bit
			cm.aggPublic.Add(cm.aggPublic, cm.publics[signer])
		}
	}
}

// MaskBit returns a boolean value indicating whether
// the indicated signer is enabled (true) or disabled (false)
func (cm *mask) MaskBit(signer int) bool {
	if signer > len(cm.publics) {
		panic("MaskBit given index out of range")
	}
	byt := signer >> 3
	bit := byte(1) << uint(signer&7)
	return (cm.mask[byt] & bit) != 0
}

// bytes returns the byte representation of the mask
// The bits that are left are set to a default value (1) for
// non malleability.
func (cm *mask) bytes() []byte {
	clone := make([]byte, len(cm.mask))
	for i := range clone {
		clone[i] = 0xff
	}
	copy(clone[:], cm.mask)
	return clone
}

// Aggregate returns the aggregate public key of all *participating* signers
func (cm *mask) Aggregate() kyber.Point {
	return cm.aggPublic
}

// --------------------------------------------------------------------------
// cosi from: https://github.com/dedis/cothority/tree/byzcoin_ng_first/protocols/byzcoin/cosi -
// --------------------------------------------------------------------------

/*
Package cosi is the Collective Signing implementation according to the paper of
Bryan Ford: http://arxiv.org/pdf/1503.08768v1.pdf .

Stages of CoSi

The CoSi-protocol has 4 stages:

1. Announcement: The leader multicasts an announcement
of the start of this round down through the spanning tree,
optionally including the statement S to be signed.

2. Commitment: Each node i picks a random secret vi and
computes its individual commit Vi = Gvi . In a bottom-up
process, each node i waits for an aggregate commit Vˆj from
each immediate child j, if any. Node i then computes its
own aggregate commit Vˆi = Vi \prod{j ∈ Cj}{Vˆj}, where Ci is the
set of i’s immediate children. Finally, i passes Vi up to its
parent, unless i is the leader (node 0).

3. Challenge: The leader computes a collective challenge c =
H(Vˆ0 ∥ S), then multicasts c down through the tree, along
with the statement S to be signed if it was not already
announced in phase 1.

4. Response: In a final bottom-up phase, each node i waits
to receive a partial aggregate response rˆj from each of
its immediate children j ∈ Ci. Node i now computes its
individual response ri = vi − cxi, and its partial aggregate
response rˆi = ri + \sum{j ∈ Cj}{rˆj} . Node i finally passes rˆi
up to its parent, unless i is the root.
*/

// // Cosi is the struct that implements the basic cosi.
// type Cosi struct {
// 	// Suite used
// 	suite abstract.Suite
// 	// the longterm private key we use during the rounds
// 	private abstract.Scalar
// 	// timestamp of when the announcement is done (i.e. timestamp of the four
// 	// phases)
// 	timestamp int64
// 	// random is our own secret that we wish to commit during the commitment phase.
// 	random abstract.Scalar
// 	// commitment is our own commitment
// 	commitment abstract.Point
// 	// V_hat is the aggregated commit (our own + the children's)
// 	aggregateCommitment abstract.Point
// 	// challenge holds the challenge for this round
// 	challenge abstract.Scalar
// 	// response is our own computed response
// 	response abstract.Scalar
// 	// aggregateResponses is the aggregated response from the children + our own
// 	aggregateResponse abstract.Scalar
// }

// NewCosi returns a new Cosi struct given the suite + longterm secret.
// func NewCosi(suite abstract.Suite, private abstract.Scalar) *Cosi {
// 	return &Cosi{
// 		suite:   suite,
// 		private: private,
// 	}
// }

// Announcement holds only the timestamp for that round
type Announcement struct {
	Timestamp int64
}

//Commitment sends it's own commit Vi and the aggregate children's commit V^i
type Commitment struct {
	Commitment     kyber.Point
	ChildrenCommit kyber.Point
}

// Challenge is the Hash of V^0 || S, where S is the Timestamp
// and the message
type Challenge struct {
	Challenge kyber.Scalar
}

// Response holds the actual node's response ri and the
// aggregate response r^i
type Response struct {
	Response     kyber.Scalar
	ChildrenResp kyber.Scalar
}

// Signature is the final message out of the Cosi-protocol. It can
// be used together with the message and the aggregate public key
// to verify that it's valid.
type Signature struct {
	Challenge kyber.Scalar
	Response  kyber.Scalar
}

// Exception is what a node that does not want to sign should include when
// passing up a response
type Exception struct {
	Public     kyber.Point
	Commitment kyber.Point
}

// CreateAnnouncement creates an Announcement message with the timestamp set
// to the current time.
func (c *CoSi) CreateAnnouncement() *Announcement {
	now := time.Now().Unix()
	c.timestamp = now
	return &Announcement{now}
}

// Announce stores the timestamp and relays the message.
func (c *CoSi) Announce(in *Announcement) *Announcement {
	c.timestamp = in.Timestamp
	return in
}

// CreateCommitment creates the commitment out of the random secret and returns the message to pass up in the tree. This is typically called by the leaves.
func (c *CoSi) CreateCommitment() *Commitment {
	c.genCommit()
	return &Commitment{
		Commitment: c.commitment,
	}
}

// Commit creates the commitment / secret + aggregate children commitments from
// the children's messages.
func (c *CoSi) Commit(comms []*Commitment) *Commitment {
	// generate our own commit
	c.genCommit()

	// take the children commitment
	childVHat := c.suite.Point().Null()
	for _, com := range comms {
		// Add commitment of one child
		childVHat = childVHat.Add(childVHat, com.Commitment)
		// add commitment of it's children if there is one (i.e. if it is not a
		// leaf)
		if com.ChildrenCommit != nil {
			childVHat = childVHat.Add(childVHat, com.ChildrenCommit)
		}
	}
	// add our own commitment to the global V_hat
	c.aggregateCommitment = c.suite.Point().Add(childVHat, c.commitment)
	return &Commitment{
		ChildrenCommit: childVHat,
		Commitment:     c.commitment,
	}

}

// CreateChallenge creates the challenge out of the message it has been given.
// This is typically called by Root.
func (c *CoSi) CreateChallenge(msg []byte) (*Challenge, error) {
	// H( Commit || AggPublic || M)
	hash := sha512.New()
	_, err := c.aggregateCommitment.MarshalTo(hash)
	if err != nil {
		return nil, err
	}
	//raha err2 is added!
	_, err2 := c.mask.Aggregate().MarshalTo(hash)
	if err2 != nil {
		return nil, err
	}
	hash.Write(msg)
	chalBuff := hash.Sum(nil)
	// reducing the challenge
	c.challenge = c.suite.Scalar().SetBytes(chalBuff)
	c.message = msg
	//return c.challenge, nil
	return &Challenge{
		Challenge: c.challenge,
	}, err

	// if c.aggregateCommitment == nil {
	// 	return nil, errors.New("Empty aggregate-commitment")
	// }
	// pb, err := c.aggregateCommitment.MarshalBinary()
	// cipher := c.Suite.cipher(pb)
	// cipher.Message(nil, nil, msg)
	// c.challenge = c.suite.Scalar().Pick(cipher)
	// return &Challenge{
	// 	Challenge: c.challenge,
	// }, err
}

// Challenge keeps in memory the Challenge from the message.
func (c *CoSi) Challenge(ch *Challenge) *Challenge {
	c.challenge = ch.Challenge
	return ch
}

// CreateResponse is called by a leaf to create its own response from the
// challenge + commitment + private key. It returns the response to send up to
// the tree.
func (c *CoSi) CreateResponse() (*Response, error) {
	err := c.genResponse()
	return &Response{Response: c.response}, err
}

// Response generates the response from the commitment, challenge and the
// responses of its children.
func (c *CoSi) Response(responses []*Response) (*Response, error) {
	// create your own response
	if err := c.genResponse(); err != nil {
		return nil, err
	}
	aggregateResponse := c.suite.Scalar().Zero()
	for _, resp := range responses {
		// add responses of child
		aggregateResponse = aggregateResponse.Add(aggregateResponse, resp.Response)
		// add responses of it's children if there is one (i.e. if it is not a
		// leaf)
		if resp.ChildrenResp != nil {
			aggregateResponse = aggregateResponse.Add(aggregateResponse, resp.ChildrenResp)
		}
	}
	// Add our own
	c.aggregateResponse = c.suite.Scalar().Add(aggregateResponse, c.response)
	return &Response{
		Response:     c.response,
		ChildrenResp: aggregateResponse,
	}, nil

}

// GetAggregateResponse returns the aggregated response that this cosi has
// accumulated.
func (c *CoSi) GetAggregateResponse() kyber.Scalar {
	return c.aggregateResponse
}

// GetChallenge returns the challenge that were passed down to this cosi.
// func (c *CoSi) GetChallenge() kyber.Scalar {
// 	return c.challenge
// }

// GetCommitment returns the commitment generated by this CoSi (not aggregated).
// func (c *CoSi) GetCommitment() kyber.Point {
// 	return c.commitment
// }

// Signature returns a cosi Signature <=> a Schnorr signature. CAREFUL: you must
// call that when you are sure you have all the aggregated respones (i.e. the
// root of the tree if you use a tree).
func (c *CoSi) Signature() *Signature {
	return &Signature{
		c.challenge,
		c.aggregateResponse,
	}
}

// VerifyResponses verifies the response this CoSi has against the aggregated
// public key the tree is using.
// Check that: base**r_hat * X_hat**c == V_hat
// func (c *CoSi) VerifyResponses(aggregatedPublic abstract.Point) error {
// 	commitment := c.suite.Point()
// 	commitment = commitment.Add(commitment.Mul(nil, c.aggregateResponse), c.suite.Point().Mul(aggregatedPublic, c.challenge))
// 	// T is the recreated V_hat
// 	T := c.suite.Point().Null()
// 	T = T.Add(T, commitment)
// 	//  put that into exception mechanism later
// 	// T.Add(T, cosi.ExceptionV_hat)
// 	if !T.Equal(c.aggregateCommitment) {
// 		return errors.New("recreated commitment is not equal to one given")
// 	}
// 	return nil

// }

// genCommit generates a random secret vi and computes it's individual commit
// Vi = G^vi
func (c *CoSi) genCommit() {
	kp := key.NewKeyPair(onet.Suite)
	//raha: random is replaced by private .. fix later!
	c.random = kp.Private
	c.commitment = kp.Public
}

// genResponse creates the response
// func (c *CoSi) genResponse() error {
// 	if c.private == nil {
// 		return errors.New("No private key given in this cosi")
// 	}
// 	if c.random == nil {
// 		return errors.New("No random secret computed in this cosi")
// 	}
// 	if c.challenge == nil {
// 		return errors.New("No challenge computed in this cosi")
// 	}
// 	// resp = random - challenge * privatekey
// 	// i.e. ri = vi - c * xi
// 	resp := c.suite.Scalar().Mul(c.private, c.challenge)
// 	c.response = resp.Sub(c.random, resp)
// 	// no aggregation here
// 	c.aggregateResponse = c.response
// 	return nil
// }

// VerifySignature verifies if the challenge and the secret (from the response phase) form a
// correct signature for this message using the aggregated public key.
// func VerifySignature(suite abstract.Suite, msg []byte, public abstract.Point, challenge, secret abstract.Scalar) error {
// 	// recompute the challenge and check if it is the same
// 	commitment := suite.Point()
// 	commitment = commitment.Add(commitment.Mul(nil, secret), suite.Point().Mul(public, challenge))

// 	return verifyCommitment(suite, msg, commitment, challenge)

// }

func verifyCommitment(suite network.Suite, msg []byte, commitment kyber.Point, challenge kyber.Scalar) error {
	//pb, err := commitment.MarshalBinary()
	//if err != nil {
	//	return err
	//}
	// raha
	//cipher := suite.Cipher(pb)
	//cipher.Message(nil, nil, msg)
	// reconstructed challenge
	//reconstructed := suite.Scalar().Pick(cipher)
	//if !reconstructed.Equal(challenge) {
	//	return errors.New("Reconstructed challenge not equal to one given")
	//}
	return nil
}

// VerifySignatureWithException will verify the signature taking into account
// the exceptions given. An exception is the pubilc key + commitment of a peer that did not
// sign.
// NOTE: No exception mechanism for "before" commitment has been yet coded.
func VerifySignatureWithException(suite network.Suite, public kyber.Point, msg []byte, challenge, secret kyber.Scalar, exceptions []Exception) error {
	// first reduce the aggregate public key
	subPublic := suite.Point().Add(suite.Point().Null(), public)
	aggExCommit := suite.Point().Null()
	for _, ex := range exceptions {
		subPublic = subPublic.Sub(subPublic, ex.Public)
		aggExCommit = aggExCommit.Add(aggExCommit, ex.Commitment)
	}

	// recompute the challenge and check if it is the same
	commitment := suite.Point()
	//raha
	commitment = commitment.Add(commitment.Mul(secret, nil), suite.Point().Mul(challenge, public))
	// ADD the exceptions commitment here
	commitment = commitment.Add(commitment, aggExCommit)
	// check if it is ok
	return verifyCommitment(suite, msg, commitment, challenge)
}

// VerifyCosiSignatureWithException is a wrapper around VerifySignatureWithException
// but it takes a Signature instead of the Challenge/Response
func VerifyCosiSignatureWithException(suite network.Suite, public kyber.Point, msg []byte, signature *Signature, exceptions []Exception) error {
	return VerifySignatureWithException(suite, public, msg, signature.Challenge, signature.Response, exceptions)
}
