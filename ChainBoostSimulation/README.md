
## Getting Started ##
note: running on an OS other than IOS needs a change in c extention config code

- Install Go
- Clone or Downloade the ChainBoost's source code from Git <https://github.com/chainBstSc/basedfs>
- Open a terminal in the directory where the folder basedfs is located
- run the following command: 
    - "/usr/local/go/bin/go test -timeout 50000s -run ^TestSimulation$ github.com/basedfs/simul/manage/simulation"
    - this will call the TestSimulation function in the file: ([simul_test.go](https://github.com/chainBstSc/basedfs/blob/master/simul/manage/simulation/simul_test.go))
- the stored blockchain in Excel file "mainchainbc.xlsx"  can be found under the `build` directory that is going to be created after simulation run[^3]
- in the case of debugging the following code in ([simul_test.go](https://github.com/chainBstSc/basedfs/blob/master/simul/manage/simulation/simul_test.go)) indicates the debug logging level, with 0 being the least logging and 5 being the most (every tiny detail is logged in this level)
```
log.SetDebugVisible(1)
```

## Config File ##

Config File "BaseDFS.toml" is located under the following directory:
([BaseDFS.toml](https://github.com/chainBstSc/basedfs/blob/master/simul/manage/simulation/BaseDFS.toml))

## To Change the Configs ##
- to change number of servers, change two values: 1- `Hosts` and 2- `Nodes` - with a same number :)
- `BlockSize` is the maximum block size (in Byte) allowed in each round (the submitted block may be less than this size based on the available transactions in the queues)[^1]
- `FileSizeDistributionMean` and `FileSizeDistributionVariance` are specifying the mean and variance of the Normal distribution used to generate file-sizes in the ServAgrs
- `ServAgrDurationDistributionMean` and `ServAgrDurationDistributionVariance` is the same for ServAgrs' duration
- `InitialPowerDistributionMean` and `InitialPowerDistributionVariance` is the same for the intial power we assign to each server
- `SectorNumber` is the number of sectors in each block of file with impact the por transaction size
- `PercentageTxPay` the block size percentage allocated for regular payment transactions (if regular payment txs are less, other types of txs will take its space)
- `NumberOfPayTXsUpperBound` the upper bound for a random number of regular payment transactions issued in each round
- `SimulationRounds` is number of mainchain's round that we want the protocol to stop after it
- `MCRoundDuration` the time interval between each round (in seconds)
- `SimulationSeed` 
- `nbrSubTrees`
- `threshold`
- `SCRoundDuration`
- `EpochCount`
- `CommitteeWindow`
- `SimState`

## PreScript file ##

About preScript file which runs before simulation on all servers:
In localHost.go file (it should be in another file when the simulation is distributed), in Deploy function, at one point, it says:
// Check for PreScript and copy it to the deploy-dir

In the simulation config file, there is a line that specify preScript file:
```
PreScript = "testPreScript.toml"
```

Now I have added the following address (to the code) to look for it in there: "/Users/raha/Documents/GitHub/basedfs/simul/platform/"
The code copy this file to where it call deploy-dir which is: "/Users/raha/Documents/GitHub/basedfs/simul/manage/simulation/build"

Later, again in localhost.go file, in the Start() function, it run the preScript file:
 
The code:
```
exec.command("sh", -c , ... ) 
```
will run the following shell-script.

//To create a file in each server: 

```
touch mainbchainbc.csv
```


This file will be created in the deploy-dir

To write/read/update this file:

In the initialization phase, the data comes from the simulation “config file”
In build.go, after deploy.start() (which is where the prescript is decoded and runned and the centralbc file is generated and its directory is deploy-dir which is “/simul/manage/simulation/build/”) we have:
```
err := deployP.Start()
// Raha: initializing central blockchain -------------------------
   blockchain.InitializeCentralBC(rc.Get("RoundDuration"),
           rc.Get("PercentageTxPay"),
           rc.Get("DistributionMeanFileSize"), rc.Get("DistributionVarianceFileSize"),
           rc.Get("DistributionMeanContractDuration"), rc.Get("DistributionVarianceContractDuration"),
           rc.Get("DistributionMeanInitialPower"), rc.Get("DistributionVarianceInitialPower"),
           rc.Get("Nodes"),
           rc.Get("BlockSize"))
// --------------------------------------------
```
to add other system-wide configurations params to existing Onet’s system-wide configurations params, I added few lines of codes in following files:
1- localhost.go
2- build.go
3- platform.go

- [ ] For future reference: The modification I made in the Onet’s module, I start them with the “Raha” keyword. So they are easy to find!

3- In platform.go I modified the  "Config struct" and added our params.
```
type Config struct {
   // string denoting the group used for simulations
   // XXX find ways to remove that "one suite" assumption
   Suite       string
   MonitorPort int
   Debug       int
   // raha: adding some other system-wide configurations
   RoundDuration      string
   PercentageTxPay    string
   BlockSize string
}
```

This struct is used for all platforms (local host, mininet, deterlab). This means we should not have to make any change when changing platform from local host to mininet (multiple servers)

2- In build.go I read these params from our protocol’s config file (.toml) and send them to platform.Configure function, which is again a platform independent change:


```
deployP.Configure(&platform.Config{
MonitorPort: monitorPort,
Debug:       log.DebugVisible(),
Suite:       runconfigs[0].Get("Suite"),
// raha: adding some other system-wide configurations
RoundDuration:      runconfigs[0].Get("RoundDuration"),
PercentageTxPay:    runconfigs[0].Get("PercentageTxPay"),
BlockSize:          runconfigs[0].Get("BlockSize"),
       })
```


1- In localhost.go, first I added those params to “Localhost struct” 
```
type Localhost struct {
...
   // raha: adding some other system-wide configurations
   RoundDuration      string
   PercentageTxPay    string
   BlockSize string
}
```


and then modified the
```
 “func (d *Localhost) Configure(pc *Config)” function, so when it is called in build.go, it initializes its params with the added input parameters:

// Configure various internal variables
func (d *Localhost) Configure(pc *Config) {
   d.Lock()
   defer d.Unlock()
   pwd, _ := os.Getwd()
   d.runDir = pwd + "/build"
   os.RemoveAll(d.runDir)
   log.ErrFatal(os.Mkdir(d.runDir, 0770))
   d.Suite = pc.Suite
   // ------------------------------
   // raha: adding some other system-wide configurations
   d.RoundDuration = pc.RoundDuration
   d.PercentageTxPay = pc.PercentageTxPay
   d.BlockSize = pc.BlockSize
   // ------------------------------
   d.localDir = pwd
   d.debug = pc.Debug
   d.running = false
   d.monitorPort = pc.MonitorPort
   if d.Simulation == "" {
       log.Fatal("No simulation defined in simulation")
   }
   log.Lvl3(fmt.Sprintf("Localhost dirs: RunDir %s", d.runDir))
   log.Lvl3("Localhost configured ...")
}
```
So, what we have at the end is:
-------------------------------------------
1- we have added the params in config file
2- in StartBiuld() function, the modified platform.config structure get initialized by reading the params passed from the config file
3- in (*localhost).Configure function, the values are passed to the modified localhost structure
4- in blockchain.InitializeCentralBC, which is called in build.go, RunTest function (before starting the protocol yet!), “the config params that are needed for initializing the centralbc file” are passed to the blockchain.InitializeCentralBC function from (modified?) *platform.RunConfig structure
5- the function blockchain.InitializeCentralBC uses these params to initialize the centralbc file
6- in localhost.go, in function (*localhost).Start, “the config params that are needed inside the protocol” are passed to function Simulate() (function simulate has modified input params)
7- in function simulate(), after creating an instance of our BasedDFSProtocol, the passed params are sent to the protocol structure (our protocol has these params in its structure and get initialized here) 
8- but note that just the first node who run the protocol has these params in the protocol structure initialized, so in order to have other nodes start running the protocol (and get initialized with these params) we sent a message to all nodes and use a HelloBaseDFS structure that carries these params in it in the function HelloBaseDFS() which is called in the Start() function by the first node. So, when each node receives the message, in the Dispatch() function, the passed params are sent to their protocol structure and their protocol gets initialized too.
The End. :)


-------------
- [ ] Note that the cpu time of blockchain’s two layer (RAM and Storage) communication is not counted/ eliminated from the protocol’s latency.
- [ ] If we use ec2 for experiment, we should be careful about time zones in measuring time for latency measurement.
