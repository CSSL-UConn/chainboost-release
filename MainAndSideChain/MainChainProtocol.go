package MainAndSideChain

// ----------------------------------------------------------------------------------------------
// -------------------------- main chain's protocol ---------------------------------------
// ----------------------------------------------------------------------------------------------

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/basedfs/log"
	"github.com/basedfs/onet"
	"golang.org/x/xerrors"
)

/* ----------------------------------------- TYPES -------------------------------------------------
------------------------------------------------------------------------------------------------  */

type NewLeader struct {
	LeaderTreeNodeID onet.TreeNodeID
	MCRoundNumber    int
}
type NewLeaderChan struct {
	*onet.TreeNode
	NewLeader
}

type NewRound struct {
	Seed  string
	Power uint64
}
type NewRoundChan struct {
	*onet.TreeNode
	NewRound
}

/* ----------------------------------- FUNCTIONS -------------------------------------------------
------------------------------------------------------------------------------------------------  */

func (bz *BaseDFS) StartMainChainProtocol() {
	// the root node is filling the first block in first round
	log.Lvl2(bz.Name(), "Filling round number ", bz.MCRoundNumber)
	// for the first round we have the root node set as a round leader, so  it is true! and he takes txs from the queue
	bz.updateBCPowerRound(bz.TreeNode().Name(), true)
	bz.updateMainChainBCTransactionQueueCollect()
	bz.updateMainChainBCTransactionQueueTake()
	time.Sleep(time.Duration(bz.MCRoundDuration) * time.Second)
	bz.readBCAndSendtoOthers()
}
func (bz *BaseDFS) RootPreNewRound(msg NewLeaderChan) {
	// -----------------------------------------------------
	// rounds without a leader: in this case the leader info is filled with root node's info, transactions are going to be collected normally but
	// since the block is empty, no transaction is going to be taken from queues => leader = false
	// -----------------------------------------------------
	if msg.LeaderTreeNodeID == bz.TreeNode().ID && bz.MCRoundNumber != 1 && !bz.HasLeader && msg.MCRoundNumber == bz.MCRoundNumber {
		bz.HasLeader = true
		bz.updateBCPowerRound(bz.Tree().Search(msg.LeaderTreeNodeID).Name(), false)
		// in the case of a leader-less round
		log.Lvl1("final result MC: leader TreeNodeID: ", msg.LeaderTreeNodeID.String(), "(root node) filled round number", bz.MCRoundNumber, "with empty block")
		bz.updateMainChainBCTransactionQueueCollect()
		bz.readBCAndSendtoOthers()
		log.Lvl2("new round is announced")
		return
	}
	// -----------------------------------------------------
	// normal rounds with a leader => leader = true
	// -----------------------------------------------------
	if !bz.HasLeader && msg.MCRoundNumber == bz.MCRoundNumber {
		// ToDoRaha: first validate the leadership proof
		log.Lvl1("final result MC: leader: ", bz.Tree().Search(msg.LeaderTreeNodeID).Name(), " is the round leader for round number ", bz.MCRoundNumber)
		// -----------------------------------------------
		// dynamically change the side chain's committee with last main chain's leader
		if bz.SimState == 2 { // i.e. if side chain running is set in simulation
			bz.UpdateSideChainCommittee(msg)
		}
		// -----------------------------------------------
		bz.HasLeader = true
		bz.updateBCPowerRound(bz.Tree().Search(msg.LeaderTreeNodeID).Name(), true)
		bz.updateMainChainBCTransactionQueueCollect()
		bz.updateMainChainBCTransactionQueueTake()
	} else {
		log.Lvl2("this round already has a leader!")
	}
	//waiting for the time of round duration
	time.Sleep(time.Duration(bz.MCRoundDuration) * time.Second)
	// empty list of elected leaders  in this round
	for len(bz.NewLeaderChan) > 0 {
		<-bz.NewLeaderChan
	}
	// announce new round and give away required checkleadership info to nodes
	bz.readBCAndSendtoOthers()
	log.Lvl2("new round is announced")
}

//
func (bz *BaseDFS) MainChainCheckLeadership(msg NewRoundChan) error {
	var vrfOutput [64]byte
	toBeHashed := []byte(msg.Seed)
	proof, ok := bz.ECPrivateKey.ProveBytes(toBeHashed[:])
	if !ok {
		log.Lvl2("error while generating proof")
	}
	_, vrfOutput = bz.ECPrivateKey.Pubkey().VerifyBytes(proof, toBeHashed[:])
	var vrfoutputInt64 uint64
	buf := bytes.NewReader(vrfOutput[:])
	err := binary.Read(buf, binary.LittleEndian, &vrfoutputInt64)
	if err != nil {
		// log.Lvl2("Panic Raised:\n\n")
		// panic(err)
		return xerrors.New("problem creatde after recieving msg from NewRoundChan:   " + err.Error())
	}
	// -----------
	// the criteria for selecting the leader
	if vrfoutputInt64 < msg.Power {
		// -----------
		log.Lvl4(bz.Name(), "I may be elected for round number ", bz.MCRoundNumber)
		bz.SendTo(bz.Root(), &NewLeader{LeaderTreeNodeID: bz.TreeNode().ID, MCRoundNumber: bz.MCRoundNumber})
	}
	return nil
}