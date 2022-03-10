package MainAndSideChain

import (
	"github.com/chainBoostScale/ChainBoost/MainAndSideChain/BLSCoSi"
	"github.com/chainBoostScale/ChainBoost/onet"
	"github.com/chainBoostScale/ChainBoost/onet/log"
	"github.com/chainBoostScale/ChainBoost/onet/network"
	"golang.org/x/xerrors"
)

/* ----------------------------------------- TYPES -------------------------------------------------
------------------------------------------------------------------------------------------------  */

// channel used by side chain's leader in each side chain's round
type RtLSideChainNewRound struct {
	SCRoundNumber            int
	CommitteeNodesTreeNodeID []onet.TreeNodeID
	blocksize                int
}
type RtLSideChainNewRoundChan struct {
	*onet.TreeNode
	RtLSideChainNewRound
}
type LtRSideChainNewRound struct { //fix this
	NewRound      bool
	SCRoundNumber int
	SCSig         BLSCoSi.BlsSignature
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
func (bz *ChainBoost) DispatchProtocol() error {

	running := true
	var err error

	for running {
		select {

		// --------------------------------------------------------
		// message recieved from BLSCoSi (SideChain):
		// ******* just the current side chain's "LEADER" recieves this msg
		// --------------------------------------------------------
		case sig := <-bz.BlsCosi.FinalSignature:
			if err := BLSCoSi.BdnSignature(sig).Verify(bz.BlsCosi.Suite, bz.BlsCosi.Msg, bz.BlsCosi.SubTrees[0].Roster.Publics()); err == nil {
				log.LLvl2("final result SC:", bz.Name(), " : ", bz.BlsCosi.BlockType, "with side chain's round number", bz.SCRoundNumber, "Confirmed in Side Chain")
				err := bz.SendTo(bz.Root(), &LtRSideChainNewRound{
					NewRound:      true,
					SCRoundNumber: bz.SCRoundNumber,
					SCSig:         sig,
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

//SideChainLeaderPreNewRound:
func (bz *ChainBoost) SideChainLeaderPreNewRound(msg RtLSideChainNewRoundChan) error {
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
	bz.BlsCosi.SubTrees, err = BLSCoSi.NewBlsProtocolTree(onet.NewTree(committeeRoster, &x), bz.NbrSubTrees)
	if err == nil {
		if bz.SCRoundNumber == 1 {
			log.Lvl1("final result SC: Next bls cosi tree is: ", bz.BlsCosi.SubTrees[0].Roster.List,
				" with ", bz.Name(), " as Root \n running BlsCosi round number", bz.SCRoundNumber)
		}
	} else {
		return xerrors.New("Problem in cosi protocol run:   " + err.Error())
	}
	// ---
	// from bc: update msg size with next block size on side chain
	s := make([]byte, msg.blocksize, msg.blocksize)
	bz.BlsCosi.Msg = append(bz.BlsCosi.Msg, s...) // Msg is the meta block
	// ----
	go func() error {
		bz.BlsCosi.Start()
		if err != nil {
			return xerrors.New("Problem in cosi protocol run:   " + err.Error())
		}
		return nil
	}()
	return xerrors.New("Problem in cosi protocol run: should not get here")
}

//
func (bz *ChainBoost) SideChainRootPostNewRound(msg LtRSideChainNewRoundChan) error {
	var err error
	bz.SCRoundNumber = msg.SCRoundNumber
	bz.SCSig = msg.SCSig
	var blocksize int
	//----
	bz.BCLock.Lock()
	defer bz.BCLock.Unlock()
	//----
	if bz.MCRoundDuration*bz.MCRoundPerEpoch/bz.SCRoundDuration == bz.SCRoundNumber {

		bz.BlsCosi.BlockType = "Summery Block" // just to know!
		// issueing a sync transaction from last submitted summery block to the main chain
		blocksize = bz.syncMainChainBCTransactionQueueCollect()
		//update the last row in round table with summery block's size
		// in this round in which a summery block will be generated, new transactions will be added to the queue but not taken
		bz.updateSideChainBCRound(msg.Name(), blocksize)
		bz.updateSideChainBCTransactionQueueCollect()
		// reset side chain round number
		bz.SCRoundNumber = 1 // in side chain round number zero the summery blocks are published in side chain
		// ------------- Epoch changed -----------
		// i.e. the current published block on side chain is summery block
		// change committee:
		log.LLvl1("final result SC: BlsCosi: the Summery Block was for epoch number: ", bz.MCRoundNumber/bz.MCRoundPerEpoch)
		// changing next side chain's leader for the next epoch rounds from the last miner in the main chain's window of miners
		bz.NextSideChainLeader = bz.CommitteeNodesTreeNodeID[0]
		// changing side chain's committee to last miners in the main chain's window of miners
		bz.CommitteeNodesTreeNodeID = bz.CommitteeNodesTreeNodeID[0:bz.CommitteeWindow]
		log.Lvl1("final result SC: BlsCosi: next side chain's epoch leader is: ", bz.Tree().Search(bz.NextSideChainLeader).Name())
		for i, a := range bz.CommitteeNodesTreeNodeID {
			log.Lvl2("final result SC: BlsCosi: next side chain's epoch committee number ", i, ":", bz.Tree().Search(a).Name())
		}

	} else {
		// next meta block on side chain blockchian is added by the root node
		bz.updateSideChainBCRound(msg.Name(), 0) // we dont use blocksize param bcz when we are generating meta block the block size is measured and added in the func: updateSideChainBCTransactionQueueTake

		bz.updateSideChainBCTransactionQueueCollect()
		blocksize = bz.updateSideChainBCTransactionQueueTake()
		bz.BlsCosi.BlockType = "Meta Block" // just to know!

		// increase side chain round number
		bz.SCRoundNumber = bz.SCRoundNumber + 1
	}
	// --------------------------------------------------------------------
	//triggering next side chain round leader to run next round of blscosi
	// --------------------------------------------------------------------
	err = bz.SendTo(bz.Tree().Search(bz.NextSideChainLeader), &RtLSideChainNewRound{
		SCRoundNumber:            bz.SCRoundNumber,
		CommitteeNodesTreeNodeID: bz.CommitteeNodesTreeNodeID,
		blocksize:                blocksize,
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
func (bz *ChainBoost) StartSideChainProtocol() {
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
func (bz *ChainBoost) UpdateSideChainCommittee(msg MainChainNewLeaderChan) {
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

// this code block was used in hello ChainBoost to start the side chain protocol -- its here to keep back-up in case we noticed an unresolved issue in side chain run
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
