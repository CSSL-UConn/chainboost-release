package main

import (
	"github.com/BurntSushi/toml"
	"github.com/chainBoostScale/ChainBoost/onet"
	"github.com/chainBoostScale/ChainBoost/onet/log"
	simul "github.com/chainBoostScale/ChainBoost/simulation"

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
	log.Lvl2("Size is:", size, "rounds:", e.Rounds)
	for round := 0; round < e.Rounds; round++ {
		log.Lvl1("Starting round", round)
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
	log.Lvl2("Size is:", size, "rounds:", e.Rounds)
	for round := 0; round < e.Rounds; round++ {
		log.Lvl1("Starting round", round)
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
	//ToDoRaha!!
	log.LLvl1("The simulation starts from here in the distributed servers???")
	simul.Start("ChainBoost.toml")
}
