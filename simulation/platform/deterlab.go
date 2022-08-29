// Deterlab is responsible for setting up everything to test the application
// on deterlab.net
// Given a list of hostnames, it will create an overlay
// tree topology, using all but the last node. It will create multiple
// nodes per server and run timestamping processes. The last node is
// reserved for the logging server, which is forwarded to localhost:8081
//
// Creates the following directory structure:
// build/ - where all cross-compiled executables are stored
// remote/ - directory to be copied to the deterlab server
//
// The following apps are used:
//   deter - runs on the user-machine in deterlab and launches the others
//   forkexec - runs on the other servers and launches the app, so it can measure its cpu usage

package platform

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"runtime"

	//"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	//"github.com/chainBoostScale/ChainBoost/onet/app"
	"github.com/chainBoostScale/ChainBoost/MainAndSideChain/blockchain"
	"github.com/chainBoostScale/ChainBoost/onet"
	"github.com/chainBoostScale/ChainBoost/onet/app"
	"github.com/chainBoostScale/ChainBoost/onet/log"
	"golang.org/x/xerrors"
)

// Deterlab holds all fields necessary for a Deterlab-run
type Deterlab struct {
	// *** Deterlab-related configuration
	// The login on the platform
	Login string
	// The outside host on the platform
	Host string
	// The name of the project
	Project string
	// Name of the Experiment - also name of hosts
	Experiment string
	// Directory holding the simulation-main file
	simulDir string
	// Directory where the deterlab-users-file is held
	usersDir string
	// Directory where everything is copied into
	deployDir string
	// Directory for building
	buildDir string
	// Directory holding all go-files of onet/simulation/platform
	platformDir string
	// DNS-resolvable names
	Phys []string
	// VLAN-IP names (physical machines)
	Virt []string
	// Channel to communication stopping of experiment
	sshDeter chan string
	// Whether the simulation is started
	started bool

	// ProxyAddress : the proxy will redirect every traffic it
	// receives to this address
	ProxyAddress string
	// MonitorAddress is the address given to clients to connect to the monitor
	// It is actually the Proxy that will listen to that address and clients
	// won't know a thing about it
	MonitorAddress string
	// Port number of the monitor and the proxy
	MonitorPort int

	// Number of available servers
	Servers int
	// Name of the simulation
	Simulation string
	// Number of machines
	Hosts int
	// Debugging-level: 0 is none - 5 is everything
	Debug int
	// RunWait for long simulations
	RunWait string
	// suite used for the simulation
	Suite string
	// PreScript defines a script that is run before the simulation
	PreScript string
	// Tags to use when compiling
	Tags string

	// : adding some other system-wide configurations
	MCRoundDuration          int
	PercentageTxPay          int
	MainChainBlockSize       int
	SideChainBlockSize       int
	SectorNumber             int
	NumberOfPayTXsUpperBound int
	SimulationRounds         int
	SimulationSeed           int
	//-- bls cosi
	NbrSubTrees     int
	Threshold       int
	SCRoundDuration int
	CommitteeWindow int
	MCRoundPerEpoch int
	SimState        int
}

var simulConfig *onet.SimulationConfig

// Configure initialises the directories and loads the saved config
// for Deterlab
func (d *Deterlab) Configure(pc *Config) {
	// Directory setup - would also be possible in /tmp
	pwd, _ := os.Getwd()
	d.Suite = pc.Suite
	d.simulDir = pwd
	d.deployDir = pwd + "/deploy"
	d.buildDir = pwd + "/build"
	_, file, _, _ := runtime.Caller(0)
	d.platformDir = path.Dir(file)
	os.RemoveAll(d.deployDir)
	os.Mkdir(d.deployDir, 0770)
	os.Mkdir(d.buildDir, 0770)
	d.MonitorPort = pc.MonitorPort
	log.LLvl1("Dirs are:", pwd, d.deployDir, "configed monitor port is:", pc.MonitorPort)
	d.loadAndCheckDeterlabVars()
	// ------------------------------
	// : adding some other system-wide configurations
	d.MCRoundDuration = pc.MCRoundDuration
	d.PercentageTxPay = pc.PercentageTxPay
	d.MainChainBlockSize = pc.MainChainBlockSize
	d.SideChainBlockSize = pc.SideChainBlockSize
	d.SectorNumber = pc.SectorNumber
	d.NumberOfPayTXsUpperBound = pc.NumberOfPayTXsUpperBound
	d.SimulationRounds = pc.SimulationRounds
	d.SimulationSeed = pc.SimulationSeed
	d.NbrSubTrees = pc.NbrSubTrees
	d.Threshold = pc.Threshold
	d.SCRoundDuration = pc.SCRoundDuration
	d.CommitteeWindow = pc.CommitteeWindow
	d.MCRoundPerEpoch = pc.MCRoundPerEpoch
	d.SimState = pc.SimState
	// ------------------------------
	d.Debug = pc.Debug
	if d.Simulation == "" {
		log.Fatal("No simulation defined in runconfig")
	}

	// Setting up channel
	d.sshDeter = make(chan string)
}

