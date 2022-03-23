package MainAndSideChain

// ----------------------------------------------------------------------------------------------
// -------------------------- main chain's protocol ---------------------------------------
// ----------------------------------------------------------------------------------------------

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/chainBoostScale/ChainBoost/onet"
	"github.com/chainBoostScale/ChainBoost/onet/log"
	"golang.org/x/xerrors"
)

/* ----------------------------------------- TYPES -------------------------------------------------
------------------------------------------------------------------------------------------------  */

type NewLeader struct {
	LeaderTreeNodeID onet.TreeNodeID
	MCRoundNumber    int
}
type MainChainNewLeaderChan struct {
	*onet.TreeNode
	NewLeader
}

type NewRound struct {
	Seed  string
	Power uint64
}
type MainChainNewRoundChan struct {
	*onet.TreeNode
	NewRound
}

/* ----------------------------------- FUNCTIONS -------------------------------------------------
------------------------------------------------------------------------------------------------  */

func (bz *ChainBoost) StartMainChainProtocol() {
	// the root node is filling the first block in first round
	log.Lvl2(bz.Name(), "Filling round number ", bz.MCRoundNumber)
	// for the first round we have the root node set as a round leader, so  it is true! and he takes txs from the queue
	//----
	bz.BCLock.Lock()
	defer bz.BCLock.Unlock()
	bz.MCPLock.Lock()
	defer bz.MCPLock.Unlock()
	//----
	bz.updateBCPowerRound(bz.TreeNode().Name(), true)
	bz.updateMainChainBCTransactionQueueCollect()
	bz.updateMainChainBCTransactionQueueTake()
	time.Sleep(time.Duration(bz.MCRoundDuration) * time.Second)
	bz.readBCAndSendtoOthers()
}
func (bz *ChainBoost) RootPreNewRound(msg MainChainNewLeaderChan) {
	// -----------------------------------------------------
	// rounds without a leader: in this case the leader info is filled with root node's info, transactions are going to be collected normally but
	// since the block is empty, no transaction is going to be taken from queues => leader = false
	// -----------------------------------------------------
	if msg.LeaderTreeNodeID == bz.TreeNode().ID && bz.MCRoundNumber != 1 && !bz.HasLeader && msg.MCRoundNumber == bz.MCRoundNumber {
		bz.HasLeader = true
		// wait for MCDuration/SCduration number of go routines
		// to be Done (Done is triggered after finishing SideChainRootPostNewRound)
		if bz.SimState == 2 {
			bz.wg.Wait()
			//
			x := int(bz.MCRoundDuration / bz.SCRoundDuration)
			bz.wg.Add(x)
		}
		//----
		bz.BCLock.Lock()
		defer bz.BCLock.Unlock()
		bz.MCPLock.Lock()
		defer bz.MCPLock.Unlock()
		//----
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
		bz.HasLeader = true
		// wait for MCDuration/SCduration number of go routines
		// to be Done (Done is triggered after finishing SideChainRootPostNewRound)
		if bz.SimState == 2 {
			bz.wg.Wait()
			//
			x := int(bz.MCRoundDuration / bz.SCRoundDuration)
			bz.wg.Add(x)
		}
		// ToDoRaha: later validate the leadership proof
		log.Lvl1("final result MC: leader: ", bz.Tree().Search(msg.LeaderTreeNodeID).Name(), " is the round leader for round number ", bz.MCRoundNumber)
		// -----------------------------------------------
		// dynamically change the side chain's committee with last main chain's leader
		if bz.SimState == 2 { // i.e. if side chain running is set in simulation
			bz.UpdateSideChainCommittee(msg)
		}
		// -----------------------------------------------
		//----
		bz.BCLock.Lock()
		defer bz.BCLock.Unlock()
		bz.MCPLock.Lock()
		defer bz.MCPLock.Unlock()
		//----
		bz.updateBCPowerRound(bz.Tree().Search(msg.LeaderTreeNodeID).Name(), true)
		bz.updateMainChainBCTransactionQueueCollect()
		bz.updateMainChainBCTransactionQueueTake()
		// announce new round and give away required checkleadership info to nodes
		bz.readBCAndSendtoOthers()
		log.Lvl2("new round is announced")
	} else {
		log.Lvl2("this round already has a leader!")
	}
}

//
func (bz *ChainBoost) MainChainCheckLeadership(msg MainChainNewRoundChan) error {
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
		return xerrors.New("problem creatde after recieving msg from MainChainNewRoundChan:   " + err.Error())
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
