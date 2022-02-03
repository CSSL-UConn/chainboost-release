package platform

import (
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	onet "github.com/basedfs"
	"github.com/basedfs/BaseDFSProtocol"
	"github.com/basedfs/blscosi/protocol"
	"github.com/basedfs/log"
	"github.com/basedfs/network"
	"github.com/basedfs/simul/monitor"
	"github.com/basedfs/vrf"
	"go.dedis.ch/kyber/v3/pairing"

	//"go.dedis.ch/kyber/v3/sign/bls"
	"golang.org/x/xerrors"
)

type simulInit struct{}
type simulInitDone struct{}

// Simulate starts the server and will setup the protocol.
// Raha: in cas of localhost simulation,
// in localhost.go simulate function will be called and its params are:
// Simulate(d.Suite, host, d.Simulation, "") which means
// host = 127.0.0." + strconv.Itoa(index+1)
// monitorAddress = ""
// simul = localhost.simulation

// raha: adding some other system-wide configurations
func Simulate(PercentageTxPay, MCRoundDuration, BlockSize, SectorNumber, NumberOfPayTXsUpperBound, ProtocolTimeout, SimulationSeed, NbrSubTrees, Threshold, SCRoundDuration, CommitteeWindow, EpochCount int,
	suite, serverAddress, simul, monitorAddress string) error {
	scs, err := onet.LoadSimulationConfig(suite, ".", serverAddress)
	if err != nil {
		// We probably are not needed
		log.Lvl2(err, serverAddress)
		return nil
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
		log.Lvl4("Raha: Tree used in BaseDfs is", rootSC.Tree.Roster.List)
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
			BlsCosiSubTrees, _ := protocol.NewBlsProtocolTree(onet.NewTree(committee, &x), NbrSubTrees)
			// raha: BLSCoSi protocol
			// message should be initialized with main chain's genesis block
			// raha: BlsCosi protocol is created here => call to CreateProtocol() => call an empty Dispatch()
			pi, err := rootSC.Overlay.CreateProtocol("bdnCoSiProto", BlsCosiSubTrees[0], onet.NilServiceID)
			if err != nil {
				return xerrors.New("couldn't create protocol: " + err.Error())
			}
			cosiProtocol := pi.(*BaseDFSProtocol.BlsCosi)
			cosiProtocol.CreateProtocol = rootSC.Overlay.CreateProtocol // Raha: it doesn't call any fuunction! just initializtion of methods that is going to be used later
			//cosiProtocol.CreateProtocol = rootService.CreateProtocol //raha: it used to be initialized by this function call
			// params from config file:
			cosiProtocol.Timeout = time.Duration(ProtocolTimeout) * time.Second
			cosiProtocol.Threshold = Threshold
			if NbrSubTrees > 0 {
				err := cosiProtocol.SetNbrSubTree(NbrSubTrees)
				if err != nil {
					return err
				}
			}
			// ---------------------------------------------------------------
			//              ------   BaseDFS protocol  ------
			// ---------------------------------------------------------------
			// raha: BaseDfs protocol is created here => calling CreateProtocol() => calling Dispatch()
			p, err := rootSC.Overlay.CreateProtocol("BaseDFS", rootSC.Tree, onet.NilServiceID)
			if err != nil {
				return xerrors.New("couldn't create protocol: " + err.Error())
			}
			basedfsprotocol := p.(*BaseDFSProtocol.BaseDFS)
			//basedfsprotocol.SetTimeout(time.Duration(TimeOut) * time.Second)
			// raha: finally passing our system-wide configurations to our protocol
			basedfsprotocol.PercentageTxPay = PercentageTxPay
			basedfsprotocol.MCRoundDuration = MCRoundDuration
			basedfsprotocol.BlockSize = BlockSize
			basedfsprotocol.SectorNumber = SectorNumber
			basedfsprotocol.NumberOfPayTXsUpperBound = NumberOfPayTXsUpperBound
			basedfsprotocol.ProtocolTimeout = time.Duration(ProtocolTimeout) * time.Second
			basedfsprotocol.SimulationSeed = SimulationSeed
			basedfsprotocol.NbrSubTrees = NbrSubTrees
			basedfsprotocol.Threshold = Threshold
			basedfsprotocol.SCRoundDuration = SCRoundDuration
			basedfsprotocol.CommitteeWindow = CommitteeWindow
			basedfsprotocol.EpochCount = EpochCount
			log.LLvl2("passing our system-wide configurations to the protocol",
				"\n  PercentageTxPay: ", PercentageTxPay,
				"\n  MCRoundDuration: ", MCRoundDuration,
				"\n BlockSize: ", BlockSize,
				"\n SectorNumber: ", SectorNumber,
				"\n NumberOfPayTXsUpperBound: ", NumberOfPayTXsUpperBound,
				"\n timeout: ", ProtocolTimeout,
				"\n SimulationSeed of: ", SimulationSeed,
				"\n nbrSubTrees of: ", NbrSubTrees,
				"\n  threshold of: ", Threshold,
				"\n SCRoundDuration: ", SCRoundDuration,
				"\n CommitteeWindow: ", CommitteeWindow,
				"\n EpochCount: ", EpochCount,
			)
			// ---------------------------------------------------------------
			// raha: BLSCoSi protocol
			// raha: added: this way, the roster that runs this protocol is  initiated by the main roster, the one that runs the basedfs protocol
			// cosiProtocol.TreeNodeInstance = basedfsprotocol.TreeNodeInstance
			basedfsprotocol.BlsCosi = cosiProtocol
			// ---------------------------------------------------------------
			/* Raha: note that in overlay.go the CreateProtocol function will call the Dispatch() function by creating a go routine
			that's why I call it here in a go routine too.
			ToDO: But I should check how this part will be doing when testing on multiple servers
			raha: should be a single dispatch assigned for each node?! yes, it is in the basedfs start ..
			here, we call the DispatchProtocol function which handles messages in baseDfs protocol + the finalSignature message in BlsCosi protocol
			other messages communicated in BlsCosi protocol are handled by
			func (p *SubBlsCosi) Dispatch() which is called when the startSubProtocol in Blscosi.go,
			create subprotocols => hence calls func (p *SubBlsCosi) Dispatch() */
			// ---------------------------------------------------------------
			for _, child := range rootSC.Tree.List() {
				if child != basedfsprotocol.TreeNode() {
					err := basedfsprotocol.SendTo(child, &BaseDFSProtocol.HelloBaseDFS{
						ProtocolTimeout:          basedfsprotocol.ProtocolTimeout,
						PercentageTxPay:          basedfsprotocol.PercentageTxPay,
						MCRoundDuration:          basedfsprotocol.MCRoundDuration,
						BlockSize:                basedfsprotocol.BlockSize,
						SectorNumber:             basedfsprotocol.SectorNumber,
						NumberOfPayTXsUpperBound: basedfsprotocol.NumberOfPayTXsUpperBound,
						SimulationSeed:           basedfsprotocol.SimulationSeed,
						// --------------------- bls cosi ------------------------
						NbrSubTrees:     basedfsprotocol.NbrSubTrees,
						Threshold:       basedfsprotocol.Threshold,
						CommitteeWindow: basedfsprotocol.CommitteeWindow,
						SCRoundDuration: basedfsprotocol.SCRoundDuration,
						EpochCount:      basedfsprotocol.EpochCount,
					})
					if err != nil {
						log.Lvl1(basedfsprotocol.Info(), "couldn't send hello to child", child.Name())
					}
				}
			}
			go func() {
				err := basedfsprotocol.DispatchProtocol()
				if err != nil {
					log.Lvl1("protocol dispatch calling error: " + err.Error())
				}
			}()
			basedfsprotocol.Start()
			// raha: bls cosi  start function is called inside basedfs protocol
			// ---------------------------------------------------------------
			// when it finishes  is when:
			// ToDo
			log.LLvl1("here")
			px := <-basedfsprotocol.DoneBaseDFS
			log.Lvl1("Back to simulation module. Final result is", px)
			wait = false
		}

		//ToDo: clear this section

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
	return nil
}