type pkg struct {
	name      string
	processor string
	system    string
	path      string
}

// Build prepares all binaries for the Deterlab-simulation.
// If 'build' is empty, all binaries are created, else only
// the ones indicated. Either "simul" or "users"
func (d *Deterlab) Build(build string, arg ...string) error {
	log.LLvl1("Building for", d.Login, d.Host, d.Project, build, "simulDir=", d.simulDir)
	start := time.Now()

	var wg sync.WaitGroup

	if err := os.RemoveAll(d.buildDir); err != nil {
		return xerrors.Errorf("removing folders: %v", err)
	}
	if err := os.Mkdir(d.buildDir, 0777); err != nil {
		return xerrors.Errorf("making folder: %v", err)
	}

	// start building the necessary binaries - it's always the same,
	// but built for another architecture.
	packages := []pkg{
		//: changed
		// deter has an amd64, linux architecture
		{"simul", "arm64", "darwin", path.Join("/Users//Documents/github.com/chainBoostScale/ChainBoost/simulation/manage", "simulation")},
		{"users", "amd64", "linux", path.Join("/Users//Documents/github.com/chainBoostScale/ChainBoost/simulation/platform", "deterlab_users")},
		{"simul", "amd64", "linux", path.Join("/Users//Documents/github.com/chainBoostScale/ChainBoost/simulation/manage", "simulation")},
		//{"simul", "amd64", "linux", d.simulDir},
		//{"simul", "arm64", "linux", "/go/src/github.com/chainBoostScale/ChainBoost/simulation/manage/simulation"},
		//{"users", "arm64", "darwin", d.simulDir},
		//{"users", "arm64", "linux", d.simulDir},
		//{"users", "386", "freebsd", path.Join(d.platformDir, "deterlab_users")},
		//{"users", "arm64", "linux", path.Join(d.platformDir, "deterlab_users")},
	}
	if build == "" {
		build = "simul,users"
	}
	var tags []string
	if d.Tags != "" {
		tags = append([]string{"-tags"}, strings.Split(d.Tags, " ")...)
	}
	log.LLvl1("Starting to build all executables", packages)
	for _, p := range packages {
		if !strings.Contains(build, p.name) {
			log.LLvl1("Skipping build of", p.name)
			continue
		}
		log.LLvl1("Building", p)
		wg.Add(1)
		go func(p pkg) {
			defer wg.Done()
			dst := path.Join(d.buildDir, p.name)
			//
			var path string
			var err error

			d.simulDir = "/Users//Documents/github.com/chainBoostScale/ChainBoost/simulation/manage/simulation"
			d.platformDir = "/Users//Documents/github.com/chainBoostScale/ChainBoost/simulation/platform"

			path, err = filepath.Rel(d.simulDir, p.path)
			log.ErrFatal(err)

			var out string
			if p.name == "simul" {
				log.LLvl1("Building: simul")
				out, err = Build(path, dst,
					p.processor, p.system, append(arg, tags...)...)
			} else {
				log.LLvl1("Building: users")
				out, err = Build(path, dst,
					p.processor, p.system, arg...)
			}
			if err != nil {
				KillGo()
				log.Error(out)
				log.Fatal(err)
			}
		}(p)
	}
	// wait for the build to finish
	wg.Wait()
	log.LLvl1("Build is finished after", time.Since(start))
	return nil
}

// Cleanup kills all eventually remaining processes from the last Deploy-run
func (d *Deterlab) Cleanup() error {
	// Cleanup eventual ssh from the proxy-forwarding to the logserver
	err := exec.Command("pkill", "-9", "-f", "ssh -nNTf").Run()
	if err != nil {
		log.LLvl1("Error stopping ssh:", err)
	}

	// SSH to the deterlab-server and end all running users-processes
	log.LLvl1("Going to kill everything")
	var sshKill chan string
	sshKill = make(chan string)
	go func() {
		// Cleanup eventual residues of previous round - users and sshd
		if _, err := SSHRun(d.Login, d.Host, "killall -9 users sshd"); err != nil {
			log.LLvl1("Error while cleaning up:", err)
		}

		err := SSHRunStdout(d.Login, d.Host, "test -f remote/users && ( cd remote; ./users -kill )")
		if err != nil {
			log.LLvl1("NOT-Normal error from cleanup", err.Error())
			sshKill <- "error"
		}
		sshKill <- "stopped"
	}()

	for {
		select {
		case msg := <-sshKill:
			if msg == "stopped" {
				log.LLvl1("Users stopped")
				return nil
			}
			log.LLvl1("Received other command", msg, "probably the app didn't quit correctly")
		case <-time.After(time.Second * 20):
			log.LLvl1("Timeout error when waiting for end of ssh")
			return nil
		}
	}
}

