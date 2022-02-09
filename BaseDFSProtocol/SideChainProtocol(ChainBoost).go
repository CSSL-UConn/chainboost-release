package BaseDFSProtocol

import (
	"time"

	onet "github.com/basedfs"
	"github.com/basedfs/blscosi/bdnproto"
	"github.com/basedfs/blscosi/protocol"
	"github.com/basedfs/log"
	"github.com/basedfs/network"
	"golang.org/x/xerrors"
)

/* ----------------------------------------- TYPES -------------------------------------------------
------------------------------------------------------------------------------------------------  */

// channel used by side chain's leader in each side chain's round
type RtLSideChainNewRound struct {
	SCRoundNumber            int
	CommitteeNodesTreeNodeID []onet.TreeNodeID
}
type RtLSideChainNewRoundChan struct {
	*onet.TreeNode
	RtLSideChainNewRound
}
type LtRSideChainNewRound struct { //fix this
	NewRound      bool
	SCRoundNumber int
}
type LtRSideChainNewRoundChan struct {
	*onet.TreeNode
	LtRSideChainNewRound
}

/* ----------------------------------- FUNCTIONS -------------------------------------------------
------------------------------------------------------------------------------------------------  */

/* ----------------------------------------------------------------------
 DispatchProtocol listen on the different channels in side chain protocol
------------------------------------------------------------------------ */
func (bz *BaseDFS) DispatchProtocol() error {

	running := true
	var err error

	for running {
		select {

		// --------------------------------------------------------
		// message recieved from BLSCoSi (SideChain):
		// ******* just the current side chain's leader on recieve this msg
		// --------------------------------------------------------
		case sig := <-bz.BlsCosi.FinalSignature:
			if err := bdnproto.BdnSignature(sig).Verify(bz.BlsCosi.Suite, bz.BlsCosi.Msg, bz.BlsCosi.SubTrees[0].Roster.Publics()); err == nil {
				log.LLvl2("final result SC:", bz.Name(), " : ", bz.BlsCosi.BlockType, "with side chain's round number", bz.SCRoundNumber, "Confirmed in Side Chain")
				err := bz.SendTo(bz.Root(), &LtRSideChainNewRound{
					NewRound:      true,
					SCRoundNumber: bz.SCRoundNumber,
				})
				if err != nil {
					return xerrors.New("can't send new round msg to root" + err.Error())
				}
			} else {
				return xerrors.New("The recieved final signature is not accepted:  " + err.Error())
			}
		}
	}
	return err
}

//
func (bz *BaseDFS) SideChainLeaderPreNewRound(msg RtLSideChainNewRoundChan) error {
	var err error
	bz.SCRoundNumber = msg.SCRoundNumber
	bz.BlsCosi.Msg = []byte{0xFF}
	// -----------------------------------------------
	// --- updating the next side chain's leader
	// -----------------------------------------------
	bz.CommitteeNodesTreeNodeID = msg.CommitteeNodesTreeNodeID
	var CommitteeNodesServerIdentity []*network.ServerIdentity
	if bz.SCRoundNumber == 1 {
		for _, a := range bz.CommitteeNodesTreeNodeID[0 : bz.CommitteeWindow-1] {
			CommitteeNodesServerIdentity = append(CommitteeNodesServerIdentity, bz.Tree().Search(a).ServerIdentity)
		}
		log.Lvl1("final result SC: ", bz.Name(), " is running next side chain's epoch with new committee")
		for i, a := range bz.CommitteeNodesTreeNodeID {
			log.Lvl2("final result SC: BlsCosi: next side chain's epoch committee number ", i, ":", bz.Tree().Search(a).Name())
		}
		//log.Lvl1("wait")
	} else {
		//log.LLvl1(len(bz.CommitteeNodesTreeNodeID))
		//log.LLvl1(bz.CommitteeNodesTreeNodeID[len(bz.CommitteeNodesTreeNodeID)-(bz.CommitteeWindow):])
		for _, a := range bz.CommitteeNodesTreeNodeID[len(bz.CommitteeNodesTreeNodeID)-(bz.CommitteeWindow):] {
			CommitteeNodesServerIdentity = append(CommitteeNodesServerIdentity, bz.Tree().Search(a).ServerIdentity)
		}
		log.Lvl1("final result SC: ", bz.Name(), " is running next side chain's epoch with the same committee members:")
		for i, a := range CommitteeNodesServerIdentity {
			log.Lvl2("final result SC: BlsCosi: next side chain's epoch committee number ", i, ":", a.Address)
		}
		//log.Lvl1("wait")
	}
	if bz.SCRoundNumber == 0 {
		bz.BlsCosi.BlockType = "Summery Block"
	} else {
		bz.BlsCosi.BlockType = "Meta Block"
	}
	committeeRoster := onet.NewRoster(CommitteeNodesServerIdentity)
	// --- root should have root index of 0 (this is based on what happens in gen_tree.go)
	var x = *bz.TreeNode()
	x.RosterIndex = 0
	// ---
	bz.BlsCosi.SubTrees, err = protocol.NewBlsProtocolTree(onet.NewTree(committeeRoster, &x), bz.NbrSubTrees)
	if err == nil {
		if bz.SCRoundNumber == 1 {
			log.Lvl1("final result SC: Next bls cosi tree is: ", bz.BlsCosi.SubTrees[0].Roster.List,
				" with ", bz.Name(), " as Root \n running BlsCosi round number", bz.SCRoundNumber)
		}
	} else {
		return xerrors.New("Problem in cosi protocol run:   " + err.Error())
	}
	// ----
	err = bz.BlsCosi.Start()
	if err != nil {
		return xerrors.New("Problem in cosi protocol run:   " + err.Error())
	}
	return nil
}

