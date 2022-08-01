package main

import (
	"flag"
	"os"

	//"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"runtime"
	"strconv"

	"github.com/chainBoostScale/ChainBoost/onet"
	"github.com/chainBoostScale/ChainBoost/onet/log"

	//"github.com/chainBoostScale/ChainBoost/simulation/monitor"
	"github.com/chainBoostScale/ChainBoost/simulation/platform"
)

//todoraha: commeneted: why!!!
var kill = false

//var suite string
// func init() {
// 	flag.BoolVar(&kill, "kill", false, "kill everything (and don't start anything)")
// 	flag.StringVar(&suite, "suite", "ed25519", "suite used for simulation")
// }

// The address of this server - if there is only one server in the config
// file, it will be derived from it automatically
var serverAddress string

// ip addr of the logger to connect to
var monitorAddress string

// Simul is != "" if this node needs to start a simulation of that protocol
var simul string

// suite is Ed25519 by default
var suite string

// Initialize before 'init' so we can directly use the fields as parameters
// to 'Flag'
func init() {
	flag.BoolVar(&kill, "kill", false, "kill everything (and don't start anything)")
	flag.StringVar(&serverAddress, "address", "", "our address to use")
	flag.StringVar(&simul, "simul", "", "start simulating that protocol")
	flag.StringVar(&monitorAddress, "monitor", "", "remote monitor")
	flag.StringVar(&suite, "suite", "Ed25519", "cryptographic suite to use")

}

