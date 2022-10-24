package simul

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/chainBoostScale/ChainBoost/onet/log"
	"github.com/chainBoostScale/ChainBoost/simulation/platform"
	"golang.org/x/xerrors"
)

// todoRaha: change it to ghada lab later

var nobuild = false
var clean = true
var build = "simul"
var machines = 3
var simRange = ""
var race = false
var runWait = 180 * time.Second
var experimentWait = 0 * time.Second

var platformDst = "localhost"

//var platformDst = "deterlab"
//var platformDst string //= "deterlab"

func init() {
	flag.BoolVar(&nobuild, "nobuild", false, "Don't rebuild all helpers")
	flag.BoolVar(&clean, "clean", false, "Only clean platform")
	flag.StringVar(&build, "build", "", "List of packages to build")
	flag.BoolVar(&race, "race", false, "Build with go's race detection enabled (doesn't work on all platforms)")
	flag.IntVar(&machines, "machines", machines, "Number of machines on Deterlab")
	flag.StringVar(&simRange, "range", simRange, "Range of simulations to run. 0: or 3:4 or :4")
	flag.DurationVar(&runWait, "runwait", runWait, "How long to wait for each simulation to finish - overwrites .toml-value")
	flag.DurationVar(&experimentWait, "experimentwait", experimentWait, "How long to wait for the whole experiment to finish")
	flag.StringVar(&platformDst, "platform", platformDst, "platform to deploy to [localhost,mininet,deterlab]")

}

// Reads in the platform that we want to use and prepares for the tests
//func startBuild(customPlatform string) {
func startBuild() {
	//platformDst = customPlatform
	flag.Parse()
	log.Lvl5("platformDst is:", platformDst)
	deployP := platform.NewPlatform(platformDst)
	if deployP == nil {
		log.Fatal("Platform not recognized.", platformDst)
	}
	log.LLvl1("Deploying to", platformDst)

	simulations := flag.Args()
	if len(simulations) == 0 {
		log.Fatal("Please give a simulation to run")
	}
	log.LLvl1("simulations are: ", simulations)

	for _, simulation := range simulations {
		runconfigs := platform.ReadRunFile(deployP, simulation)

		if len(runconfigs) == 0 {
			log.Fatal("No tests found in", simulation)
		}

		/* : instead of reading these config variables from the toml connfig file, we set an initialized
		   value for each on simul.go and we can dynamically change them when running each simulation
		   via flags with exact same names
		   e.g.: go test -platform=deterlab -MCRoundDuration=10 -timeout 300000s -run ^TestSimulation$
		*/

		// : converting string values read from toml file to int values
		MCRoundDuration, _ := strconv.Atoi(runconfigs[0].Get("MCRoundDuration"))
		PercentageTxPay, _ := strconv.Atoi(runconfigs[0].Get("PercentageTxPay"))
		MainChainBlockSize, _ := strconv.Atoi(runconfigs[0].Get("MainChainBlockSize"))
		SideChainBlockSize, _ := strconv.Atoi(runconfigs[0].Get("SideChainBlockSize"))
		SectorNumber, _ := strconv.Atoi(runconfigs[0].Get("SectorNumber"))
		NumberOfPayTXsUpperBound, _ := strconv.Atoi(runconfigs[0].Get("NumberOfPayTXsUpperBound"))
		SimulationRounds, _ := strconv.Atoi(runconfigs[0].Get("SimulationRounds"))
		SimulationSeed, _ := strconv.Atoi(runconfigs[0].Get("SimulationSeed"))
		NbrSubTrees, _ := strconv.Atoi(runconfigs[0].Get("NbrSubTrees"))
		Threshold, _ := strconv.Atoi(runconfigs[0].Get("Threshold"))
		SCRoundDuration, _ := strconv.Atoi(runconfigs[0].Get("SCRoundDuration"))
		CommitteeWindow, _ := strconv.Atoi(runconfigs[0].Get("CommitteeWindow"))
		MCRoundPerEpoch, _ := strconv.Atoi(runconfigs[0].Get("MCRoundPerEpoch"))
		SimState, _ := strconv.Atoi(runconfigs[0].Get("SimState"))

		deployP.Configure(&platform.Config{
			Debug: log.DebugVisible(),
			Suite: runconfigs[0].Get("Suite"),
			// : adding some other system-wide configurations
			MCRoundDuration:          MCRoundDuration,
			PercentageTxPay:          PercentageTxPay,
			MainChainBlockSize:       MainChainBlockSize,
			SideChainBlockSize:       SideChainBlockSize,
			SectorNumber:             SectorNumber,
			NumberOfPayTXsUpperBound: NumberOfPayTXsUpperBound,
			SimulationRounds:         SimulationRounds,
			SimulationSeed:           SimulationSeed,
			NbrSubTrees:              NbrSubTrees,
			Threshold:                Threshold,
			SCRoundDuration:          SCRoundDuration,
			CommitteeWindow:          CommitteeWindow,
			MCRoundPerEpoch:          MCRoundPerEpoch,
			SimState:                 SimState,
		})

		if clean {
			err := deployP.Deploy(runconfigs[0])
			if err != nil {
				log.Fatal("Couldn't deploy:", err)
			}
			if err := deployP.Cleanup(); err != nil {
				log.Error("Couldn't cleanup correctly:", err)
			}
		} else {
			logname := strings.Replace(filepath.Base(simulation), ".toml", "", 1)
			testsDone := make(chan bool)
			// Raha: set timeout for the experiment from the config file
			//timeout, err := getExperimentWait(runconfigs)
			//t, err := strconv.Atoi(runconfigs[0].Get("timeout"))
			//timeout := time.Duration(int64(t)) * time.Second
			// if err != nil {
			// 	log.Fatal("ExperimentWait:", err)
			// 	panic("Raha: set timeout for the experiment from the config file")
			// }
			go func() {
				RunTests(deployP, logname, runconfigs)
				testsDone <- true
			}()
			select {
			case <-testsDone:
				// case <-time.After(timeout):
				// 	log.Fatal("Test failed to finish (by returning from RunTests) in", timeout)
			}
		}
	}
}