// Deploy creates the appropriate configuration-files and copies everything to the
// deterlab-installation.
func (d *Deterlab) Deploy(rc *RunConfig) error {
	if err := os.RemoveAll(d.deployDir); err != nil {
		return xerrors.Errorf("removing folders: %v", err)
	}
	if err := os.Mkdir(d.deployDir, 0777); err != nil {
		return xerrors.Errorf("making folder: %v", err)
	}

	// Check for PreScript and copy it to the deploy-dir
	d.PreScript = rc.Get("PreScript")
	if d.PreScript != "" {
		_, err := os.Stat(d.PreScript)
		if !os.IsNotExist(err) {
			if err := app.Copy(d.deployDir, d.PreScript); err != nil {
				return xerrors.Errorf("copying: %v", err)
			}
		}
	}

	// deploy will get rsync to /remote on the NFS

	log.LLvl1("Deterlab: Deploying and writing config-files")
	sim, err := onet.NewSimulation(d.Simulation, string(rc.Toml()))
	if err != nil {
		return xerrors.Errorf("simulation: %v", err)
	}
	// Initialize the deter-struct with our current structure (for debug-levels
	// and such), then read in the app-configuration to overwrite eventual
	// 'Machines', 'ppm', '' or other fields
	deter := *d
	deterConfig := d.deployDir + "/deter.toml"
	_, err = toml.Decode(string(rc.Toml()), &deter)
	if err != nil {
		return xerrors.Errorf("decoding toml: %v", err)
	}
	//-----------------------------------
	// ToDoRaha: filling 2 attributes in deter struct: deter.Virt, deter.Phys
	// by an "string array of IPs and DNS resolvable host names"

	// createHosts and parseHost functions are deterlab API specific.
	// deter.createHosts()
	// Phys: DNS-resolvable names, Virt: VLAN-IP names (physical machines)
	log.LLvl1("Getting the hosts (Raha: vm addresses?)")
	deter.Phys = []string{}
	deter.Virt = []string{}
	//---
	deter.Phys = append(deter.Phys, "192.168.3.220:22")
	deter.Virt = append(deter.Virt, "192.168.3.220")
	deter.Phys = append(deter.Phys, "192.168.3.221:22")
	deter.Virt = append(deter.Virt, "192.168.3.221")
	deter.Phys = append(deter.Phys, "192.168.3.222:22")
	deter.Virt = append(deter.Virt, "192.168.3.222")
	deter.Phys = append(deter.Phys, "192.168.3.223:22")
	deter.Virt = append(deter.Virt, "192.168.3.223")
	//-----------------------------------

	log.LLvl1("Writing the config file :", deter)
	onet.WriteTomlConfig(deter, deterConfig, d.deployDir)

	simulConfig, err = sim.Setup(d.deployDir, deter.Virt)
	if err != nil {
		return xerrors.Errorf("simulation setup: %v", err)
	}
	simulConfig.Config = string(rc.Toml())
	log.LLvl1("Saving configuration")
	if err := simulConfig.Save(d.deployDir); err != nil {
		log.Error("Couldn't save configuration:", err)
	}

	// Copy limit-files for more connections
	ioutil.WriteFile(path.Join(d.deployDir, "simul.conf"),
		[]byte(simulConnectionsConf), 0444)

	// --------------------------------------------

	// Raha: initializing main chain's blockchain -------------------------
	log.LLvl1("Initializing main chain's blockchain")
	blockchain.InitializeMainChainBC(
		rc.Get("FileSizeDistributionMean"), rc.Get("FileSizeDistributionVariance"),
		rc.Get("ServAgrDurationDistributionMean"), rc.Get("ServAgrDurationDistributionVariance"),
		rc.Get("InitialPowerDistributionMean"), rc.Get("InitialPowerDistributionVariance"),
		rc.Get("Nodes"), rc.Get("SimulationSeed"))
	// Raha: initializing side chain's blockchain -------------------------
	blockchain.InitializeSideChainBC()

	// --------------------------------------------
	//ToDo : is it the best way to do so?!
	// Copying central bc files to deploy-directory so it gets transferred to distributed servers
	err = exec.Command("cp", d.simulDir+"/"+"mainchainbc.xlsx", d.deployDir).Run()
	if err != nil {
		log.Fatal("error copying mainchainbc.xlsx: ", err)
	}
	err = exec.Command("cp", d.simulDir+"/"+"sidechainbc.xlsx", d.deployDir).Run()
	if err != nil {
		log.Fatal("error copying sidechainbc.xlsx: ", err)
	}

	//ToDo : is it the best way to do so?!
	// Copying chainBoost.toml file to deploy-directory so it gets transferred to distributed servers
	err = exec.Command("cp", d.simulDir+"/"+d.Simulation+".toml", d.deployDir).Run()
	if err != nil {
		log.Fatal("error copying chainBoost.toml-file:", d.simulDir, d.Simulation+".toml to ", d.deployDir, err)
	}

	err = exec.Command("cp", d.simulDir+"/"+"simul.go", d.deployDir).Run()
	if err != nil {
		log.Fatal("error copying chainBoost.toml-file:", d.simulDir, d.Simulation+".toml", d.deployDir, err)
	}

	// Copying build-files to deploy-directory
	build, err := ioutil.ReadDir(d.buildDir)
	for _, file := range build {
		err = exec.Command("cp", d.buildDir+"/"+file.Name(), d.deployDir).Run()
		if err != nil {
			log.Fatal("error copying build-file:", d.buildDir, file.Name(), d.deployDir, err)
		}
	}

	// Copy everything over to uconn's gateway server
	log.LLvl1("Copying over to", d.Login, "@", d.Host)

	// todo: it works with out id_rsa now but I am not sure how I am authenticated to the gateway, will I need it or not!, I will keep it for now
	SSHString := "ssh -i '/Users//.ssh/id_rsa'"
	//ToDoRaha: fix this later
	err = Rsync(d.Login, d.Host, SSHString, d.deployDir+"/", "~/remote/")
	if err != nil {
		log.Fatal(err)
	}
	log.LLvl1("Done copying")

	return nil
}

