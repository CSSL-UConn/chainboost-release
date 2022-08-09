package MainAndSideChain

// ----------------------------------------------------------------------------------------------
// -------------------------- main chain's protocol ---------------------------------------
// ----------------------------------------------------------------------------------------------

import (
	"bytes"
	"encoding/binary"
	"math"
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
	Seed        string
	Power       int
	MaxFileSize int
}
type MainChainNewRoundChan struct {
	*onet.TreeNode
	NewRound
}

/* ----------------------------------- FUNCTIONS -------------------------------------------------
------------------------------------------------------------------------------------------------  */

func (bz *ChainBoost) StartMainChainProtocol() {
	// the root node is filling the first block in first round
	log.LLvl1(bz.Name(), " :the root node is filling the first block in round number: ", bz.MCRoundNumber)
	// for the first round we have the root node set as a round leader, so  it is true! and he takes txs from the queue
	//----
	bz.BCLock.Lock()
	defer bz.BCLock.Unlock()
	//----
	//
	log.Lvl1("Raha Debug: wgMCRound.Done")
	bz.wgMCRound.Done()
	//
	bz.updateBCPowerRound(bz.TreeNode().Name(), true)
	bz.updateMainChainBCTransactionQueueCollect()
	bz.updateMainChainBCTransactionQueueTake()
	time.Sleep(time.Duration(bz.MCRoundDuration) * time.Second)
	bz.readBCAndSendtoOthers()
	log.Lvl1("new round is announced")
}
func (bz *ChainBoost) RootPreNewRound(msg MainChainNewLeaderChan) {
	takenTime := time.Now()
	// -----------------------------------------------------
	// rounds without a leader =>
	// in this case the leader info is filled with root node's info, transactions are going to be collected normally but
	// since the block is empty, no transaction is going to be taken from queues => leader = false
	// -----------------------------------------------------
	if msg.LeaderTreeNodeID == bz.TreeNode().ID && bz.MCRoundNumber != 1 && bz.MCLeader.HasLeader && msg.MCRoundNumber == bz.MCRoundNumber {
		//
		log.Lvl1("Raha Debug: wgMCRound.Done")
		log.Lvl1("Raha Debug:mc round number:", bz.MCRoundNumber)
		bz.wgMCRound.Done()
		//
		// -----------------------------------------------------
		// wait for MCDuration/SCduration number of go routines
		// to be Done (Done is triggered after finishing SideChainRootPostNewRound)
		if bz.SimState == 2 && bz.MCRoundNumber%bz.MCRoundPerEpoch == 0 {
			log.Lvl1("Raha Debug: wgSCRound.Wait")
			bz.wgSCRound.Wait()
			log.Lvl1("Raha Debug: wgSCRound.Wait: PASSED")
			// this equation result has to be int!
			if int(bz.MCRoundPerEpoch*(bz.MCRoundDuration/bz.SCRoundDuration)) != bz.MCRoundPerEpoch*(bz.MCRoundDuration/bz.SCRoundDuration) {
				log.LLvl1("Panic Raised:\n\n")
				panic("err in setting config params: {bz.MCRoundPerEpoch*(bz.MCRoundDuration / bz.SCRoundDuration)} has to be int")
			}
			// each epoch this number of sc rounds should be passed
			log.Lvl1("Raha Debug: wgSCRound.Add(", bz.MCRoundPerEpoch*(bz.MCRoundDuration/bz.SCRoundDuration), ")")
			bz.wgSCRound.Add(bz.MCRoundPerEpoch * (bz.MCRoundDuration / bz.SCRoundDuration))
		}
		// -----------------------------------------------------
		bz.BCLock.Lock()
		//----
		bz.updateBCPowerRound(bz.Tree().Search(msg.LeaderTreeNodeID).Name(), false)
		// in the case of a leader-less round
		log.Lvl1("Final result MC: leader TreeNodeID: ROOT NODE filled round number", bz.MCRoundNumber, "with empty block")
		bz.updateMainChainBCTransactionQueueCollect()
		bz.readBCAndSendtoOthers()
		log.Lvl1("new round is announced")
		//----
		bz.BCLock.Unlock()
		// -----------------------------------------------------
		// normal rounds with a leader => leader = true
		// -----------------------------------------------------
	} else if msg.MCRoundNumber == bz.MCRoundNumber && bz.MCRoundNumber != 1 && bz.MCLeader.HasLeader {
		log.Lvl1("Raha Debug: wgMCRound.Done")
		log.Lvl1("Raha Debug:mc round number:", bz.MCRoundNumber)
		bz.wgMCRound.Done()
		// -----------------------------------------------------
		// wait for MCDuration/SCduration number of go routines
		// to be Done (Done is triggered after finishing SideChainRootPostNewRound)
		if bz.SimState == 2 && bz.MCRoundNumber%bz.MCRoundPerEpoch == 0 {
			log.Lvl1("Raha Debug: wgSCRound.Wait")
			bz.wgSCRound.Wait()
			log.Lvl1("Raha Debug: wgSCRound.Wait: PASSED")
			// this equation result has to be int!
			if int(bz.MCRoundPerEpoch*(bz.MCRoundDuration/bz.SCRoundDuration)) != bz.MCRoundPerEpoch*(bz.MCRoundDuration/bz.SCRoundDuration) {
				log.LLvl1("Panic Raised:\n\n")
				panic("err in setting config params: {bz.MCRoundPerEpoch*(bz.MCRoundDuration / bz.SCRoundDuration)} has to be int")
			}
			// each epoch this number of sc rounds should be passed
			log.Lvl1("Raha Debug: wgSCRound.Add(", bz.MCRoundPerEpoch*(bz.MCRoundDuration/bz.SCRoundDuration), ")")
			bz.wgSCRound.Add(bz.MCRoundPerEpoch * (bz.MCRoundDuration / bz.SCRoundDuration))
		}
		// -----------------------------------------------------
		// ToDoRaha: later validate the leadership proof
		log.LLvl1("Final result MC: leader: ", bz.Tree().Search(msg.LeaderTreeNodeID).Name(), " is the round leader for round number ", bz.MCRoundNumber)
		// -----------------------------------------------
		// dynamically change the side chain's committee with last main chain's leader
		if bz.SimState == 2 { // i.e. if side chain running is set in simulation
			bz.UpdateSideChainCommittee(msg)
		}
		// -----------------------------------------------
		bz.BCLock.Lock()
		//----
		bz.updateBCPowerRound(bz.Tree().Search(msg.LeaderTreeNodeID).Name(), true)
		bz.updateMainChainBCTransactionQueueCollect()
		bz.updateMainChainBCTransactionQueueTake()
		// announce new round and give away required checkleadership info to nodes
		bz.readBCAndSendtoOthers()
		log.Lvl1("new round is announced")
		//----
		bz.BCLock.Unlock()
	} else if msg.MCRoundNumber == bz.MCRoundNumber {
		log.Lvl1("this round already has a leader!")
	}
	log.Lvl1("RootPreNewRound took:", time.Since(takenTime).String())
}
func (bz *ChainBoost) MainChainCheckLeadership(msg MainChainNewRoundChan) error {
	var vrfOutput [64]byte
	toBeHashed := []byte(msg.Seed)

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
	log.Lvlf5("VRF output:", vrfoutputInt64)
	//-----------
	vrfoutputInt := int(vrfoutputInt64) % msg.MaxFileSize
	vrfoutputInt = msg.MaxFileSize - int(math.Abs(float64(vrfoutputInt)))
	//the criteria for selecting potential leaders
	if vrfoutputInt < msg.Power {
		// -----------
		log.Lvl2(bz.Name(), "I may be elected for round number ", bz.MCRoundNumber, "with power: ", msg.Power, "and vrf output of:", vrfoutputInt)
		bz.SendTo(bz.Root(), &NewLeader{LeaderTreeNodeID: bz.TreeNode().ID, MCRoundNumber: bz.MCRoundNumber})
	}
	return nil
}
