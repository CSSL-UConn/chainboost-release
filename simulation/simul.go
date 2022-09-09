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
	"os/exec"

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
var Suite string

// logging level
var Debug = 5

// -------------------------------------
// Raha: chainboost dynamic config variables
// -------------------------------------
var PercentageTxPay = 30
var MCRoundDuration = 5          //sec
var MainChainBlockSize = 2000000 //byte
var SideChainBlockSize = 1000000
var SectorNumber = 2
var NumberOfPayTXsUpperBound = 2000
var SimulationRounds = 20
var SimulationSeed = 9
var NbrSubTrees = 1
var Threshold = 8 //out of committee nodes
var SCRoundDuration = 1
var CommitteeWindow = 10 //nodes
var MCRoundPerEpoch = 1
var SimState = 2

// -------------------------------------
// Initialize before 'init' so we can directly use the fields as parameters

func init() {
	flag.StringVar(&serverAddress, "address", "", "our address to use")
	flag.StringVar(&monitorAddress, "monitor", "", "remote monitor")
	flag.StringVar(&simul, "simul", "", "start simulating that protocol")
	flag.StringVar(&Suite, "suite", "Ed25519", "cryptographic suite to use")
	//----
	flag.IntVar(&Debug, "Debug", Debug, "logging level")
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
	log.RegisterFlags()
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
			log.LLvl5("Running toml-file:", rc)
			os.Args = []string{os.Args[0], rc}
			Start()
		}
		return
	}
	flag.Parse()
	if simul == "" {
		startBuild()
	} else {
		log.Lvl3("simul and PercentageTxPay flags are:", simul, " ", PercentageTxPay)
		// -------------------------------------
		//get current vm's ip
		var serverAddress, monitorAddress string
		conn, er := net.Dial("udp", "8.8.8.8:80")
		if er != nil {
			log.Fatal(er)
		}
		defer conn.Close()
		localAddr := conn.LocalAddr().(*net.UDPAddr)
		//log.LLvl5("localAddr.IP:", localAddr.IP)
		host := localAddr.IP.String()
		serverAddress = host

		// : port 2000 is bcz in start.py file they have initialized it with port 2000!
		monitorAddress = "192.168.3.220:2000"
		//suite = "bn256.adapter"

		//todo: what monitor is for? what port?

		err := platform.Simulate(PercentageTxPay, MCRoundDuration, MainChainBlockSize, SideChainBlockSize,
			SectorNumber, NumberOfPayTXsUpperBound, SimulationRounds, SimulationSeed,
			NbrSubTrees, Threshold, SCRoundDuration, CommitteeWindow,
			MCRoundPerEpoch, SimState,
			Suite, serverAddress, simul, monitorAddress)
		if err != nil {
			log.LLvl1("Raha: err returned from simulate: ", err)
			log.ErrFatal(err)
			cmd := exec.Command("sudo", "kill", "-9", "-1")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			var err error
			//go func() {
			err = cmd.Run()
			//}()
			if err != nil {
				log.Fatal("Raha: Couldn't killall listening threads:", err)
			} else {
				log.Lvl1("Raha: all listener on VM", localAddr, "are killed.")
			}
		} else {
			log.LLvl1("Raha: func simulate on ", localAddr, " returned without err")
		}
	}
	os.Chdir(wd)
}
