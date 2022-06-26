package main

import (
	"net"
	"os"

	//"os/exec"

	"github.com/BurntSushi/toml"
	"github.com/chainBoostScale/ChainBoost/onet"
	"github.com/chainBoostScale/ChainBoost/onet/log"
	"github.com/chainBoostScale/ChainBoost/simulation/platform"

	"golang.org/x/xerrors"
)

/*Defines the simulation for each-protocol*/
func init() {
	onet.SimulationRegister("ChainBoost", NewSimulation)
}

// Simulation only holds the BFTree simulation
type simulation struct {
	onet.SimulationBFTree
}

// NewSimulation returns the new simulation, where all fields are
// initialised using the config-file
func NewSimulation(config string) (onet.Simulation, error) {
	es := &simulation{}
	// Set defaults before toml.Decode
	es.Suite = "Ed25519"

	_, err := toml.Decode(config, es)
	if err != nil {
		return nil, xerrors.Errorf("decoding: %v", err)
	}
	return es, nil
}

// Setup creates the tree used for that simulation
func (e *simulation) Setup(dir string, hosts []string) (
	*onet.SimulationConfig, error) {
	sc := &onet.SimulationConfig{}
	log.LLvl1("raha: creating roster with hosts number of nodes, out of given addresses and starting from port:2000")
	e.CreateRoster(sc, hosts, 2000)
	err := e.CreateTree(sc)
	if err != nil {
		return nil, xerrors.Errorf("creating tree: %v", err)
	}
	return sc, nil
}

// Run is used on the destination machines and runs a number of
// rounds
/*
func (e *simulation) Run(config *onet.SimulationConfig) error {
	size := config.Tree.Size()
	log.LLvl1("Size is:", size, "rounds:", e.Rounds)
	for round := 0; round < e.Rounds; round++ {
		log.LLvl1("Starting round", round)
		round := monitor.NewTimeMeasure("round")
		p, err := config.Overlay.CreateProtocol("Count", config.Tree, onet.NilServiceID)
		if err != nil {
			return xerrors.Errorf("creating protocol: %v", err)
		}
		go p.Start()
		children := <-p.(*manage.ProtocolCount).Count
		round.Record()
		if children != size {
			return xerrors.New("Didn't get " + strconv.Itoa(size) +
				" children")
		}
	}
	return nil
}
*/
//Raha
func (e *simulation) Run(config *onet.SimulationConfig) error {
	size := config.Tree.Size()
	log.LLvl1("Size is:", size, "rounds:", e.Rounds)
	for round := 0; round < e.Rounds; round++ {
		log.LLvl1("Starting round", round)
		//round := monitor.NewTimeMeasure("round")
		p, err := config.Overlay.CreateProtocol("ChainBoost", config.Tree, onet.NilServiceID)
		if err != nil {
			return xerrors.Errorf("creating protocol: %v", err)
		}
		go p.Start()
		//round.Record()
	}
	return nil
}

func main() {
	log.LLvl1("running ./simul exe in: main package, main function")

	//raha commented
	//simul.Start("ChainBoost.toml")

	wd, err := os.Getwd()
	log.LLvl1("Running toml-file: ", "ChainBoost.toml")
	os.Args = []string{os.Args[0], "ChainBoost.toml"}
	log.LLvl1("Raha: simul is not empty!")

	// -------------------------------------
	//get current vm's ip
	var serverAddress, monitorAddress string
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	log.LLvl1("cmd out:", localAddr.IP)
	host := localAddr.IP.String()
	if host == "192.168.3.212" {
		serverAddress = host
		monitorAddress = host
	} else {
		serverAddress = host
		monitorAddress = "192.168.3.212"
	}
	// -------------------------------------
	/*
		func Simulate(PercentageTxPay, MCRoundDuration, MainChainBlockSize, SideChainBlockSize,
			SectorNumber, NumberOfPayTXsUpperBound, SimulationRounds, SimulationSeed,
			NbrSubTrees, Threshold, SCRoundDuration, CommitteeWindow,
			MCRoundPerEpoch, SimState int,
		suite, serverAddress, simul, monitorAddress string) error
	*/
	//ToDoRaha: Now: these values should be read from the chainBoost toml file!
	//todoraha: what monitor is for? what port?
	// raha: port 2000 is bcz in start.py file they have initialized it with port 2000!

	err = platform.Simulate(30, 10, 30000, 25000, 2, 50, 40, 9, 1, 4, 5, 5, 5, 2,
		"bn256.adapter", serverAddress, "ChainBoost", monitorAddress+":2000")
	if err != nil {
		log.LLvl1("Raha: err returned from simulate: ", err)
	} else {
		log.LLvl1("Raha: func simulate returned without err")
	}
	log.ErrFatal(err)
	os.Chdir(wd)
}