// RunTests the given tests and puts the output into the
// given file name. It outputs RunStats in a CSV format.
func RunTests(deployP platform.Platform, name string, runconfigs []*platform.RunConfig) {

	if nobuild == false {
		if race {
			if err := deployP.Build(build, "-race"); err != nil {
				log.Error("Couln't finish build without errors:",
					err)
			}
		} else {
			if err := deployP.Build(build); err != nil {
				log.Error("Couln't finish build without errors:",
					err)
			}
		}
	}

	//mkTestDir()
	//args := os.O_CREATE | os.O_RDWR | os.O_TRUNC
	// If a range is given, we only append
	// if simRange != "" {
	// 	args = os.O_CREATE | os.O_RDWR | os.O_APPEND
	// }
	// files := []*os.File{}
	// defer func() {
	// 	for _, f := range files {
	// 		if err := f.Close(); err != nil {
	// 			log.Error("Couln't close", f.Name())
	// 		}
	// 	}
	// }()

	start, stop := getStartStop(len(runconfigs))
	for i, rc := range runconfigs {
		// Implement a simple range-argument that will skip checks not in range
		if i < start || i > stop {
			log.LLvl1("Skipping", rc, "because of range")
			continue
		}

		// run test t nTimes times
		// take the average of all successful runs
		log.LLvl1("Running test with config:", rc)
		err := RunTest(deployP, rc)
		if err != nil {
			log.Error("Error running test:", err)
			continue
		}
	}
}

// RunTest a single test - takes a test-file as a string that will be copied
// to the deterlab-server
func RunTest(deployP platform.Platform, rc *platform.RunConfig) error {
	CheckHosts(rc)
	rc.Delete("simulation")

	//todoRaha : temp commented
	// if err := deployP.Cleanup(); err != nil {
	// 	log.Error(err)
	// 	return nil, xerrors.Errorf("cleanup: %v", err)
	// }

	if err := deployP.Deploy(rc); err != nil {
		log.Error(err)
		return xerrors.Errorf("deploy: %v", err)
	}

	done := make(chan error)

	go func() {
		var err error
		// Start monitor before so ssh tunnel can connect to the monitor
		// in case of deterlab.
		err = deployP.Start()
		if err != nil {
			done <- err
			return
		}

		log.LLvl1("\n======= ******* ======== \nSimulation is built and sent over to the remote server.\nGo to the remote directory created on the home. \nand then run the users exe file there\n======= ******* ======== \n")

		// if err = deployP.Wait(); err != nil {
		// 	log.Error("Test failed:", err)
		// 	if err := deployP.Cleanup(); err != nil {
		// 		log.LLvl1("Couldn't cleanup platform:", err)
		// 	}
		// 	done <- err
		// 	return
		// }
		done <- nil
	}()

	// timeout, err := getRunWait(rc)
	// if err != nil {
	// 	log.Fatal("RunWait:", err)
	// }

	// can timeout the command if it takes too long
	select {
	case err := <-done:
		if err != nil {
			return xerrors.Errorf("simulation error: %v", err)
		}
		//return stats, nil
		return nil
		// case <-time.After(timeout):
		// 	return nil, xerrors.New("simulation timeout")
	}
}