// Start creates a tunnel for the monitor-output and contacts the Deterlab-
// server to run the simulation
func (d *Deterlab) Start(args ...string) error {
	// setup port forwarding for viewing log server
	d.started = true
	// Remote tunneling : the sink port is used both for the sink and for the
	// proxy => the proxy redirects packets to the same port the sink is
	// listening.
	// -n = stdout == /Dev/null, -N => no command stream, -T => no tty

	//todo: commented temp  do we need them?
	log.LLvl1("Setup remote port forwarding skipped")
	// redirection := strconv.Itoa(d.MonitorPort) + ":" + d.ProxyAddress + ":" + strconv.Itoa(d.MonitorPort)
	// cmd := []string{"-nNTf", "-o", "StrictHostKeyChecking=no", "-o", "ExitOnForwardFailure=yes", "-R",
	// 	redirection, fmt.Sprintf("%s@%s", d.Login, d.Host)}
	// exCmd := exec.Command("ssh", cmd...)
	// if err := exCmd.Start(); err != nil {
	// 	log.Fatal("Failed to start the ssh port forwarding:", err)
	// }
	// if err := exCmd.Wait(); err != nil {
	// 	log.Fatal("ssh port forwarding exited in failure:", err)
	// }
	// log.LLvl1("Setup remote port forwarding", cmd)
	//----------
	// ToDoRaha: let's not call ./user locally
	// go func() {
	// 	log.LLvl1("Raha: Running ./users on the server:", d.Login, "@", d.Host)
	// 	err := SSHRunStdout(d.Login, d.Host, "cd remote; ./users -suite="+d.Suite)
	// 	if err != nil {
	// 		log.LLvl1("Raha: err while running users.exe on", d.Login, "@", d.Host, " : ", err)
	// 	}
	// 	d.sshDeter <- "finished"
	// }()

	return nil
}

// Wait for the process to finish
func (d *Deterlab) Wait() error {
	wait, err := time.ParseDuration(d.RunWait)
	if err != nil || wait == 0 {
		wait = 600 * time.Second
		err = nil
	}
	if d.started {
		log.LLvl1("Simulation is started")
		select {
		case msg := <-d.sshDeter:
			if msg == "finished" {
				log.LLvl1("Received finished-message, not killing users")
				return nil
			}
			log.LLvl1("Received out-of-line message", msg)
		case <-time.After(wait):
			log.LLvl1("Quitting after waiting", wait)
			d.started = false
		}
		d.started = false
	}
	return nil
}