type conf struct {
	IndividualStats string
}

// raha added

func init() {
	//network.RegisterMessage(ProofOfRetTxChan{})
	//network.RegisterMessage(PreparedBlockChan{})
	network.RegisterMessage(BaseDFSProtocol.HelloBaseDFS{})
	network.RegisterMessage(BaseDFSProtocol.NewRound{})
	network.RegisterMessage(BaseDFSProtocol.NewLeader{})
	network.RegisterMessage(BaseDFSProtocol.LtRSideChainNewRound{})
	network.RegisterMessage(BaseDFSProtocol.RtLSideChainNewRound{})
	onet.GlobalProtocolRegister("BaseDFS", NewBaseDFSProtocol)
}

func NewBaseDFSProtocol(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {

	pi, err := n.Overlay.CreateProtocol("bdnCoSiProto", n.Tree(), onet.NilServiceID)
	if err != nil {
		log.LLvl1("couldn't create protocol: " + err.Error())
	}
	cosiProtocol := pi.(*BaseDFSProtocol.BlsCosi)
	cosiProtocol.CreateProtocol = n.Overlay.CreateProtocol
	cosiProtocol.TreeNodeInstance = n
	//cosiProtocol.Timeout = BaseDFSProtocol.DefaultTimeout
	//cosiProtocol.SubleaderFailures = BaseDFSProtocol.DefaultSubleaderFailures
	//cosiProtocol.Threshold = BaseDFSProtocol.DefaultThreshold(0)
	//cosiProtocol.Sign = bls.Sign
	//cosiProtocol.Verify = bls.Verify
	//cosiProtocol.Aggregate = BaseDFSProtocol.Aggregate
	//cosiProtocol.VerificationFn = func(a, b []byte) bool { return true }
	//cosiProtocol.SubProtocolName = "blsSubCoSiProtoDefault"
	//cosiProtocol.Suite = pairing.NewSuiteBn256()
	//cosiProtocol.BlockType = BaseDFSProtocol.DefaultBlockType()

	bz := &BaseDFSProtocol.BaseDFS{
		TreeNodeInstance:  n,
		Suite:             pairing.NewSuiteBn256(),
		DoneBaseDFS:       make(chan bool, 1),
		LeaderProposeChan: make(chan bool, 1),
		MCRoundNumber:     1,
		SCRoundNumber:     1,
		HasLeader:         false,
		FirstQueueWait:    0,
		SecondQueueWait:   0,
		ProtocolTimeout:   0,
		BlsCosiStarted:    false,
		BlsCosi:           cosiProtocol,
	}

	if err := n.RegisterChannel(&bz.HelloChan); err != nil {
		return bz, err
	}
	if err := n.RegisterChannel(&bz.NewRoundChan); err != nil {
		return bz, err
	}
	if err := n.RegisterChannel(&bz.NewLeaderChan); err != nil {
		return bz, err
	}

	// if err := n.RegisterChannelLength(&bz.NewLeaderChan, len(bz.Tree().List())); err != nil {
	// 	log.Error("Couldn't register channel:    ", err)
	// }

	//ToDo: what exactly does this sentence do?! do we need it?!
	bz.CommitteeNodesTreeNodeID = make([]onet.TreeNodeID, bz.CommitteeWindow)

	// bls key pair for each node for VRF
	_, bz.ECPrivateKey = vrf.VrfKeygen()
	// --------------------------------------- blscosi -------------------
	if err := n.RegisterChannel(&bz.RtLSideChainNewRoundChan); err != nil {
		return bz, err
	}
	if err := n.RegisterChannel(&bz.LtRSideChainNewRoundChan); err != nil {
		return bz, err
	}
	return bz, nil
}

// package platform

// import (
// 	"sync"
// 	"time"

// 	"github.com/BurntSushi/toml"
// 	onet "github.com/basedfs"
// 	"github.com/basedfs/BaseDFSProtocol"
// 	"github.com/basedfs/log"
// 	"github.com/basedfs/network"
// 	"github.com/basedfs/simul/monitor"
// 	"golang.org/x/xerrors"
// )

// type simulInit struct{}
// type simulInitDone struct{}

// // Simulate starts the server and will setup the protocol.
// // Raha: in cas of localhost simulation,
// // in localhost.go simulate function will be called and its params are:
// // Simulate(d.Suite, host, d.Simulation, "") which means
// // host = 127.0.0." + strconv.Itoa(index+1)
// // monitorAddress = ""
// // simul = localhost.simulation

// // raha: adding some other system-wide configurations
// func Simulate(PercentageTxPay, MCRoundDuration, BlockSize, SectorNumber, NumberOfPayTXsUpperBound, ProtocolTimeout, SimulationSeed, NbrSubTrees, Threshold, SCRoundDuration, CommitteeWindow, EpochCount int,
// 	suite, serverAddress, simul, monitorAddress string) error {
// 	scs, err := onet.LoadSimulationConfig(suite, ".", serverAddress)
// 	if err != nil {
// 		// We probably are not needed
// 		log.Lvl2(err, serverAddress)
// 		return nil
// 	}
// 	if monitorAddress != "" {
// 		if err := monitor.ConnectSink(monitorAddress); err != nil {
// 			log.Error("Couldn't connect monitor to sink:", err)
// 			return xerrors.New("couldn't connect monitor to sink: " + err.Error())
// 		}
// 	}
// 	sims := make([]onet.Simulation, len(scs))
// 	simulInitID := network.RegisterMessage(simulInit{})
// 	simulInitDoneID := network.RegisterMessage(simulInitDone{})
// 	var rootSC *onet.SimulationConfig
// 	var rootSim onet.Simulation
// 	// having a waitgroup so the binary stops when all servers are closed
// 	var wgServer, wgSimulInit sync.WaitGroup
// 	var ready = make(chan bool)
// 	measureNodeBW := true
// 	measuresLock := sync.Mutex{}
// 	measures := make([]*monitor.CounterIOMeasure, len(scs))
// 	if len(scs) > 0 {
// 		cfg := &conf{}
// 		_, err := toml.Decode(scs[0].Config, cfg)
// 		if err != nil {
// 			return xerrors.New("error while decoding config: " + err.Error())
// 		}
// 		measureNodeBW = cfg.IndividualStats == ""
// 	}
// 	for i, sc := range scs {
// 		// Starting all servers for that server
// 		server := sc.Server

// 		if measureNodeBW {
// 			hostIndex, _ := sc.Roster.Search(sc.Server.ServerIdentity.ID)
// 			measures[i] = monitor.NewCounterIOMeasureWithHost("bandwidth", sc.Server, hostIndex)
// 		}

// 		log.Lvl3(serverAddress, "Starting server", server.ServerIdentity.Address)
// 		// Launch a server and notifies when it's done
// 		wgServer.Add(1)
// 		measure := measures[i]
// 		go func(c *onet.Server) {
// 			ready <- true
// 			defer wgServer.Done()
// 			c.Start()
// 			if measure != nil {
// 				measuresLock.Lock()
// 				measure.Record()
// 				measuresLock.Unlock()
// 			}
// 			log.Lvl3(serverAddress, "Simulation closed server", c.ServerIdentity)
// 		}(server)
// 		// wait to be sure the goroutine started
// 		<-ready

// 		sim, err := onet.NewSimulation(simul, sc.Config)
// 		if err != nil {
// 			return xerrors.New("couldn't create new simulation: " + err.Error())
// 		}
// 		sims[i] = sim
// 		// Need to store sc in a tmp-variable so it's correctly passed
// 		// to the Register-functions.
// 		scTmp := sc
// 		server.RegisterProcessorFunc(simulInitID, func(env *network.Envelope) error {
// 			// The node setup must be done in a goroutine to prevent the connection
// 			// from the root to this node to stale forever.
// 			go func() {
// 				defer func() {
// 					if measure != nil {
// 						measuresLock.Lock()
// 						// Remove the initialization of the simulation from this statistic
// 						measure.Reset()
// 						measuresLock.Unlock()
// 					}
// 				}()

// 				err = sim.Node(scTmp)
// 				log.ErrFatal(err)
// 				_, err := scTmp.Server.Send(env.ServerIdentity, &simulInitDone{})
// 				log.ErrFatal(err)
// 			}()
// 			return nil
// 		})
// 		server.RegisterProcessorFunc(simulInitDoneID, func(env *network.Envelope) error {
// 			wgSimulInit.Done()
// 			if measure != nil {
// 				measuresLock.Lock()
// 				// Reset the root bandwidth after the children sent the ACK.
// 				measure.Reset()
// 				measuresLock.Unlock()
// 			}
// 			return nil
// 		})
// 		if server.ServerIdentity.ID.Equal(sc.Tree.Root.ServerIdentity.ID) {
// 			log.Lvl2(serverAddress, "will start protocol")
// 			rootSim = sim
// 			rootSC = sc
// 		}
// 	}

// 	var simError error
// 	if rootSim != nil {
// 		// If this cothority has the root-server, it will start the simulation
// 		log.Lvl2("Starting protocol", simul, "on server", rootSC.Server.ServerIdentity.Address)
// 		// Raha: I want to see the list of nodes!
// 		log.Lvl4("Raha: Tree used in BaseDfs is", rootSC.Tree.Roster.List)
// 		// First count the number of available children
// 		//childrenWait := monitor.NewTimeMeasure("ChildrenWait")
// 		wait := true
// 		for wait {
// 			// Raha: the protocols are created and instanciated here:
// 			//p, err := rootSC.Overlay.CreateProtocol("Count", rootSC.Tree, onet.NilServiceID)
// 			//p, err := rootSC.Overlay.CreateProtocol("OpinionGathering", rootSC.Tree, onet.NilServiceID)

// 			// raha: BaseDfs protocol is created here => call to CreateProtocol() => call an empty Dispatch()
// 			p, err := rootSC.Overlay.CreateProtocol("BaseDFS", rootSC.Tree, onet.NilServiceID)
// 			if err != nil {
// 				return xerrors.New("couldn't create protocol: " + err.Error())
// 			}

// 			//proto := p.(*manage.ProtocolCount)
// 			//proto := p.(*manage.ProtocolOpinionGathering)
// 			// ---------------------------------------------------------------
// 			// raha: BLSCoSi protocol
// 			// raha: BlsCosi protocol is created here => call to CreateProtocol() => call an empty Dispatch()
// 			pi, err := rootSC.Overlay.CreateProtocol("bdnCoSiProto", rootSC.Tree, onet.NilServiceID)
// 			if err != nil {
// 				return xerrors.New("couldn't create protocol: " + err.Error())
// 			}
// 			// --------------------------------------------------------

// 			cosiProtocol := pi.(*BaseDFSProtocol.BlsCosi)
// 			cosiProtocol.CreateProtocol = rootSC.Overlay.CreateProtocol // Raha: it doesn't call any fuunction! just initializtion of methods that is going to be used later
// 			//cosiProtocol.CreateProtocol = rootService.CreateProtocol //raha: it used to be initialized by this function call
// 			// params from config file:
// 			cosiProtocol.Timeout = time.Duration(ProtocolTimeout) * time.Second
// 			cosiProtocol.Threshold = Threshold
// 			if NbrSubTrees > 0 {
// 				err := cosiProtocol.SetNbrSubTree(NbrSubTrees)
// 				if err != nil {
// 					return err
// 				}
// 			}
// 			// ---------------------------------------------------------------

// 			basedfsprotocol := p.(*BaseDFSProtocol.BaseDFS)
// 			//basedfsprotocol.SetTimeout(time.Duration(TimeOut) * time.Second)
// 			// raha: finally passing our system-wide configurations to our protocol
// 			basedfsprotocol.PercentageTxPay = PercentageTxPay
// 			basedfsprotocol.MCRoundDuration = MCRoundDuration
// 			basedfsprotocol.BlockSize = BlockSize
// 			basedfsprotocol.SectorNumber = SectorNumber
// 			basedfsprotocol.NumberOfPayTXsUpperBound = NumberOfPayTXsUpperBound
// 			basedfsprotocol.ProtocolTimeout = time.Duration(ProtocolTimeout) * time.Second
// 			basedfsprotocol.SimulationSeed = SimulationSeed
// 			basedfsprotocol.NbrSubTrees = NbrSubTrees
// 			basedfsprotocol.Threshold = Threshold
// 			basedfsprotocol.SCRoundDuration = SCRoundDuration
// 			basedfsprotocol.CommitteeWindow = CommitteeWindow
// 			basedfsprotocol.EpochCount = EpochCount
// 			log.LLvl2("passing our system-wide configurations to the protocol",
// 				"\n  PercentageTxPay: ", PercentageTxPay,
// 				"\n  MCRoundDuration: ", MCRoundDuration,
// 				"\n BlockSize: ", BlockSize,
// 				"\n SectorNumber: ", SectorNumber,
// 				"\n NumberOfPayTXsUpperBound: ", NumberOfPayTXsUpperBound,
// 				"\n timeout: ", ProtocolTimeout,
// 				"\n SimulationSeed of: ", SimulationSeed,
// 				"\n nbrSubTrees of: ", NbrSubTrees,
// 				"\n  threshold of: ", Threshold,
// 				"\n SCRoundDuration: ", SCRoundDuration,
// 				"\n CommitteeWindow: ", CommitteeWindow,
// 				"\n EpochCount: ", EpochCount,
// 			)
// 			// ---------------------------------------------------------------
// 			// raha: BLSCoSi protocol
// 			// raha: added: this way, the roster that runs this protocol is  initiated by the main roster, the one that runs the basedfs protocol
// 			// cosiProtocol.TreeNodeInstance = basedfsprotocol.TreeNodeInstance
// 			basedfsprotocol.BlsCosi = cosiProtocol
// 			// ---------------------------------------------------------------
// 			/* Raha: note that in overlay.go the CreateProtocol function will call the Dispatch() function by creating a go routine
// 			that's why I call it here in a go routine too.
// 			ToDO: But I should check how this part will be doing when testing on multiple servers
// 			raha: should be a single dispatch assigned for each node?!
// 			here, we call the DispatchProtocol function which handles messages in baseDfs protocol + the finalSignature message in BlsCosi protocol
// 			other messages communicated in BlsCosi protocol are handled by
// 			func (p *SubBlsCosi) Dispatch() which is called when the startSubProtocol in Blscosi.go,
// 			create subprotocols => hence calls func (p *SubBlsCosi) Dispatch() */
// 			// ---------------------------------------------------------------
// 			go func() {
// 				err := basedfsprotocol.DispatchProtocol()
// 				if err != nil {
// 					log.Lvl1("protocol dispatch calling error: " + err.Error())
// 				}
// 			}()
// 			basedfsprotocol.Start()
// 			// raha: bls cosi  start function is called inside basedfs start call

// 			// when it finishes  is when:
// 			// ToDo
// 			// ---------------------------------------------------------------
// 			px := <-basedfsprotocol.DoneBaseDFS
// 			log.Lvl1("Back to simulation module. Final result is", px)
// 			wait = false
// 		}

// 		//ToDo: clear this section

// 		// 	//childrenWait.Record()
// 		// 	log.Lvl2("Broadcasting start, (Raha: I think its about having mutiple servers",
// 		// 		" which doesnt apply to when we are running a localhost simulation)")
// 		// 	syncWait := monitor.NewTimeMeasure("SimulSyncWait")
// 		// 	wgSimulInit.Add(len(rootSC.Tree.Roster.List))
// 		// 	for _, conode := range rootSC.Tree.Roster.List {
// 		// 		go func(si *network.ServerIdentity) {
// 		// 			_, err := rootSC.Server.Send(si, &simulInit{})
// 		// 			log.ErrFatal(err, "Couldn't send to conode:")
// 		// 		}(conode)
// 		// 	}
// 		// 	wgSimulInit.Wait()
// 		// 	syncWait.Record()
// 		// 	log.Lvl1("Starting new node", simul)

// 		// 	measureNet := monitor.NewCounterIOMeasure("bandwidth_root", rootSC.Server)
// 		// 	simError = rootSim.Run(rootSC)
// 		// 	measureNet.Record()

// 		// Test if all ServerIdentities are used in the tree, else we'll run into
// 		// troubles with CloseAll
// 		//if !rootSC.Tree.UsesList() {
// 		//	log.Error("The tree doesn't use all ServerIdentities from the list!\n" +
// 		//		"This means that the CloseAll will fail and the experiment never ends!")
// 		//}

// 		// Recreate a tree out of the original roster, to be sure all nodes are included and
// 		// that the tree is easy to close.
// 		//closeTree := rootSC.Roster.GenerateBinaryTree()
// 		//pi, err := rootSC.Overlay.CreateProtocol("CloseAll", closeTree, onet.NilServiceID)
// 		//if err != nil {
// 		//	return xerrors.New("couldn't create closeAll protocol: " + err.Error())
// 		//}
// 		//pi.Start()
// 	}

// 	//log.Lvl3(serverAddress, scs[0].Server.ServerIdentity, "is waiting for all servers to close")
// 	//wgServer.Wait()
// 	//log.Lvl2(serverAddress, "has all servers closed")
// 	//if monitorAddress != "" {
// 	//	monitor.EndAndCleanup()
// 	//}

// 	// Give a chance to the simulation to stop the servers and clean up but returns the simulation error anyway.
// 	if simError != nil {
// 		return xerrors.New("error from simulation run: " + simError.Error())
// 	}
// 	return nil
// }

// type conf struct {
// 	IndividualStats string
// }
