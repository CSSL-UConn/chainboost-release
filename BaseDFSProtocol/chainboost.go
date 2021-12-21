package BaseDFSProtocol

// Package bdnproto implements the Boneh-Drijvers-Neven signature scheme
// to protect the aggregates against rogue public-key attacks.
// This is a modified version of blscosi/protocol which is now deprecated.
//package bdnproto

import (
	"errors"
	"time"

	onet "github.com/basedfs"
	"github.com/basedfs/blscosi/bdnproto"
	"github.com/basedfs/blscosi/protocol"
	"github.com/basedfs/log"
	"go.dedis.ch/kyber/v3/pairing"

	//"go.dedis.ch/kyber/v3/sign"
	"go.dedis.ch/kyber/v3/sign/bdn"
)

// BdnProtocolName is the name of the main protocol for the BDN signature scheme.
const BdnProtocolName = "bdnCoSiProto"

// BdnSubProtocolName is the name of the subprotocol for the BDN signature scheme.
const BdnSubProtocolName = "bdnSubCosiProto"

// GlobalRegisterBdnProtocols registers both protocol to the global register.
//func GlobalRegisterBdnProtocols() {
func init() {
	onet.GlobalProtocolRegister(BdnProtocolName, NewBdnProtocol)
	onet.GlobalProtocolRegister(BdnSubProtocolName, NewSubBdnProtocol)
}

//NewBdnProtocol is used to register the protocol with an always-true verification.
func NewBdnProtocol(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
	vf := func(a, b []byte) bool { return true }
	return NewBdnCosi(n, vf, BdnSubProtocolName, pairing.NewSuiteBn256())
}

// NewBdnCosi makes a protocol instance for the BDN CoSi protocol.
func NewBdnCosi(n *onet.TreeNodeInstance, vf protocol.VerificationFn, subProtocolName string, suite *pairing.SuiteBn256) (onet.ProtocolInstance, error) {
	c, err := protocol.NewBlsCosi(n, vf, subProtocolName, suite)
	if err != nil {
		return nil, err
	}

	mbc := c.(*protocol.BlsCosi)
	mbc.Sign = bdn.Sign
	mbc.Verify = bdn.Verify
	mbc.Aggregate = aggregate

	return mbc, nil
}

// NewSubBdnProtocol is the default sub-protocol function used for registration
// with an always-true verification.
func NewSubBdnProtocol(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
	vf := func(a, b []byte) bool { return true }
	return NewSubBdnCosi(n, vf, pairing.NewSuiteBn256())
}

// NewSubBdnCosi uses the default sub-protocol to make one compatible with
// the robust scheme.
func NewSubBdnCosi(n *onet.TreeNodeInstance, vf protocol.VerificationFn, suite *pairing.SuiteBn256) (onet.ProtocolInstance, error) {
	pi, err := protocol.NewSubBlsCosi(n, vf, suite)
	if err != nil {
		return nil, err
	}

	subCosi := pi.(*protocol.SubBlsCosi)
	subCosi.Sign = bdn.Sign
	subCosi.Verify = bdn.Verify
	subCosi.Aggregate = aggregate

	return subCosi, nil
}

// // aggregate uses the robust aggregate algorithm to aggregate the signatures.
// func aggregate(suite pairing.Suite, mask *sign.Mask, sigs [][]byte) ([]byte, error) {
// 	sig, err := bdn.AggregateSignatures(suite, sigs, mask)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return sig.MarshalBinary()
// }

// raha: brough from bdnproto_test ---------------------------------
// raha: made the function exportable (uppercase)
var testSuite = pairing.NewSuiteBn256()

const testServiceName = "TestServiceBdnCosi"

//var testServiceID onet.ServiceID

func RunBLSCoSiProtocol(bz *BaseDFS) error {
	var cosiProtocol *BlsCosi = bz.BlsCosi
	roster := bz.Roster()
	r1 := cosiProtocol.Roster()
	log.LLvl1(r1, "\n VS \n", roster)
	err := cosiProtocol.Start()
	if err != nil {
		return err
	}

	select {
	case sig := <-cosiProtocol.FinalSignature:
		pubs := roster.ServicePublics(testServiceName)
		return bdnproto.BdnSignature(sig).Verify(testSuite, cosiProtocol.Msg, pubs)
	case <-time.After(2 * time.Hour):
	}

	return errors.New("timeout")
}

// testService allows setting the pairing keys of the protocol.
type testService struct {
	// We need to embed the ServiceProcessor, so that incoming messages
	// are correctly handled.
	*onet.ServiceProcessor
}

// Starts a new service. No function needed.
func NewService(c *onet.Context) (onet.Service, error) {
	s := &testService{
		ServiceProcessor: onet.NewServiceProcessor(c),
	}
	return s, nil
}