// DeterlabUsers is called on the users.deterlab.net-server and will:
// - copy the simulation-files to the server
// - start the simulation
func main() {
	log.LLvl1("Raha: ./users file is called on the gateway server and is running the users exe, hence the main function in users.go file")
	// //raha: why?!!! commented next line
	// //log.Fatal("De")

	// //init with deter.toml
	deter := deterFromConfig()
	flag.Parse()
	// // kill old processes
	var wg sync.WaitGroup
	re := regexp.MustCompile(" +")
	// hosts, err := exec.Command("/usr/testbed/bin/node_list", "-e", deter.Project+","+deter.Experiment).Output()
	// if err != nil {
	// 	log.Fatal("Deterlab experiment", deter.Project+"/"+deter.Experiment, "tttt seems not to be swapped in. Aborting.")
	// 	os.Exit(-1)
	// }
	//hosts := "csi-lab-ssh.engr.uconn.edu"
	//todoraha: change this list based on list of VMs or make it dynamic later
	hosts := "192.168.3.220:22 192.168.3.221:22 192.168.3.222:22 192.168.3.223:22"
	hostsTrimmed := strings.TrimSpace(re.ReplaceAllString(string(hosts), " "))
	hostlist := strings.Split(hostsTrimmed, " ")

	doneHosts := make([]bool, len(hostlist))
	log.LLvl1("Found the following hosts:", hostlist)
	//raha: commented
	//if kill {
	log.LLvl1("Cleaning up", len(hostlist), "hosts.")
	//}

	for i, h := range hostlist {
		wg.Add(1)
		go func(i int, h string) {
			defer wg.Done()
			//raha: commented
			//if kill {
			log.LLvl1("Cleaning up host", h, ".")
			//raha commented
			//runSSH(h, "sudo killall -9 simul scp 2>/dev/null >/dev/null")
			s := strings.Split(h, ":")
			hs := s[0]
			runSSH(hs, "kill -9 -1")
			time.Sleep(1 * time.Second)
			// raha commented
			//runSSH(h, "sudo killall -9 simul 2>/dev/null >/dev/null")
			//time.Sleep(1 * time.Second)
			// Also kill all other process that start with "./" and are probably
			// locally started processes
			//runSSH(h, "sudo pkill -9 -f '\\./'")
			//time.Sleep(1 * time.Second)
			// if log.DebugVisible() > 3 {
			// 	log.LLvl1("Cleaning report:")
			// 	_ = platform.SSHRunStdout("", h, "ps aux")
			// }
			//log.Lvl1("Host", h, "cleaned up")
			//} else {
			//log.LLvl1("Raha: skipping: Setting the file-limit higher")
			log.LLvl1("Setting the file-limit higher on", h)
			// Copy configuration file to make higher file-limits
			err := platform.SSHRunStdout("root", h, "sudo cp remote/simul.conf /etc/security/limits.d")
			if err != nil {
				log.Fatal("Couldn't copy limit-file:", err)
			} else {
				log.Lvl1("successfully copied limit-file")
			}
			//}
			doneHosts[i] = true
		}(i, h)
	}
	cleanupChannel := make(chan string)
	go func() {
		wg.Wait()
		log.LLvl1("Done waiting")
		//todoraha: we will need it once I figure out how to use flags!
		cleanupChannel <- "done"
	}()
	select {
	case msg := <-cleanupChannel:
		log.LLvl1("Received msg from cleanupChannel", msg)
	//todoraha now: is it a good wait out time for us?
	case <-time.After(time.Second * 20000):
		for i, m := range doneHosts {
			if !m {
				log.LLvl1("Missing host:", hostlist[i], "- You should run")
				log.LLvl1("/usr/testbed/bin/node_reboot", hostlist[i])
			}
		}
		log.Fatal("Didn't receive all replies while cleaning up - aborting.")
	}
	if kill {
		log.LLvl1("Only cleaning up - returning")
		return
	}
	// ADDITIONS : the monitoring part
	// Proxy will listen on Sink:SinkPort and redirect every packet to
	// RedirectionAddress:SinkPort-1. With remote tunnel forwarding it will
	// be forwarded to the real sink
	//-------------------
	//todoraha: what proxy and monitor port are doing?
	// addr, port := deter.ProxyAddress, uint16(deter.MonitorPort+1)
	// log.LLvl1("Launching proxy redirecting to", addr, ":", port)
	// prox, err := monitor.NewProxy(uint16(deter.MonitorPort), addr, port)
	// if err != nil {
	// 	log.Fatal("Couldn't start proxy:", err)
	// }
	// go prox.Run()
	// log.LLvl1("starting", deter.Servers, "cothorities for a total of", deter.Hosts, "processes.")
	//-------------------
	killing := false
	for i, phys := range deter.Phys {
		log.LLvl1("Launching simul on", phys)
		wg.Add(1)
		go func(phys, internal string) {
			defer wg.Done()
			monitorAddr := deter.MonitorAddress + ":" + strconv.Itoa(deter.MonitorPort)
			log.LLvl1("Starting servers on physical machine ", internal, "with monitor = ",
				monitorAddr)
			// If PreScript is defined, run the appropriate script _before_ the simulation.
			//log.LLvl1("Raha: skipping:run the appropriate script")
			if deter.PreScript != "" {
				log.LLvl1("raha: deter.PreScript running?")
				err := platform.SSHRunStdout("root", phys, "cd remote; sudo ./"+deter.PreScript+" deterlab")
				if err != nil {
					log.Fatal("error deploying PreScript: ", err)
				}
			} else {
				log.LLvl1("raha: deter.PreScript is empty.")
			}
			args := " -address=" + internal +
				" -simul=" + deter.Simulation +
				" -monitor=" + monitorAddr +
				//todoraha
				//" -debug=" + strconv.Itoa(log.DebugVisible()) +
				" -debug= 5" +
				" -suite=" + suite
			log.LLvl1("Args is", args)

			// -----------------------------------------
			// Raha added this part!
			// -----------------------------------------
			// Copy everything over to each vm
			log.LLvl1("Copying over to", phys)
			err := platform.SSHRunStdout("root", phys, "mkdir -p remote")
			if err != nil {
				log.Fatal(err)
			}
			err = platform.Rsync("root", phys, "ssh", "/home/zam20015/remote", "/root/")
			if err != nil {
				log.Fatal(err)
			}
			log.LLvl1("Done copying to VMs")
			// -----------------------------------------
			log.LLvl1("Raha: running ./simul with non-empty simul tag!!!")
			err = platform.SSHRunStdout("root", phys, "cd remote; sudo ./simul "+args)
			// -----------------------------------------
			if err != nil && !killing {
				log.LLvl1("Error starting simul - will kill all others:", err, internal)
				killing = true
				cmd := exec.Command("kill", "-9", "-1")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				var err error
				go func() {
					err = cmd.Run()
				}()
				if err != nil {
					log.Fatal("Raha: CMD: Couldn't killall listening threads:", err)
				} else {
					log.LLvl1("Raha: all listener on VM", internal, "are killed.")
				}
			}
			log.LLvl1("Finished with simul on", internal)
		}(phys, deter.Virt[i])
	}
	// wait for the servers to finish before stopping
	wg.Wait()
	//totdoraha: commented
	//prox.Stop()
}

// Reads in the deterlab-config and drops out if there is an error
func deterFromConfig(name ...string) *platform.Deterlab {
	d := &platform.Deterlab{}
	configName := "deter.toml"
	if len(name) > 0 {
		configName = name[0]
	}
	err := onet.ReadTomlConfig(d, configName)
	_, caller, line, _ := runtime.Caller(1)
	who := caller + ":" + strconv.Itoa(line)
	if err != nil {
		log.Fatal("Couldn't read config in", who, ":", err)
	}
	log.SetDebugVisible(d.Debug)
	return d
}

// Runs a command on the remote host and outputs an eventual error if debug level >= 3
func runSSH(host, cmd string) {
	if _, err := platform.SSHRun("root", host, cmd); err != nil {
		// ToDoRaha: it gives error but let's wait for repeating the open listener err
		log.Lvl1("Host", host, "got error", err.Error(), "while running", cmd)
	} else {
		log.Lvl1("Host", host, "cleaned up")
	}
}
