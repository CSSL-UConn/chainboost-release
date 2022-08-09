/*
Package simul allows for easy simulation on different platforms. THe following platforms
are available:

	- localhost - for up to 100 nodes
	- mininet - for up to 1'000 nodes
	- deterlab - for up to 50'000 nodes

Usually you start small, then work your way up to the full potential of your
protocol!
*/
package simul

import (
	"flag"
	"net"
	"os"

	"github.com/chainBoostScale/ChainBoost/onet/log"
	"github.com/chainBoostScale/ChainBoost/simulation/platform"
)

// The address of this server - if there is only one server in the config
// file, it will be derived from it automatically
var serverAddress string

// ip addr of the logger to connect to
var monitorAddress string

// Simul is != "" if this node needs to start a simulation of that protocol
var simul string

// suite is Ed25519 by default
var suite string

// -------------------------------------
// Raha: chainboost dynamic config variables
// -------------------------------------
var PercentageTxPay = 30
var MCRoundDuration = 10
var MainChainBlockSize = 2000000
var SideChainBlockSize = 500000
var SectorNumber = 2
var NumberOfPayTXsUpperBound = 1000
var SimulationRounds = 50
var SimulationSeed = 9
var NbrSubTrees = 1
var Threshold = 4
var SCRoundDuration = 5
var CommitteeWindow = 5
var MCRoundPerEpoch = 5
var SimState = 2

// -------------------------------------
// Initialize before 'init' so we can directly use the fields as parameters
// to 'Flag'
//raha:like above???!!!

func init() {
	flag.StringVar(&serverAddress, "address", "", "our address to use")
	flag.StringVar(&monitorAddress, "monitor", "", "remote monitor")
	flag.StringVar(&simul, "simul", "", "start simulating that protocol")
	flag.StringVar(&suite, "suite", "Ed25519", "cryptographic suite to use")
	//----
	flag.IntVar(&PercentageTxPay, "PercentageTxPay", PercentageTxPay, "PercentageTxPay")
	flag.IntVar(&MCRoundDuration, "MCRoundDuration", MCRoundDuration, "MCRoundDuration")
	flag.IntVar(&MainChainBlockSize, "MainChainBlockSize", MainChainBlockSize, "MainChainBlockSize")
	flag.IntVar(&SideChainBlockSize, "SideChainBlockSize", SideChainBlockSize, "SideChainBlockSize")
	flag.IntVar(&SectorNumber, "SectorNumber", SectorNumber, "SectorNumber")
	flag.IntVar(&NumberOfPayTXsUpperBound, "NumberOfPayTXsUpperBound", NumberOfPayTXsUpperBound, "NumberOfPayTXsUpperBound")
	flag.IntVar(&SimulationRounds, "SimulationRounds", SimulationRounds, "SimulationRounds")
	flag.IntVar(&SimulationSeed, "SimulationSeed", SimulationSeed, "SimulationSeed")
	flag.IntVar(&NbrSubTrees, "NbrSubTrees", NbrSubTrees, "NbrSubTrees")
	flag.IntVar(&Threshold, "Threshold", Threshold, "Threshold")
	flag.IntVar(&SCRoundDuration, "SCRoundDuration", SCRoundDuration, "SCRoundDuration")
	flag.IntVar(&CommitteeWindow, "CommitteeWindow", CommitteeWindow, "CommitteeWindow")
	flag.IntVar(&MCRoundPerEpoch, "MCRoundPerEpoch", MCRoundPerEpoch, "MCRoundPerEpoch")
	flag.IntVar(&SimState, "SimState", SimState, "SimState")
}

// Start has to be called by the main-file that imports the protocol and/or the
// service. If a user calls the simulation-file, `simul` is empty, and the
// build is started.
// Only the platform will call this binary with a simul-flag set to the name of the
// simulation to run.
// If given an array of rcs, each element will be interpreted as a .toml-file
// to load and simulate.
func Start(rcs ...string) {
	wd, er := os.Getwd()
	if len(rcs) > 0 {
		log.ErrFatal(er)
		for _, rc := range rcs {
			log.LLvl1("Running toml-file:", rc)
			os.Args = []string{os.Args[0], rc}
			Start()
		}
		return
	}
	flag.Parse()
	if simul == "" {
		log.Lvl1("Raha: simul is empty!, this is for the platform/ROOT node?")
		startBuild()
	} else {
		log.LLvl1("Raha: simul is not empty, it is for simple nodes/miners")
		// -------------------------------------
		//get current vm's ip
		var serverAddress, monitorAddress string
		conn, er := net.Dial("udp", "8.8.8.8:80")
		if er != nil {
			log.Fatal(er)
		}
		defer conn.Close()
		localAddr := conn.LocalAddr().(*net.UDPAddr)
		log.LLvl1("localAddr.IP:", localAddr.IP)
		host := localAddr.IP.String()
		serverAddress = host

		// raha: port 2000 is bcz in start.py file they have initialized it with port 2000!
		monitorAddress = "192.168.3.220:2000"
		suite = "bn256.adapter"

		//todoraha: what monitor is for? what port?

		err := platform.Simulate(PercentageTxPay, MCRoundDuration, MainChainBlockSize, SideChainBlockSize,
			SectorNumber, NumberOfPayTXsUpperBound, SimulationRounds, SimulationSeed,
			NbrSubTrees, Threshold, SCRoundDuration, CommitteeWindow,
			MCRoundPerEpoch, SimState,
			suite, serverAddress, simul, monitorAddress)
		if err != nil {
			log.LLvl1("Raha: err returned from simulate: ", err)
			log.ErrFatal(err)
		} else {
			log.LLvl1("Raha: func simulate returned without err")
		}
	}
	os.Chdir(wd)
}

// func StartDistributedSimulation(rcs ...string) {
// 	wd, err := os.Getwd()
// 	if len(rcs) > 0 {
// 		log.ErrFatal(err)
// 		for _, rc := range rcs {
// 			log.LLvl1("Running toml-file:", rc)
// 			os.Args = []string{os.Args[0], rc}
// 			StartDistributedSimulation()
// 		}
// 		return
// 	}
// 	flag.Parse()
// 	if simul == "" {
// 		log.LLvl1("Raha: simul is empty")
// 		startBuild("deterlab")
// 	} else {
// 		log.LLvl1("Raha: simul is not empty!")
// 		err := platform.Simulate(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
// 			suite, serverAddress, simul, monitorAddress)
// 		log.ErrFatal(err)
// 	}
// 	os.Chdir(wd)
// }