// CheckHosts verifies that at least two out of the three parameters: hosts, BF
// and depth are set in RunConfig. If one is missing, it tries to fix it. When
// more than one is missing, it stops the program.
func CheckHosts(rc *platform.RunConfig) {
	hosts, _ := rc.GetInt("hosts")
	bf, _ := rc.GetInt("bf")
	depth, _ := rc.GetInt("depth")
	if hosts == 0 {
		if depth == 0 || bf == 0 {
			log.Fatal("When hosts is not set, depth and BF must be set.")
		}
		hosts = calcHosts(bf, depth)
		rc.Put("hosts", strconv.Itoa(hosts))
	} else if bf == 0 {
		if depth == 0 || hosts == 0 {
			log.Fatal("When BF is not set, depth and hosts must be set.")
		}
		bf = 1
		for calcHosts(bf, depth) < hosts {
			bf++
		}
		rc.Put("bf", strconv.Itoa(bf))
	} else if depth == 0 {
		if hosts == 0 || bf == 0 {
			log.Fatal("When depth is not set, hsots and BF must be set.")
		}
		depth = 1
		for calcHosts(bf, depth) < hosts {
			depth++
		}
		rc.Put("depth", strconv.Itoa(depth))
	}
	// don't do anything if all three parameters are set
}

// Geometric sum to count the total number of nodes:
// Root-node: 1
// 1st level: bf (branching-factor)*/
// 2nd level: bf^2 (each child has bf children)
// 3rd level: bf^3
// So total: sum(level=0..depth)(bf^level)
func calcHosts(bf, depth int) int {
	if bf <= 0 {
		log.Fatal("illegal branching-factor")
	} else if depth <= 0 {
		log.Fatal("illegal depth")
	} else if bf == 1 {
		return depth + 1
	}
	return int((1 - math.Pow(float64(bf), float64(depth+1))) /
		float64(1-bf))
}

type runFile struct {
	Machines int
	Args     string
	Runs     string
}

func mkTestDir() {
	err := os.MkdirAll("test_data/", 0777)
	if err != nil {
		log.Fatal("failed to make test directory")
	}
}

func generateResultFileName(name string, index int) string {
	if index == 0 {
		// don't add the bucket index if it is the global one
		return fmt.Sprintf("test_data/%s.csv", name)
	}

	return fmt.Sprintf("test_data/%s_%d.csv", name, index)
}

// returns a tuple of start and stop configurations to run
func getStartStop(rcs int) (int, int) {
	ssStr := strings.Split(simRange, ":")
	start, err := strconv.Atoi(ssStr[0])
	stop := rcs - 1
	if err == nil {
		stop = start
		if len(ssStr) > 1 {
			stop, err = strconv.Atoi(ssStr[1])
			if err != nil {
				stop = rcs
			}
		}
	}
	log.LLvl1("Range is", start, ":", stop)
	return start, stop
}

// getRunWait returns either the command-line value or the value from the runconfig
// file
func getRunWait(rc *platform.RunConfig) (time.Duration, error) {
	rcWait, err := rc.GetDuration("runwait")
	if err == platform.ErrorFieldNotPresent {
		return runWait, nil
	}
	if err == nil {
		return rcWait, nil
	}
	return 0, err
}

// getExperimentWait returns, in the following order of precedence:
// 1. the command-line value
// 2. the value from runconfig
// 3. #runconfigs * runWait
func getExperimentWait(rcs []*platform.RunConfig) (time.Duration, error) {
	if experimentWait > 0 {
		return experimentWait, nil
	}
	rcExp, err := rcs[0].GetDuration("experimentwait")
	if err == nil {
		return rcExp, nil
	}
	// Probably a parse error parsing the duration.
	if err != platform.ErrorFieldNotPresent {
		return 0, err
	}

	// Otherwise calculate a useful default.
	wait := 0 * time.Second
	for _, rc := range rcs {
		w, err := getRunWait(rc)
		if err != nil {
			return 0, err
		}
		wait += w
	}
	return wait, nil
}
