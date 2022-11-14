package platform

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	MainAndSideChain "github.com/chainBoostScale/ChainBoost/MainAndSideChain"
	"github.com/chainBoostScale/ChainBoost/MainAndSideChain/BLSCoSi"
	"github.com/chainBoostScale/ChainBoost/onet"
	"github.com/chainBoostScale/ChainBoost/onet/log"
	"github.com/chainBoostScale/ChainBoost/onet/network"
	"github.com/chainBoostScale/ChainBoost/vrf"
	"go.dedis.ch/kyber/v3/pairing"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
)

type simulInit struct{}
type simulInitDone struct{}

var batchSize = 1000

// Simulate starts the server and will setup the protocol.
// adding some other system-wide configurations
func Simulate(PercentageTxPay, MCRoundDuration, MainChainBlockSize, SideChainBlockSize, SectorNumber, NumberOfPayTXsUpperBound,
	SimulationRounds, SimulationSeed, NbrSubTrees, Threshold, SCRoundDuration, CommitteeWindow, MCRoundPerEpoch, SimState, StoragePaymentEpoch int,
	suite, serverAddress, simul string) error {
	scs, err := onet.LoadSimulationConfig(suite, ".", serverAddress)
	if err != nil {
		// We probably are not needed
		log.LLvl1(err, serverAddress)
		return err
	}
	sims := make([]onet.Simulation, len(scs))
	simulInitID := network.RegisterMessage(simulInit{})
	simulInitDoneID := network.RegisterMessage(simulInitDone{})
	var rootSC *onet.SimulationConfig
	var rootSim onet.Simulation
	// having a waitgroup so the binary stops when all servers are closed
	var wgServer, wgSimulInit sync.WaitGroup
	var ready = make(chan bool)
	if len(scs) > 0 {
		cfg := &conf{}
		_, err := toml.Decode(scs[0].Config, cfg)
		if err != nil {
			return xerrors.New("error while decoding config: " + err.Error())
		}
	}
	for i, sc := range scs {
		// Starting all servers for that server
		server := sc.Server

		log.Lvl4("in function simulate: ", serverAddress, "Starting server", server.ServerIdentity.Address)
		// Launch a server and notifies when it's done
		wgServer.Add(1)
		go func(c *onet.Server) {
			ready <- true
			defer wgServer.Done()
			log.Lvl2(": starting a server:", c.ServerIdentity.Address)
			c.Start()
			log.LLvl1(serverAddress, "Simulation closed server", c.ServerIdentity)
		}(server)

		// wait to be sure the goroutine started
		<-ready
		log.Lvl5("simul flag value is:", simul)
		sim, err := onet.NewSimulation(simul, sc.Config)
		if err != nil {
			return xerrors.New("couldn't create new simulation: " + err.Error())
		}
		sims[i] = sim
		// Need to store sc in a tmp-variable so it's correctly passed
		// to the Register-functions.
		scTmp := sc
		server.RegisterProcessorFunc(simulInitID, func(env *network.Envelope) error {
			// The node setup must be done in a goroutine to prevent the connection
			// from the root to this node to stale forever.
			go func() {
				err = sim.Node(scTmp)
				log.ErrFatal(err)
				_, err := scTmp.Server.Send(env.ServerIdentity, &simulInitDone{})
				log.ErrFatal(err)
			}()
			return nil
		})

		server.RegisterProcessorFunc(simulInitDoneID, func(env *network.Envelope) error {
			wgSimulInit.Done()
			return nil
		})
		if server.ServerIdentity.ID.Equal(sc.Tree.Root.ServerIdentity.ID) {
			log.LLvl5(serverAddress, "will start protocol")
			rootSim = sim
			rootSC = sc
		}
	}

	var simError error
	if rootSim != nil {
		// If this cothority has the root-server, it will start the simulation
		// ---------------------------------------------------------------
		//              ---------- BLS CoSi protocol -------------
		// ---------------------------------------------------------------
		// initialization of committee members in side chain
		committeeNodes := rootSC.Tree.Roster.List[:CommitteeWindow-1]
		committeeNodes = append([]*network.ServerIdentity{rootSC.Tree.List()[CommitteeWindow].ServerIdentity}, committeeNodes...)
		committee := onet.NewRoster(committeeNodes)
		var x = *rootSC.Tree.List()[CommitteeWindow]
		x.RosterIndex = 0
		BlsCosiSubTrees, _ := BLSCoSi.NewBlsProtocolTree(onet.NewTree(committee, &x), NbrSubTrees)
		pi, err := rootSC.Overlay.CreateProtocol("bdnCoSiProto", BlsCosiSubTrees[0], onet.NilServiceID)
		if err != nil {
			return xerrors.New("couldn't create protocol: " + err.Error())
		}
		cosiProtocol := pi.(*BLSCoSi.BlsCosi)
		cosiProtocol.CreateProtocol = rootSC.Overlay.CreateProtocol
		/*
		   cosiProtocol.CreateProtocol = rootService.CreateProtocol //: it used to be initialized by this function call
		   params from config file: */
		cosiProtocol.Threshold = Threshold
		if NbrSubTrees > 0 {
			err := cosiProtocol.SetNbrSubTree(NbrSubTrees)
			if err != nil {
				return err
			}
		}
		// ---------------------------------------------------------------
		//              ------   ChainBoost protocol  ------
		// ---------------------------------------------------------------
		// calling CreateProtocol() => calling Dispatch()
		p, err := rootSC.Overlay.CreateProtocol("ChainBoost", rootSC.Tree, onet.NilServiceID)
		if err != nil {
			return xerrors.New("couldn't create protocol: " + err.Error())
		}
		ChainBoostProtocol := p.(*MainAndSideChain.ChainBoost)
		// passing our system-wide configurations to our protocol
		ChainBoostProtocol.PercentageTxPay = PercentageTxPay
		ChainBoostProtocol.MCRoundDuration = MCRoundDuration
		ChainBoostProtocol.MainChainBlockSize = MainChainBlockSize
		ChainBoostProtocol.SideChainBlockSize = SideChainBlockSize
		ChainBoostProtocol.SectorNumber = SectorNumber
		ChainBoostProtocol.NumberOfPayTXsUpperBound = NumberOfPayTXsUpperBound
		ChainBoostProtocol.SimulationRounds = SimulationRounds
		ChainBoostProtocol.SimulationSeed = SimulationSeed
		ChainBoostProtocol.NbrSubTrees = NbrSubTrees
		ChainBoostProtocol.Threshold = Threshold
		ChainBoostProtocol.SCRoundDuration = SCRoundDuration
		ChainBoostProtocol.CommitteeWindow = CommitteeWindow
		ChainBoostProtocol.MCRoundPerEpoch = MCRoundPerEpoch
		ChainBoostProtocol.SimState = SimState
		ChainBoostProtocol.StoragePaymentEpoch = StoragePaymentEpoch
		log.Lvl1("passing our system-wide configurations to the protocol",
			"\n  PercentageTxPay: ", PercentageTxPay,
			"\n  MCRoundDuration: ", MCRoundDuration,
			"\n MainChainBlockSize: ", MainChainBlockSize,
			"\n SideChainBlockSize: ", SideChainBlockSize,
			"\n SectorNumber: ", SectorNumber,
			"\n NumberOfPayTXsUpperBound: ", NumberOfPayTXsUpperBound,
			"\n SimulationRounds: ", SimulationRounds,
			"\n SimulationSeed of: ", SimulationSeed,
			"\n nbrSubTrees of: ", NbrSubTrees,
			"\n threshold of: ", Threshold,
			"\n SCRoundDuration: ", SCRoundDuration,
			"\n CommitteeWindow: ", CommitteeWindow,
			"\n MCRoundPerEpoch: ", MCRoundPerEpoch,
			"\n SimState: ", SimState,
			"\n StoragePaymentEpoch: ", StoragePaymentEpoch,
		)
		ChainBoostProtocol.BlsCosi = cosiProtocol
		// --------------------------------------------------------------
		ChainBoostProtocol.JoinedWG.Add(len(rootSC.Tree.Roster.List))
		ChainBoostProtocol.JoinedWG.Done()
		// ----

		ChainBoostProtocol.CalledWG.Add(len(rootSC.Tree.Roster.List))
		ChainBoostProtocol.CalledWG.Done()

		//Exit when first error occurs in a goroutine: https://stackoverflow.com/a/61518963
		//Pass arguments into errgroup: https://forum.golangbridge.org/t/pass-arguments-to-errgroup-within-a-loop/19999
		errs, _ := errgroup.WithContext(context.Background())
		for i := 0; i <= len(rootSC.Tree.List())/batchSize; i++ {
			id := i
			errs.Go(func() error {
				return sendMsgToJoinChainBoostProtocol(id, ChainBoostProtocol, rootSC)
			})
		}
		if err := errs.Wait(); err != nil {
			return xerrors.New("Error from simulation run: " + err.Error())
		}

		ChainBoostProtocol.CalledWG.Wait()
		log.LLvl1("Wait is passed which means all nodes have recieved the HelloChainBoost msg")

		go func() {
			err := ChainBoostProtocol.DispatchProtocol()
			if err != nil {
				log.Lvl1("protocol dispatch calling error: " + err.Error())
			}
		}()
		ChainBoostProtocol.Start()
		px := <-ChainBoostProtocol.DoneRootNode
		log.LLvl1(rootSC.Server.ServerIdentity.Address, ": Final result is", px)

		//childrenWait.Record()
		// log.LLvl1("Broadcasting start")
		// wgSimulInit.Add(len(rootSC.Tree.Roster.List))
		// for _, conode := range rootSC.Tree.Roster.List {
		//  go func(si *network.ServerIdentity) {
		//      _, err := rootSC.Server.Send(si, &simulInit{})
		//      log.ErrFatal(err, "Couldn't send to conode:")
		//  }(conode)
		// }
		// wgSimulInit.Wait()
		// syncWait.Record()
		// simError = rootSim.Run(rootSC)

		// --------------------------------------------------------------
		log.LLvl1(rootSC.Server.ServerIdentity.Address, ": close all other nodes!")

		for _, child := range rootSC.Tree.List() {
			err := ChainBoostProtocol.SendTo(child, &MainAndSideChain.SimulationDone{
				IsSimulationDone: true,
			})
			if err != nil {
				log.Lvl1(ChainBoostProtocol.Info(), "couldn't send ChainBoostDone msg to child", child.Name(), "with err:", err)
			}
		}

		// Test if all ServerIdentities are used in the tree, else we'll run into
		// troubles with CloseAll
		if !rootSC.Tree.UsesList() {
			log.Error("The tree doesn't use all ServerIdentities from the list!\n" +
				"This means that the CloseAll will fail and the experiment never ends!")
		}
		// Recreate a tree out of the original roster, to be sure all nodes are included and
		// that the tree is easy to close.
		closeTree := rootSC.Roster.GenerateBinaryTree()
		piC, err := rootSC.Overlay.CreateProtocol("CloseAll", closeTree, onet.NilServiceID)
		if err != nil {
			return xerrors.New("couldn't create closeAll protocol: " + err.Error())
		}
		log.Lvl1("close all nodes by  the returned root node at the end of simulation")
		piC.Start()
	}

	log.LLvl1(serverAddress, scs[0].Server.ServerIdentity, "is waiting for all servers to close")
	wgServer.Wait()
	log.LLvl1(serverAddress, "has all servers closed")

	// Give a chance to the simulation to stop the servers and clean up but returns the simulation error anyway.
	if simError != nil {
		return xerrors.New("error from simulation run: " + simError.Error())
	}
	log.LLvl1("Simulate is returning")
	return nil
}

