package platform

import (
	"math/rand"
	"sync"

	"github.com/BurntSushi/toml"
	MainAndSideChain "github.com/chainBoostScale/ChainBoost/MainAndSideChain"
	"github.com/chainBoostScale/ChainBoost/MainAndSideChain/BLSCoSi"
	"github.com/chainBoostScale/ChainBoost/onet"
	"github.com/chainBoostScale/ChainBoost/onet/log"
	"github.com/chainBoostScale/ChainBoost/onet/network"
	"github.com/chainBoostScale/ChainBoost/simulation/monitor"
	"github.com/chainBoostScale/ChainBoost/vrf"
	"go.dedis.ch/kyber/v3/pairing"
	"golang.org/x/xerrors"
)

type simulInit struct{}
type simulInitDone struct{}

// Simulate starts the server and will setup the protocol.
// Raha: in case of localhost simulation,
// in localhost.go simulate function will be called and its params are:
// Simulate(d.Suite, host, d.Simulation, "") which means
// host = 127.0.0." + strconv.Itoa(index+1)
// monitorAddress = ""
// simul = localhost.simulation

// raha: adding some other system-wide configurations
func Simulate(PercentageTxPay, MCRoundDuration, MainChainBlockSize, SideChainBlockSize, SectorNumber, NumberOfPayTXsUpperBound,
	SimulationRounds, SimulationSeed, NbrSubTrees, Threshold, SCRoundDuration, CommitteeWindow, MCRoundPerEpoch, SimState int,
	suite, serverAddress, simul, monitorAddress string) error {
	log.LLvl1("Raha: func Simulate is runnning!")
	scs, err := onet.LoadSimulationConfig(suite, ".", serverAddress)
	if err != nil {
		// We probably are not needed
		log.LLvl1(err, serverAddress)
		return err
	}
	//todoraha
	// if monitorAddress != "" {
	// 	log.LLvl1("raha: connecting to monitor: ", monitorAddress)
	// 	if err := monitor.ConnectSink(monitorAddress); err != nil {
	// 		log.Error("Couldn't connect monitor to sink:", err)
	// 		return xerrors.New("couldn't connect monitor to sink: " + err.Error())
	// 	}
	// }
	sims := make([]onet.Simulation, len(scs))
	simulInitID := network.RegisterMessage(simulInit{})
	simulInitDoneID := network.RegisterMessage(simulInitDone{})
	var rootSC *onet.SimulationConfig
	var rootSim onet.Simulation
	// having a waitgroup so the binary stops when all servers are closed
	var wgServer, wgSimulInit sync.WaitGroup
	var ready = make(chan bool)
	measureNodeBW := true
	measuresLock := sync.Mutex{}
	measures := make([]*monitor.CounterIOMeasure, len(scs))
	if len(scs) > 0 {
		cfg := &conf{}
		_, err := toml.Decode(scs[0].Config, cfg)
		if err != nil {
			return xerrors.New("error while decoding config: " + err.Error())
		}
		measureNodeBW = cfg.IndividualStats == ""
	}
	for i, sc := range scs {
		// Starting all servers for that server
		server := sc.Server

		if measureNodeBW {
			hostIndex, _ := sc.Roster.Search(sc.Server.ServerIdentity.ID)
			measures[i] = monitor.NewCounterIOMeasureWithHost("bandwidth", sc.Server, hostIndex)
		}

		log.Lvl4("Raha: in function simulate: ", serverAddress, "Starting server", server.ServerIdentity.Address)
		// Launch a server and notifies when it's done
		wgServer.Add(1)
		measure := measures[i]
		go func(c *onet.Server) {
			ready <- true
			defer wgServer.Done()
			log.Lvl2("raha: starting a server:", c.ServerIdentity.Address)
			c.Start()
			if measure != nil {
				measuresLock.Lock()
				measure.Record()
				measuresLock.Unlock()
			}
			log.LLvl1(serverAddress, "Simulation closed server", c.ServerIdentity)
		}(server)

		// wait to be sure the goroutine started
		<-ready
		//log.LLvl1("raha: does it get here for the buggy one?!")
		log.Lvl5("Raha: simul flag value is:", simul)
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
				defer func() {
					if measure != nil {
						measuresLock.Lock()
						// Remove the initialization of the simulation from this statistic
						measure.Reset()
						measuresLock.Unlock()
					}
				}()

				err = sim.Node(scTmp)
				log.ErrFatal(err)
				_, err := scTmp.Server.Send(env.ServerIdentity, &simulInitDone{})
				log.ErrFatal(err)
			}()
			return nil
		})
		server.RegisterProcessorFunc(simulInitDoneID, func(env *network.Envelope) error {
			wgSimulInit.Done()
			if measure != nil {
				measuresLock.Lock()
				// Reset the root bandwidth after the children sent the ACK.
				measure.Reset()
				measuresLock.Unlock()
			}
			return nil
		})
		if server.ServerIdentity.ID.Equal(sc.Tree.Root.ServerIdentity.ID) {
			log.LLvl1(serverAddress, "will start protocol")
			rootSim = sim
			rootSC = sc
		}
	}

	var simError error
	if rootSim != nil {
		// If this cothority has the root-server, it will start the simulation
		log.LLvl1("Starting protocol", simul, "on server", rootSC.Server.ServerIdentity.Address, "i.e. root node")
		// Raha: I want to see the list of nodes!
		log.Lvl5("Raha: Tree used in ChainBoost is", rootSC.Tree.Roster.List)
		//wait := true
		//for wait {
		// Raha: the protocols are created and instanciated here:

		// ---------------------------------------------------------------
		//     			---------- BLS CoSi protocol -------------
		// ---------------------------------------------------------------
		// initialization of committee members in side chain
		log.LLvl1(rootSC.Server.ServerIdentity.Address, ": initialization of side chain attributes")
		committeeNodes := rootSC.Tree.Roster.List[:CommitteeWindow-1]
		committeeNodes = append([]*network.ServerIdentity{rootSC.Tree.List()[CommitteeWindow].ServerIdentity}, committeeNodes...)
		committee := onet.NewRoster(committeeNodes)
		var x = *rootSC.Tree.List()[CommitteeWindow]
		x.RosterIndex = 0
		BlsCosiSubTrees, _ := BLSCoSi.NewBlsProtocolTree(onet.NewTree(committee, &x), NbrSubTrees)

		// todoraha: message should be initialized with main chain's genesis block

		// raha: BlsCosi protocol is created here => call to CreateProtocol() => call an empty Dispatch()
		log.Lvl1(rootSC.Server.ServerIdentity.Address, ": BlsCosi protocol is created")
		pi, err := rootSC.Overlay.CreateProtocol("bdnCoSiProto", BlsCosiSubTrees[0], onet.NilServiceID)
		if err != nil {
			return xerrors.New("couldn't create protocol: " + err.Error())
		}
		cosiProtocol := pi.(*BLSCoSi.BlsCosi)
		cosiProtocol.CreateProtocol = rootSC.Overlay.CreateProtocol
		/* Raha: it doesn't call any fuunction! just initializtion of methods that is going to be used later
		cosiProtocol.CreateProtocol = rootService.CreateProtocol //raha: it used to be initialized by this function call
		params from config file:
		cosiProtocol.Timeout = time.Duration(ProtocolTimeout) * time.Second
		*/
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
		// raha: ChainBoost protocol is created here => calling CreateProtocol() => calling Dispatch()
		log.Lvl1(rootSC.Server.ServerIdentity.Address, ": ChainBoost protocol is created")
		p, err := rootSC.Overlay.CreateProtocol("ChainBoost", rootSC.Tree, onet.NilServiceID)
		if err != nil {
			return xerrors.New("couldn't create protocol: " + err.Error())
		}
		ChainBoostProtocol := p.(*MainAndSideChain.ChainBoost)
		//ChainBoostProtocol.SetTimeout(time.Duration(TimeOut) * time.Second)
		// raha: finally passing our system-wide configurations to our protocol
		log.LLvl1(rootSC.Server.ServerIdentity.Address, ": initialization of ChainBoost protocol attributes")
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
		)
		// ---------------------------------------------------------------
		/* raha: initializing BLSCoSi protocol:
		this way, the roster that runs this protocol is initiated by the main roster,
		(the one that runs the ChainBoost protocol)
		i.e. cosiProtocol.TreeNodeInstance = ChainBoostProtocol.TreeNodeInstance
		*/
		log.Lvl1(rootSC.Server.ServerIdentity.Address, ": setting BLSCoSi prootocol as an ChainBoost protocol's attribute")
		ChainBoostProtocol.BlsCosi = cosiProtocol
		// --------------------------------------------------------------
		log.Lvl3("Starting nodes: List of nodes (full tree is): \n")
		for i, a := range rootSC.Tree.List() {
			log.Lvl5(i, " :", a.Name(), ": ", a.RosterIndex, "\n")
		}
		// ----
		log.LLvl1(rootSC.Server.ServerIdentity.Address, ": (root node) will start the protocol but will wait until all nodes join it")
		ChainBoostProtocol.JoinedWG.Add(len(rootSC.Tree.Roster.List))
		// root node is already joined :)
		ChainBoostProtocol.JoinedWG.Done()
		// ----
		// we must wait for all nodes to join the protocol (initialization/ even dispatch is removed but this seems to be necessary)
		if len(rootSC.Tree.Roster.List) > 100 {
			// ---
			ChainBoostProtocol.CalledWG.Add(len(rootSC.Tree.Roster.List) / 100)
			// ---
			for i := 0; i < len(rootSC.Tree.List())/100; i++ {
				go sendMsgToJoinChainBoostProtocol(i, ChainBoostProtocol, rootSC)
			}
			ChainBoostProtocol.CalledWG.Wait()
			log.Lvl1("Setting up nodes ... \n thia may take a few minutes, be patient ...")
		} else {
			for _, child := range rootSC.Tree.List() {
				if child != ChainBoostProtocol.TreeNode() {
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
						NbrSubTrees:     ChainBoostProtocol.NbrSubTrees,
						Threshold:       ChainBoostProtocol.Threshold,
						CommitteeWindow: ChainBoostProtocol.CommitteeWindow,
						SCRoundDuration: ChainBoostProtocol.SCRoundDuration,
						MCRoundPerEpoch: ChainBoostProtocol.MCRoundPerEpoch,
						SimState:        ChainBoostProtocol.SimState,
					})
					if err != nil {
						log.Lvl1(ChainBoostProtocol.Info(), "couldn't send hello to child", child.Name(), "with err:", err)
					}
				}
			}
		}

		// Raha: it is just the root node
		go func() {
			log.Lvl1(rootSC.Server.ServerIdentity.Address, ": root node is calling dispatch")
			err := ChainBoostProtocol.DispatchProtocol()
			if err != nil {
				log.Lvl1("protocol dispatch calling error: " + err.Error())
			}
		}()
		log.LLvl1(rootSC.Server.ServerIdentity.Address, ": root node is Starting the ChainBoost Protocol")
		ChainBoostProtocol.Start()
		// raha: bls cosi  start function is called inside ChainBoost protocol
		// ---------------------------------------------------------------
		// when it finishes  is when:
		// ToDoRaha
		log.LLvl1(rootSC.Server.ServerIdentity.Address, ": (root node) Back to simulation module: ChainBoostProtocol.Start() returned. waiting for DoneChainBoost channel")
		px := <-ChainBoostProtocol.DoneChainBoost
		log.LLvl1(rootSC.Server.ServerIdentity.Address, ": (root node) Back to simulation module. Final result is", px)
		//wait = false
		//}

		//ToDoRaha: clear this section
		//childrenWait.Record()
		// log.LLvl1("Broadcasting start, (Raha: I think its about having mutiple servers",
		// 	" which doesnt apply to when we are running a localhost simulation)")
		// syncWait := monitor.NewTimeMeasure("SimulSyncWait")
		// wgSimulInit.Add(len(rootSC.Tree.Roster.List))
		// for _, conode := range rootSC.Tree.Roster.List {
		// 	go func(si *network.ServerIdentity) {
		// 		_, err := rootSC.Server.Send(si, &simulInit{})
		// 		log.ErrFatal(err, "Couldn't send to conode:")
		// 	}(conode)
		// }
		// wgSimulInit.Wait()
		// syncWait.Record()
		// log.LLvl1("Starting new node", simul)
		// measureNet := monitor.NewCounterIOMeasure("bandwidth_root", rootSC.Server)
		// simError = rootSim.Run(rootSC)
		// measureNet.Record()

		log.LLvl1(rootSC.Server.ServerIdentity.Address, ": (root node) close all other nodes!")
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
		log.Lvl1("Raha: Starting the close all protocol to close all nodes by  the returned root node at the end of simulation")
		piC.Start()
	}
	//todoraha:
	log.LLvl1(serverAddress, scs[0].Server.ServerIdentity, "is waiting for all servers to close")
	wgServer.Wait()
	log.LLvl1(serverAddress, "has all servers closed")
	//if monitorAddress != "" {
	//	monitor.EndAndCleanup()
	//}

	// Give a chance to the simulation to stop the servers and clean up but returns the simulation error anyway.
	if simError != nil {
		return xerrors.New("error from simulation run: " + simError.Error())
	}
	log.LLvl1("Raha: func Simulate is returning")
	return nil
}