//
func (bz *BaseDFS) RootPostNewRound(msg LtRSideChainNewRoundChan) error {
	var err error
	bz.SCRoundNumber = msg.SCRoundNumber
	if bz.MCRoundDuration*bz.EpochCount/bz.SCRoundDuration == bz.SCRoundNumber+1 {
		// generate and propose summery block (bz.BlsCosi.blockType = "summeryblock")
		// reset side chain round number
		bz.SCRoundNumber = 0 // in side chain round number zero the summery blocks are published in side chain
		// from bc: update msg size with the "summery block"'s block size on side chain
		// call a function like collect and then take
	} else {
		// ------------- Epoch changed -----------
		if bz.SCRoundNumber == 0 { // i.e. the current published block on side chain is summery block
			// issue a sync transaction from last recent summery block to main chain
			// next meta block is going to be on top of last summery block (rather than last meta blocks!)
			log.LLvl1("final result SC: BlsCosi: the Summery Block was for epoch number: ", bz.MCRoundNumber/bz.EpochCount)
			//changing next side chain's leader for the next epoch rounds from the last miner in the main chain's window of miners
			bz.NextSideChainLeader = bz.CommitteeNodesTreeNodeID[0]
			// changing side chain's committee to last miners in the main chain's window of miners
			bz.CommitteeNodesTreeNodeID = bz.CommitteeNodesTreeNodeID[0:bz.CommitteeWindow]
			log.Lvl1("final result SC: BlsCosi: next side chain's epoch leader is: ", bz.Tree().Search(bz.NextSideChainLeader).Name())
			for i, a := range bz.CommitteeNodesTreeNodeID {
				log.Lvl2("final result SC: BlsCosi: next side chain's epoch committee number ", i, ":", bz.Tree().Search(a).Name())
			}
		}
		// increase side chain round number
		bz.SCRoundNumber = bz.SCRoundNumber + 1
		// from bc: update msg size with next "meta block"'s block size on side chain
		bz.BlsCosi.Msg = []byte{0xFF} // Msg is the meta block
	}
	// side chain round duration pause
	time.Sleep(time.Duration(bz.SCRoundDuration) * time.Second)
	// --------------------------------------------------------------------
	//triggering next side chain round leader to run next round of blscosi
	// --------------------------------------------------------------------
	err = bz.SendTo(bz.Tree().Search(bz.NextSideChainLeader), &RtLSideChainNewRound{
		SCRoundNumber:            bz.SCRoundNumber,
		CommitteeNodesTreeNodeID: bz.CommitteeNodesTreeNodeID,
	})
	if err != nil {
		log.Lvl2(bz.Name(), "can't send new side chain round msg to", bz.Tree().Search(bz.NextSideChainLeader).Name())
		return xerrors.New("can't send new side chain round msg to next leader" + err.Error())
	}
	return nil
}

// ----------------------------------------------------------------------------------------------
// ---------------- BLS CoSi protocol (Initialization for root node) --------
// ----------------------------------------------------------------------------------------------
func (bz *BaseDFS) StartSideChainProtocol() {
	var err error
	// -----------------------------------------------
	// --- initializing side chain's msg and side chain's committee roster index for the second run and the next runs
	// -----------------------------------------------
	for i := 0; i < bz.CommitteeWindow; i++ {
		bz.CommitteeNodesTreeNodeID = append(bz.CommitteeNodesTreeNodeID, bz.Tree().List()[i].ID)
	}
	bz.BlsCosi.Msg = []byte{0xFF}
	// -----------------------------------------------
	// --- initializing next side chain's leader
	// -----------------------------------------------
	bz.CommitteeNodesTreeNodeID = append([]onet.TreeNodeID{bz.Tree().List()[bz.CommitteeWindow+1].ID}, bz.CommitteeNodesTreeNodeID[:bz.CommitteeWindow-1]...)
	bz.NextSideChainLeader = bz.Tree().List()[bz.CommitteeWindow+1].ID
	// --------------------------------------------------------------------
	//triggering next side chain round leader to run next round of blscosi
	// --------------------------------------------------------------------
	err = bz.SendTo(bz.Tree().Search(bz.NextSideChainLeader), &RtLSideChainNewRound{
		SCRoundNumber:            bz.SCRoundNumber,
		CommitteeNodesTreeNodeID: bz.CommitteeNodesTreeNodeID,
	})
	if err != nil {
		log.Lvl2(bz.Name(), "can't send new side chain round msg to", bz.Tree().Search(bz.NextSideChainLeader).Name())
		panic("can't send new side chain round msg to the first leader")
	}
}

