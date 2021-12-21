package platform

import (
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	onet "github.com/basedfs"
	"github.com/basedfs/BaseDFSProtocol"
	"github.com/basedfs/log"
	"github.com/basedfs/network"
	"github.com/basedfs/simul/monitor"
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
func Simulate(PercentageTxPay, RoundDuration, BlockSize, SectorNumber, NumberOfPayTXsUpperBound, ProtocolTimeout, SimulationSeed, NbrSubTrees, Threshold int,
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
			log.Lvl2(serverAddress, "is root-node, will start protocol")
			rootSim = sim
			rootSC = sc
		}
	}

	var simError error
	if rootSim != nil {
		// If this cothority has the root-server, it will start the simulation
		log.Lvl2("Starting protocol", simul, "on server", rootSC.Server.ServerIdentity.Address)
		// Raha: I want to see the tree!
		log.Lvl5("Tree is", rootSC.Tree.Dump())
		// First count the number of available children
		//childrenWait := monitor.NewTimeMeasure("ChildrenWait")
		wait := true
		for wait {
			// Raha: the protocol is instanciated here:
			//p, err := rootSC.Overlay.CreateProtocol("Count", rootSC.Tree, onet.NilServiceID)
			//p, err := rootSC.Overlay.CreateProtocol("OpinionGathering", rootSC.Tree, onet.NilServiceID)
			p, err := rootSC.Overlay.CreateProtocol("BaseDFS", rootSC.Tree, onet.NilServiceID)
			if err != nil {
				return xerrors.New("couldn't create protocol: " + err.Error())
			}
			//proto := p.(*manage.ProtocolCount)
			//proto := p.(*manage.ProtocolOpinionGathering)
			proto := p.(*BaseDFSProtocol.BaseDFS)
			//proto.SetTimeout(time.Duration(TimeOut) * time.Second)
			// raha: finally passing our system-wide configurations to our protocol
			proto.PercentageTxPay = PercentageTxPay
			proto.RoundDuration = RoundDuration
			proto.BlockSize = BlockSize
			proto.SectorNumber = SectorNumber
			proto.NumberOfPayTXsUpperBound = NumberOfPayTXsUpperBound
			proto.ProtocolTimeout = time.Duration(ProtocolTimeout) * time.Second
			proto.SimulationSeed = SimulationSeed
			proto.NbrSubTrees = NbrSubTrees
			proto.Threshold = Threshold
			log.LLvl2("passing our system-wide configurations to the protocol",
				"\n  PercentageTxPay: ", PercentageTxPay,
				"\n  RoundDuration: ", RoundDuration,
				"\n BlockSize: ", BlockSize,
				"\n SectorNumber: ", SectorNumber,
				"\n NumberOfPayTXsUpperBound: ", NumberOfPayTXsUpperBound,
				"\n With timeout of: ", ProtocolTimeout,
				"\n and SimulationSeed of: ", SimulationSeed,
				"\n and nbrSubTrees of: ", NbrSubTrees,
				"\n and threshold of: ", Threshold,
			)
			// ---------------------------------------------------------------
			//raha: new protocol: BLSCoSi
			pi, err := rootSC.Overlay.CreateProtocol("bdnCoSiProto", rootSC.Tree, onet.NilServiceID)
			if err != nil {
				return xerrors.New("couldn't create protocol: " + err.Error())
			}

			cosiProtocol := pi.(*BaseDFSProtocol.BlsCosi)
			//cosiProtocol.CreateProtocol = rootSC.Overlay.CreateProtocol
			//cosiProtocol.CreateProtocol = rootService.CreateProtocol

			// message should be initialized with meta blocks
			cosiProtocol.Msg = []byte{0xFF}
			cosiProtocol.Timeout = time.Duration(ProtocolTimeout) * time.Second
			cosiProtocol.Threshold = Threshold
			// raha: added. this way, the roster that runs this protocol is  initiated by the main roster, the one that runs the basedfs protocol
			cosiProtocol.TreeNodeInstance = proto.TreeNodeInstance
			//----
			if NbrSubTrees > 0 {
				err := cosiProtocol.SetNbrSubTree(NbrSubTrees)
				if err != nil {
					return err
				}
			}
			proto.BlsCosi = cosiProtocol
			// ---------------------------------------------------------------
			proto.Start()

			px := <-proto.DoneBaseDFS
			log.Lvl1("Back to simulation module. Final result is", px)
			wait = false
		}
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
