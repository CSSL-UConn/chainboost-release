
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