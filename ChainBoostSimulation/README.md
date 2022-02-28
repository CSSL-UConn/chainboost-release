
## Getting Started ##

- Install Go
- Clone or Downloade the ChainBoost's source code from Git <https://github.com/chainBstSc/basedfs>
- Open a terminal in the directory where the folder basedfs is located
- run the following command: 
```
    /usr/local/go/bin/go test -timeout 50000s -run ^TestSimulation$ github.com/basedfs/simul/manage/simulation
```

- this will call the TestSimulation function in the file: ([simul_test.go](https://github.com/chainBstSc/basedfs/blob/master/simul/manage/simulation/simul_test.go))
- the stored blockchain in Excel file "mainchainbc.xlsx" and "sidechainbc.xlsx" can be found under the `build` directory that is going to be created after simulation run[^3]
- in the case of debugging the following code in ([simul_test.go](https://github.com/chainBstSc/basedfs/blob/master/simul/manage/simulation/simul_test.go)) indicates the debug logging level, with 0 being the least logging and 5 being the most (every tiny detail is logged in this level)
```
log.SetDebugVisible(1)
```

## Config File ##

Config File [BaseDFS.toml](https://github.com/chainBstSc/basedfs/blob/master/simul/manage/simulation/BaseDFS.toml) determines the simulation properties.

## To Change the Configs ##
- to change number of servers, change two values: 1- `Hosts` and 2- `Nodes` - with a same number :)
- `MainChainBlockSize` is the maximum block size (in Byte) allowed in each main chian's round (the submitted block may be less than this size based on the available transactions in the queues)[^1]
- `SideChainBlockSize` is the maximum block size (in Byte) allowed in each side chain's round
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

## How Transactions are Generated in Queue ##

in sheet “market matching”, the ContractPublished == 1 means that:
a “TxEscrow” transaction (this should be modified later) has been submitted (added to a block) for this contract. the column “started round number” says on what round this transaction has been submitted (i.e. the contract has started being active)
in sheet “market matching”, the ContractPublished == 0 means that:
The contract is expired (or just in first round not started yet)
when this happens, a “TxStoragePayment” transaction will be sent to the transaction queue
on the next round, with ContractPublished == 0, a “TxEscrow” transaction will be sent to the transactions queue
and again, when the “TxEscrow” transaction leave the queue, the ContractPublished will be set to 1
in sheet “market matching”, for each server (/contract) that the column ContractPublished == 1 a “TxPor” transaction will be sent to the transactions queue
Note: for now, we are assuming that regardless of file size, each server have one client and will issue one por transaction each round
Note: regular payment transactions have the priority to take the specified percentage of block size (specified in config file) and they will. So if based on the number of regular payment transactions in their queue, they take less than their allocated size, the rest of block size is going to be spent on other types of transactions.

## "propose contract” & “Commit Contract” transactions ##
“escrow creation” transaction is referencing a “contract” transaction (including a payment) and is being considered to be issued by the client .
In the “contract” transaction I had considered commitment from both side, client and server.
The point is that we can imagine two scenario:
1- they have generated a “contract” transaction together, in a sense that the client has provided some part of its information (price, duration, file tag, commitment) and have passed it to the server and then the server has signed it (commitment) and then the transaction has been issued and submitted or
2- a client issue a “propose contract” transaction including all the mentioned information, plus the escrow payment. and then the server issue a “commit contract” transaction, referencing the “propose contract” transaction. (the escrow is not locked until a “commit transaction” is submitted on top of it)


- note: running on an OS other than IOS needs a change in C extention config code

-------------
- [ ] Note that the cpu time of blockchain’s two layer (RAM and Storage) communication is not counted/ eliminated from the protocol’s latency.
- [ ] If we use ec2 for experiment, we should be careful about time zones in measuring time for latency measurement.