// Checks whether host, login and project are defined. If any of them are missing, it will
// ask on the command-line.
// For the login-variable, it will try to set up a connection to d.Host and copy over the
// public key for a more easy communication
func (d *Deterlab) loadAndCheckDeterlabVars() {
	deter := Deterlab{}
	err := onet.ReadTomlConfig(&deter, "deter.toml")
	d.Host, d.Login, d.Project, d.Experiment, d.ProxyAddress, d.MonitorAddress =
		deter.Host, deter.Login, deter.Project, deter.Experiment,
		deter.ProxyAddress, deter.MonitorAddress

	if err != nil {
		log.LLvl1("Couldn't read config-file - asking for default values")
	}

	if d.Host == "" {
		d.Host = readString("Please enter the hostname of deterlab", "csi-lab-ssh.engr.uconn.edu")
	}

	login, err := user.Current()
	log.ErrFatal(err)
	if d.Login == "" {
		d.Login = readString("Please enter the login-name on "+d.Host, login.Username)
	}

	if d.Project == "" {
		d.Project = readString("Please enter the project on deterlab", "SAFER")
	}

	if d.Experiment == "" {
		d.Experiment = readString("Please enter the Experiment on "+d.Project, "Dissent-CS")
	}

	if d.MonitorAddress == "" {
		d.MonitorAddress = readString("Please enter the Monitor address (where clients will connect)", "csi-lab-ssh.engr.uconn.edu:22")
	}
	if d.ProxyAddress == "" {
		d.ProxyAddress = readString("Please enter the proxy redirection address", "localhost")
	}

	onet.WriteTomlConfig(*d, "deter.toml")
}

// Shows a messages and reads in a string, eventually returning a default (dft) string
func readString(msg, dft string) string {
	fmt.Printf("%s [%s]:", msg, dft)

	reader := bufio.NewReader(os.Stdin)
	strnl, _ := reader.ReadString('\n')
	str := strings.TrimSpace(strnl)
	if str == "" {
		return dft
	}
	return str
}

const simulConnectionsConf = `
# This is for the onet-deterlab testbed, which can use up an awful lot of connections

* soft nofile 128000
* hard nofile 128000
`

// Write the hosts.txt file automatically
// from project name and number of servers
// func (d *Deterlab) createHosts() {
// 	// Query deterlab's API for servers
// 	log.LLvl1("Querying Deterlab's API to retrieve server names and addresses")
// 	command := fmt.Sprintf("/usr/testbed/bin/expinfo -l -e %s,%s", d.Project, d.Experiment)
// 	apiReply, err := SSHRun(d.Login, d.Host, command)
// 	if err != nil {
// 		log.Fatal("Error while querying Deterlab:", err)
// 	}
// 	log.ErrFatal(d.parseHosts(string(apiReply)))
// }
// func (d *Deterlab) parseHosts(str string) error {
// 	// Get the link-information, which is the second block in `expinfo`-output
// 	infos := strings.Split(str, "\n\n")
// 	if len(infos) < 2 {
// 		return xerrors.New("didn't recognize output of 'expinfo'")
// 	}
// 	linkInfo := infos[1]
// 	// Test for correct version in case the API-output changes
// 	if !strings.HasPrefix(linkInfo, "Virtual Lan/Link Info:") {
// 		return xerrors.New("didn't recognize output of 'expinfo'")
// 	}
// 	linkLines := strings.Split(linkInfo, "\n")
// 	if len(linkLines) < 5 {
// 		return xerrors.New("didn't recognice output of 'expinfo'")
// 	}
// 	nodes := linkLines[3:]
// 	d.Phys = []string{}
// 	d.Virt = []string{}
// 	names := make(map[string]bool)
// 	for i, node := range nodes {
// 		if i%2 == 1 {
// 			continue
// 		}
// 		matches := strings.Fields(node)
// 		if len(matches) != 6 {
// 			return xerrors.New("expinfo-output seems to have changed")
// 		}
// 		// Convert client-0:0 to client-0
// 		name := strings.Split(matches[1], ":")[0]
// 		ip := matches[2]
// 		fullName := fmt.Sprintf("%s.%s.%s.isi.deterlab.net", name, d.Experiment, d.Project)
// 		log.LLvl1("Discovered", fullName, "on ip", ip)
// 		if _, exists := names[fullName]; !exists {
// 			d.Phys = append(d.Phys, fullName)
// 			d.Virt = append(d.Virt, ip)
// 			names[fullName] = true
// 		}
// 	}
// 	log.LLvl1("Physical:", d.Phys)
// 	log.LLvl1("Internal:", d.Virt)
// 	return nil
// }