type conf struct {
	IndividualStats string
}

// raha added

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
		DoneChainBoost:    make(chan bool, 1),
		LeaderProposeChan: make(chan bool, 1),
		MCRoundNumber:     1,
		//CommitteeWindow:    10,
		SCRoundNumber:      1,
		FirstQueueWait:     0,
		SideChainQueueWait: 0,
		FirstSCQueueWait:   0,
		SecondQueueWait:    0,
		SimulationRounds:   0,
		BlsCosiStarted:     false,
		BlsCosi:            cosiProtocol,
		SimState:           1, // 1: just the main chain - 2: main chain plus side chain = chainBoost
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

	//ToDoRaha: what exactly does this sentence do?! do we need it?! nil array vs empty array?
	bz.CommitteeNodesTreeNodeID = make([]onet.TreeNodeID, bz.CommitteeWindow)
	bz.SummPoRTxs = make(map[int]int)

	// bls key pair for each node for VRF
	// ToDoraha: temp commented
	// do I need to bring this seed from config? check what it is used for?
	rand.Seed(int64(bz.TreeNodeInstance.Index()))
	seed := make([]byte, 32)
	rand.Read(seed)
	tempSeed := (*[32]byte)(seed[:32])
	log.Lvlf5("raha:debug:seed for the VRF is:", seed, "the tempSeed value is:", tempSeed)
	_, bz.ECPrivateKey = vrf.VrfKeygenFromSeedGo(*tempSeed)
	log.Lvlf5("raha:debug: the ECprivate Key is:", bz.ECPrivateKey)
	// --------------------------------------- blscosi -------------------
	if err := n.RegisterChannel(&bz.RtLSideChainNewRoundChan); err != nil {
		return bz, err
	}
	if err := n.RegisterChannel(&bz.LtRSideChainNewRoundChan); err != nil {
		return bz, err
	}
	return bz, nil
}
func sendMsgToJoinChainBoostProtocol(i int, ChainBoostProtocol *MainAndSideChain.ChainBoost, rootSC *onet.SimulationConfig) {
	for _, child := range rootSC.Tree.List()[i*1 : i*1+100] {
		if child != ChainBoostProtocol.TreeNode() {
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
				NbrSubTrees:     ChainBoostProtocol.NbrSubTrees,
				Threshold:       ChainBoostProtocol.Threshold,
				CommitteeWindow: ChainBoostProtocol.CommitteeWindow,
				SCRoundDuration: ChainBoostProtocol.SCRoundDuration,
				MCRoundPerEpoch: ChainBoostProtocol.MCRoundPerEpoch,
				SimState:        ChainBoostProtocol.SimState,
			})
			if err != nil {
				log.Lvl1(ChainBoostProtocol.Info(), "couldn't send hello to child", child.Name(), "with err:", err)
			}
		}
	}
	ChainBoostProtocol.CalledWG.Done()
}