/* -----------------------------------------------
dynamically change the side chain's committee with last main chain's leader
the committee nodes is shifted by one and the new leader is added to be used for next epoch's side chain's committee
Note that: for now we are considering the last w distinct leaders in the committee which means
if a leader is selected multiple times during an epoch, he will nnot be added multiple times
-----------------------------------------------
--- updating the CommitteeNodesTreeNodeID
----------------------------------------------- */
func (bz *BaseDFS) UpdateSideChainCommittee(msg NewLeaderChan) {
	t := 0
	for _, a := range bz.CommitteeNodesTreeNodeID {
		if a != msg.LeaderTreeNodeID {
			t = t + 1
			continue
		} else {
			break
		}
	}
	if t != len(bz.CommitteeNodesTreeNodeID) {
		log.Lvl1("final result SC:", bz.Tree().Search(msg.LeaderTreeNodeID).Name(), " is already in the committee")
		for i, a := range bz.CommitteeNodesTreeNodeID {
			log.Lvl2("final result SC: BlsCosi committee queue: ", i, ":", bz.Tree().Search(a).Name())
		}
	} else {
		NextSideChainLeaderTreeNodeID := msg.LeaderTreeNodeID
		bz.CommitteeNodesTreeNodeID = append([]onet.TreeNodeID{NextSideChainLeaderTreeNodeID}, bz.CommitteeNodesTreeNodeID...)
		log.Lvl1("final result SC:", bz.Tree().Search(msg.LeaderTreeNodeID).Name(), "is added to side chain for the next epoch's committee")
		for i, a := range bz.CommitteeNodesTreeNodeID {
			log.Lvl2("final result SC: BlsCosi committee queue: ", i, ":", bz.Tree().Search(a).Name())
		}
	}
}

// this code block was used in hello basedfs to start the side chain protocol -- its here to keep back-up in case we noticed an unresolved issue in side chain run
// ----------------------------------------------------------------------------------------------
// ---------------- BLS CoSi protocol (running for the very first time) --------
// ----------------------------------------------------------------------------------------------
// this node has been set to start running blscosi in simulation level. (runsimul.go):
/* 	if bz.Tree().List()[bz.CommitteeWindow] == bz.TreeNode() {
	// -----------------------------------------------
	// --- initializing side chain's msg and side chain's committee roster index for the first run
	// -----------------------------------------------
	for i := 0; i < bz.CommitteeWindow; i++ {
		bz.CommitteeNodesTreeNodeID = append(bz.CommitteeNodesTreeNodeID, bz.Tree().List()[i].ID)
	}
	bz.BlsCosi.Msg = []byte{0xFF}

	bz.CommitteeNodesTreeNodeID = append([]onet.TreeNodeID{bz.Tree().List()[bz.CommitteeWindow].ID}, bz.CommitteeNodesTreeNodeID[:bz.CommitteeWindow-1]...)
	var CommitteeNodesServerIdentity []*network.ServerIdentity
	for _, a := range bz.CommitteeNodesTreeNodeID {
		CommitteeNodesServerIdentity = append(CommitteeNodesServerIdentity, bz.Tree().Search(a).ServerIdentity)
	}
	committeeRoster := onet.NewRoster(CommitteeNodesServerIdentity)
	// --- root should have root index of 0 (this is based on what happens in gen_tree.go)
	var x = *bz.Tree().List()[bz.CommitteeWindow]
	x.RosterIndex = 0
	// ---
	bz.BlsCosi.SubTrees, err = protocol.NewBlsProtocolTree(onet.NewTree(committeeRoster, &x), bz.NbrSubTrees)
	if err == nil {
		log.Lvl1("final result: BlsCosi: First bls cosi tree is: ", bz.BlsCosi.SubTrees[0].Roster.List,
			" with ", bz.BlsCosi.Name(), " as Root starting BlsCosi")
	} else {
		log.Lvl1("Raha: error: ", err)
	}
	// ------------------------------------------
	err = bz.BlsCosi.Start() // first leader in side chain is this node
	if err != nil {
		log.Lvl1(bz.Info(), "couldn't start side chain")
	}
} */
