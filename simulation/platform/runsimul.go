package platform

import (
	"sync"

	"github.com/BurntSushi/toml"
	MainAndSideChain "github.com/chainBoostScale/ChainBoost/MainAndSideChain"
	"github.com/chainBoostScale/ChainBoost/MainAndSideChain/BLSCoSi"
	"github.com/chainBoostScale/ChainBoost/onet"
	"github.com/chainBoostScale/ChainBoost/onet/log"
	"github.com/chainBoostScale/ChainBoost/onet/network"
	"github.com/chainBoostScale/ChainBoost/simulation/monitor"

	//"github.com/chainBoostScale/ChainBoost/vrf"
	"go.dedis.ch/kyber/v3/pairing"

	//"go.dedis.ch/kyber/v3/sign/bls"
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
		log.LLvl1("Raha:1")
		return err
	}
	if monitorAddress != "" {
		if err := monitor.ConnectSink(monitorAddress); err != nil {
			log.Error("Couldn't connect monitor to sink:", err)
			return xerrors.New("couldn't connect monitor to sink: " + err.Error())
		}
	}
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

		log.Lvl3(serverAddress, "Starting server", server.ServerIdentity.Address)
		// Launch a server and notifies when it's done
		wgServer.Add(1)
		measure := measures[i]
		go func(c *onet.Server) {
			ready <- true
			defer wgServer.Done()
			c.Start()
			if measure != nil {
				measuresLock.Lock()
				measure.Record()
				measuresLock.Unlock()
			}
			log.Lvl3(serverAddress, "Simulation closed server", c.ServerIdentity)
		}(server)
		// wait to be sure the goroutine started
		<-ready

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
			log.LLvl1("Raha: is the error here?")
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
			log.LLvl1("Raha: is the error here?")
			return nil
		})
		if server.ServerIdentity.ID.Equal(sc.Tree.Root.ServerIdentity.ID) {
			log.Lvl2(serverAddress, "will start protocol")
			rootSim = sim
			rootSC = sc
		}
	}

	var simError error
	if rootSim != nil {
		// If this cothority has the root-server, it will start the simulation
		log.Lvl2("Starting protocol", simul, "on server", rootSC.Server.ServerIdentity.Address)
		// Raha: I want to see the list of nodes!
		log.Lvl4("Raha: Tree used in ChainBoost is", rootSC.Tree.Roster.List)
		wait := true
		for wait {
			// Raha: the protocols are created and instanciated here:

			// ---------------------------------------------------------------
			//     			---------- BLS CoSi protocol -------------
			// ---------------------------------------------------------------
			// initialization of committee members in side chain
			committeeNodes := rootSC.Tree.Roster.List[:CommitteeWindow-1]
			committeeNodes = append([]*network.ServerIdentity{rootSC.Tree.List()[CommitteeWindow].ServerIdentity}, committeeNodes...)
			committee := onet.NewRoster(committeeNodes)
			var x = *rootSC.Tree.List()[CommitteeWindow]
			x.RosterIndex = 0
			BlsCosiSubTrees, _ := BLSCoSi.NewBlsProtocolTree(onet.NewTree(committee, &x), NbrSubTrees)
			// raha: BLSCoSi protocol
			// message should be initialized with main chain's genesis block
			// raha: BlsCosi protocol is created here => call to CreateProtocol() => call an empty Dispatch()
			pi, err := rootSC.Overlay.CreateProtocol("bdnCoSiProto", BlsCosiSubTrees[0], onet.NilServiceID)
			if err != nil {
				return xerrors.New("couldn't create protocol: " + err.Error())
			}
			cosiProtocol := pi.(*BLSCoSi.BlsCosi)
			cosiProtocol.CreateProtocol = rootSC.Overlay.CreateProtocol // Raha: it doesn't call any fuunction! just initializtion of methods that is going to be used later
			//cosiProtocol.CreateProtocol = rootService.CreateProtocol //raha: it used to be initialized by this function call
			// params from config file:
			//cosiProtocol.Timeout = time.Duration(ProtocolTimeout) * time.Second
			cosiProtocol.Threshold = Threshold
			if NbrSubTrees > 0 {
				err := cosiProtocol.SetNbrSubTree(NbrSubTrees)
				if err != nil {
					log.LLvl1("Raha:2")
					return err
				}
			}
			// ---------------------------------------------------------------
			//              ------   ChainBoost protocol  ------
			// ---------------------------------------------------------------
			// raha: ChainBoost protocol is created here => calling CreateProtocol() => calling Dispatch()
			p, err := rootSC.Overlay.CreateProtocol("ChainBoost", rootSC.Tree, onet.NilServiceID)
			if err != nil {
				return xerrors.New("couldn't create protocol: " + err.Error())
			}
			ChainBoostProtocol := p.(*MainAndSideChain.ChainBoost)
			//ChainBoostProtocol.SetTimeout(time.Duration(TimeOut) * time.Second)
			// raha: finally passing our system-wide configurations to our protocol
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
			log.LLvl2("passing our system-wide configurations to the protocol",
				"\n  PercentageTxPay: ", PercentageTxPay,
				"\n  MCRoundDuration: ", MCRoundDuration,
				"\n MainChainBlockSize: ", MainChainBlockSize,
				"\n SideChainBlockSize: ", SideChainBlockSize,
				"\n SectorNumber: ", SectorNumber,
				"\n NumberOfPayTXsUpperBound: ", NumberOfPayTXsUpperBound,
				"\n SimulationRounds: ", SimulationRounds,
				"\n SimulationSeed of: ", SimulationSeed,
				"\n nbrSubTrees of: ", NbrSubTrees,
				"\n  threshold of: ", Threshold,
				"\n SCRoundDuration: ", SCRoundDuration,
				"\n CommitteeWindow: ", CommitteeWindow,
				"\n MCRoundPerEpoch: ", MCRoundPerEpoch,
				"\n SimState: ", SimState,
			)
			// ---------------------------------------------------------------
			// raha: BLSCoSi protocol
			// raha: added: this way, the roster that runs this protocol is  initiated by the main roster, the one that runs the ChainBoost protocol
			// cosiProtocol.TreeNodeInstance = ChainBoostProtocol.TreeNodeInstance
			ChainBoostProtocol.BlsCosi = cosiProtocol
			// ---------------------------------------------------------------
			/* Raha: note that in overlay.go the CreateProtocol function will call the Dispatch() function by creating a go routine
			that's why I call it here in a go routine too.
			ToDoRaha: But I should check how this part will be doing when testing on multiple servers
			raha: should be a single dispatch assigned for each node?! yes, it is in the ChainBoost start ..
			here, we call the DispatchProtocol function which handles messages in ChainBoost protocol + the finalSignature message in BlsCosi protocol
			other messages communicated in BlsCosi protocol are handled by
			func (p *SubBlsCosi) Dispatch() which is called when the startSubProtocol in Blscosi.go,
			create subprotocols => hence calls func (p *SubBlsCosi) Dispatch() */
			// ---------------------------------------------------------------
			log.LLvl1("Starting nodes: List of nodes (full tree is): \n")
			for i, a := range rootSC.Tree.List() {
				log.LLvl1(i, " :", a.Name(), ": ", a.RosterIndex, "\n")
			}
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
						log.Lvl1(ChainBoostProtocol.Info(), "couldn't send hello to child", child.Name())
					}
				}
			}
			// Raha: it is just the root  node
			go func() {
				err := ChainBoostProtocol.DispatchProtocol()
				if err != nil {
					log.Lvl1("protocol dispatch calling error: " + err.Error())
				}
			}()
			ChainBoostProtocol.Start()
			// raha: bls cosi  start function is called inside ChainBoost protocol
			// ---------------------------------------------------------------
			// when it finishes  is when:
			// ToDoRaha
			log.LLvl1("Back to simulation module: ChainBoostProtocol.Start() returned. waiting for DoneChainBoost channel .......... ")
			px := <-ChainBoostProtocol.DoneChainBoost
			log.Lvl1("Back to simulation module. Final result is", px)
			wait = false
		}

		//ToDoRaha: clear this section

		// 	//childrenWait.Record()
		// 	log.Lvl2("Broadcasting start, (Raha: I think its about having mutiple servers",
		// 		" which doesnt apply to when we are running a localhost simulation)")
		// 	syncWait := monitor.NewTimeMeasure("SimulSyncWait")
		// 	wgSimulInit.Add(len(rootSC.Tree.Roster.List))
		// 	for _, conode := range rootSC.Tree.Roster.List {
		// 		go func(si *network.ServerIdentity) {
		// 			_, err := rootSC.Server.Send(si, &simulInit{})
		// 			log.ErrFatal(err, "Couldn't send to conode:")
		// 		}(conode)
		// 	}
		// 	wgSimulInit.Wait()
		// 	syncWait.Record()
		// 	log.Lvl1("Starting new node", simul)

		// 	measureNet := monitor.NewCounterIOMeasure("bandwidth_root", rootSC.Server)
		// 	simError = rootSim.Run(rootSC)
		// 	measureNet.Record()

		// Test if all ServerIdentities are used in the tree, else we'll run into
		// troubles with CloseAll
		//if !rootSC.Tree.UsesList() {
		//	log.Error("The tree doesn't use all ServerIdentities from the list!\n" +
		//		"This means that the CloseAll will fail and the experiment never ends!")
		//}

		// Recreate a tree out of the original roster, to be sure all nodes are included and
		// that the tree is easy to close.
		//closeTree := rootSC.Roster.GenerateBinaryTree()
		//pi, err := rootSC.Overlay.CreateProtocol("CloseAll", closeTree, onet.NilServiceID)
		//if err != nil {
		//	return xerrors.New("couldn't create closeAll protocol: " + err.Error())
		//}
		//pi.Start()
	}

	//log.Lvl3(serverAddress, scs[0].Server.ServerIdentity, "is waiting for all servers to close")
	//wgServer.Wait()
	//log.Lvl2(serverAddress, "has all servers closed")
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
	//network.RegisterMessage(ProofOfRetTxChan{})
	//network.RegisterMessage(PreparedBlockChan{})
	network.RegisterMessage(MainAndSideChain.HelloChainBoost{})
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
	//cosiProtocol.Timeout = ChainBoostProtocol.DefaultTimeout
	//cosiProtocol.SubleaderFailures = ChainBoostProtocol.DefaultSubleaderFailures
	//cosiProtocol.Threshold = ChainBoostProtocol.DefaultThreshold(0)
	//cosiProtocol.Sign = bls.Sign
	//cosiProtocol.Verify = bls.Verify
	//cosiProtocol.Aggregate = ChainBoostProtocol.Aggregate
	//cosiProtocol.VerificationFn = func(a, b []byte) bool { return true }
	//cosiProtocol.SubProtocolName = "blsSubCoSiProtoDefault"
	//cosiProtocol.Suite = pairing.NewSuiteBn256()
	//cosiProtocol.BlockType = ChainBoostProtocol.DefaultBlockType()

	bz := &MainAndSideChain.ChainBoost{
		TreeNodeInstance:   n,
		Suite:              pairing.NewSuiteBn256(),
		DoneChainBoost:     make(chan bool, 1),
		LeaderProposeChan:  make(chan bool, 1),
		MCRoundNumber:      1,
		SCRoundNumber:      1,
		HasLeader:          false,
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
	if err := n.RegisterChannel(&bz.MainChainNewRoundChan); err != nil {
		return bz, err
	}
	//if err := n.RegisterChannel(&bz.MainChainNewLeaderChan); err != nil {
	//	return bz, err
	//}
	if err := n.RegisterChannelLength(&bz.MainChainNewLeaderChan, len(bz.Tree().List())); err != nil {
		log.Error("Couldn't register channel:    ", err)
	}

	//ToDoRaha: what exactly does this sentence do?! do we need it?!
	bz.CommitteeNodesTreeNodeID = make([]onet.TreeNodeID, bz.CommitteeWindow)
	bz.SummPoRTxs = make(map[int]int)
	// bls key pair for each node for VRF
	//raha todo temp comment
	//_, bz.ECPrivateKey = vrf.VrfKeygen()
	// --------------------------------------- blscosi -------------------
	if err := n.RegisterChannel(&bz.RtLSideChainNewRoundChan); err != nil {
		return bz, err
	}
	if err := n.RegisterChannel(&bz.LtRSideChainNewRoundChan); err != nil {
		return bz, err
	}
	return bz, nil
}