type conf struct {
	IndividualStats string
}

//  added

func init() {
	network.RegisterMessage(MainAndSideChain.HelloChainBoost{})
	network.RegisterMessage(MainAndSideChain.Joined{})
	network.RegisterMessage(MainAndSideChain.NewRound{})
	network.RegisterMessage(MainAndSideChain.NewLeader{})
	network.RegisterMessage(MainAndSideChain.LtRSideChainNewRound{})
	network.RegisterMessage(MainAndSideChain.RtLSideChainNewRound{})
	onet.GlobalProtocolRegister("ChainBoost", NewChainBoostProtocol)
}

func NewChainBoostProtocol(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {

	pi, err := n.Overlay.CreateProtocol("bdnCoSiProto", n.Tree(), onet.NilServiceID)
	if err != nil {
		log.LLvl1("couldn't create protocol: " + err.Error())
	}
	cosiProtocol := pi.(*BLSCoSi.BlsCosi)
	cosiProtocol.CreateProtocol = n.Overlay.CreateProtocol
	cosiProtocol.TreeNodeInstance = n

	bz := &MainAndSideChain.ChainBoost{
		TreeNodeInstance:  n,
		Suite:             pairing.NewSuiteBn256(),
		DoneRootNode:      make(chan bool, 1),
		LeaderProposeChan: make(chan bool, 1),
		MCRoundNumber:     1,
		//CommitteeWindow:    10,
		SCRoundNumber:       1,
		FirstQueueWait:      0,
		SideChainQueueWait:  0,
		FirstSCQueueWait:    0,
		SecondQueueWait:     0,
		SimulationRounds:    0,
		BlsCosiStarted:      false,
		BlsCosi:             cosiProtocol,
		SimState:            1, // 1: just the main chain - 2: main chain plus side chain = chainBoost
		StoragePaymentEpoch: 0, // 0:  after contracts expires, n: settllement after n epoch
	}

	if err := n.RegisterChannel(&bz.HelloChan); err != nil {
		return bz, err
	}
	if err := n.RegisterChannelLength(&bz.JoinedWGChan, len(bz.Tree().List())); err != nil {
		log.Error("Couldn't register channel:    ", err)
	}
	if err := n.RegisterChannel(&bz.MainChainNewRoundChan); err != nil {
		return bz, err
	}
	if err := n.RegisterChannelLength(&bz.MainChainNewLeaderChan, len(bz.Tree().List())); err != nil {
		log.Error("Couldn't register channel:    ", err)
	}
	if err := n.RegisterChannel(&bz.ChainBoostDone); err != nil {
		return bz, err
	}

	//ToDoRaha: what exactly does this sentence do?! do we need it?! nil array vs empty array?
	bz.CommitteeNodesTreeNodeID = make([]onet.TreeNodeID, bz.CommitteeWindow)
	bz.SummPoRTxs = make(map[int]int)

	// bls key pair for each node for VRF
	// ToDo: temp commented
	// do I need to bring this seed from config? check what it is used for?
	rand.Seed(int64(bz.TreeNodeInstance.Index()))
	seed := make([]byte, 32)
	rand.Read(seed)
	tempSeed := (*[32]byte)(seed[:32])
	log.Lvlf5(":debug:seed for the VRF is:", seed, "the tempSeed value is:", tempSeed)
	_, bz.ECPrivateKey = vrf.VrfKeygenFromSeedGo(*tempSeed)
	log.Lvlf5(":debug: the ECprivate Key is:", bz.ECPrivateKey)
	// --------------------------------------- blscosi -------------------
	if err := n.RegisterChannel(&bz.RtLSideChainNewRoundChan); err != nil {
		return bz, err
	}
	if err := n.RegisterChannel(&bz.LtRSideChainNewRoundChan); err != nil {
		return bz, err
	}
	return bz, nil
}
func sendMsgToJoinChainBoostProtocol(i int, ChainBoostProtocol *MainAndSideChain.ChainBoost, rootSC *onet.SimulationConfig) error {
	var err error
	var end int
	begin := i * batchSize
	if len(rootSC.Tree.List()) <= i*batchSize+batchSize {
		end = len(rootSC.Tree.List())
	} else {
		end = i*batchSize + batchSize
	}
	for _, child := range rootSC.Tree.List()[begin:end] {
		if child != ChainBoostProtocol.TreeNode() {
			timeout := 20 * time.Second
			start := time.Now()
			for time.Since(start) < timeout {
				err := ChainBoostProtocol.SendTo(child, &MainAndSideChain.HelloChainBoost{
					SimulationRounds:         ChainBoostProtocol.SimulationRounds,
					PercentageTxPay:          ChainBoostProtocol.PercentageTxPay,
					MCRoundDuration:          ChainBoostProtocol.MCRoundDuration,
					MainChainBlockSize:       ChainBoostProtocol.MainChainBlockSize,
					SideChainBlockSize:       ChainBoostProtocol.SideChainBlockSize,
					SectorNumber:             ChainBoostProtocol.SectorNumber,
					NumberOfPayTXsUpperBound: ChainBoostProtocol.NumberOfPayTXsUpperBound,
					SimulationSeed:           ChainBoostProtocol.SimulationSeed,
					// --------------------- bls cosi ------------------------
					NbrSubTrees:         ChainBoostProtocol.NbrSubTrees,
					Threshold:           ChainBoostProtocol.Threshold,
					CommitteeWindow:     ChainBoostProtocol.CommitteeWindow,
					SCRoundDuration:     ChainBoostProtocol.SCRoundDuration,
					MCRoundPerEpoch:     ChainBoostProtocol.MCRoundPerEpoch,
					SimState:            ChainBoostProtocol.SimState,
					StoragePaymentEpoch: ChainBoostProtocol.StoragePaymentEpoch,
				})
				if err != nil {
					log.Lvl1(ChainBoostProtocol.Info(), "couldn't send HelloChainBoost msg to child", child.Name(), "with err:", err)
				} else {
					ChainBoostProtocol.CalledWG.Done()
					break
				}
			}

			if time.Since(start) >= timeout {
				err = fmt.Errorf("Attempt to send HelloChainBoost msg to child " + child.Name() + " timed out!")
				return err
			}
		}
	}

	return err
}
