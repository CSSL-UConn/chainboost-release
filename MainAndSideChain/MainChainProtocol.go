package MainAndSideChain

// ----------------------------------------------------------------------------------------------
// -------------------------- main chain's protocol ---------------------------------------
// ----------------------------------------------------------------------------------------------

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"time"

	"github.com/chainBoostScale/ChainBoost/onet"
	"github.com/chainBoostScale/ChainBoost/onet/log"
	"github.com/chainBoostScale/ChainBoost/vrf"
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
	log.LLvl1(bz.Name(), " :the root node is filling the first block in first round: ", bz.MCRoundNumber)
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
		log.LLvl1("final result MC: leader TreeNodeID: ", msg.LeaderTreeNodeID.String(), "(root node) filled round number", bz.MCRoundNumber, "with empty block")
		bz.updateMainChainBCTransactionQueueCollect()
		bz.readBCAndSendtoOthers()
		log.LLvl1("new round is announced")
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
		log.LLvl1("final result MC: leader: ", bz.Tree().Search(msg.LeaderTreeNodeID).Name(), " is the round leader for round number ", bz.MCRoundNumber)
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
		log.LLvl1("new round is announced")
	} else {
		log.LLvl1("this round already has a leader!")
	}
}

func (bz *ChainBoost) MainChainCheckLeadership(msg MainChainNewRoundChan) error {
	//ToDoRaha:temp commented all verf calls
	var vrfOutput [64]byte
	toBeHashed := []byte(msg.Seed)
	// todoraha: we are skipping vrf for now

	proof, ok := bz.ECPrivateKey.ProveBytesGo(toBeHashed[:])
	if !ok {
		log.LLvl1("error while generating proof")
	}
	//---
	rand.Seed(int64(bz.TreeNodeInstance.Index()))
	seed := make([]byte, 32)
	rand.Read(seed)
	tempSeed := (*[32]byte)(seed[:32])
	//log.LLvl1("raha:debug:seed for the VRF is:", seed, "the tempSeed value is:", tempSeed)
	Pubkey, _ := vrf.VrfKeygenFromSeedGo(*tempSeed)
	//---
	_, vrfOutput = Pubkey.VerifyBytesGo(proof, toBeHashed[:])

	//ToDoRaha: a random 64 byte instead of vrf output
	//vrfOutput := make([]byte, 64)
	//rand.Read(vrfOutput)
	//log.LLvl1("Raha: the random gerenrated number is:", vrfOutput)
	var vrfoutputInt64 uint64
	buf := bytes.NewReader(vrfOutput[:])
	err := binary.Read(buf, binary.LittleEndian, &vrfoutputInt64)
	if err != nil {
		log.LLvl1("Panic Raised:\n\n")
		// panic(err)
		return xerrors.New("problem created after recieving msg from MainChainNewRoundChan:   " + err.Error())
	}
	log.LLvl1("VRF output:", vrfoutputInt64)
	//-----------
	//the criteria for selecting potential leaders
	if vrfoutputInt64 < msg.Power {
		// -----------
		log.LLvl1(bz.Name(), "I may be elected for round number ", bz.MCRoundNumber, "with power: ", msg.Power, "and vrf output of:", vrfoutputInt64)
		bz.SendTo(bz.Root(), &NewLeader{LeaderTreeNodeID: bz.TreeNode().ID, MCRoundNumber: bz.MCRoundNumber})
	}
	return nil
}
