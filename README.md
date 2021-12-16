
# ChainBoost #
<p align="center"><img width="650" height="200" src="./BaseDFSProtocol/chainboost.png" alt="ChainBoost logo"></p>

ChainBoost's official implementation in Go.

ChainBoost is a ...

## Getting Started ##
note: running on an OS other than IOS needs a change in c extention config code

- Install Go
- Clone or Downloade the ChainBoost's source code from Git <https://github.com/chainBstSc/basedfs>
- Open a terminal in the directory where the folder basedfs is located
- run the following command: 
    - "/usr/local/go/bin/go test -timeout 50000s -run ^TestSimulation$ github.com/basedfs/simul/manage/simulation"
    - this will call the TestSimulation function in the file: ([simul_test.go](https://github.com/chainBstSc/basedfs/blob/master/simul/manage/simulation/simul_test.go))
- the stored blockchain in Excel file "centralbc.xlsx"  can be found under the `build` directory that is going to be created after simulation run[^3]
- in the case of debugging the code

```
log.SetDebugVisible(1)
```

indicates the debug logging level, with 0 being the least logging and 5 being the most (every tiny detail is logged in this level)



## Config File ##

Config File "BaseDFS.toml" is located under the following directory:
([BaseDFS.toml](https://github.com/chainBstSc/basedfs/blob/master/simul/manage/simulation/BaseDFS.toml))


## To Change the Configs ##
- to change number of servers, change two values: 1- `Hosts` and 2- `Nodes` - with a same number :)
- `BlockSize` is the maximum block size (in Byte) allowed in each round (the submitted block may be less than this size based on the available transactions in the queues)[^1]
- `DistributionMeanFileSize` and `DistributionVarianceFileSize` are specifying the mean and variance of the Normal distribution used to generate file-sizes in the contracts
- `DistributionMeanContractDuration` and `DistributionVarianceContractDuration` is the same for contracts' duration
- `DistributionMeanInitialPower` and DistributionVarianceInitialPower is the same for the intial power we assign to each server
- `SectorNumber` is the number of sectors in each block of file with impact the por transaction size
- `PercentageTxPay` the block size percentage allocated for regular payment transactions (if regular payment txs are less, other types of txs will take its space)
- `NumberOfPayTXsUpperBound` the upper bound for a random number of regular payment transactions issued in each round
- `ProtocolTimeout` is the time that we want the protocol to stop after it (in seconds)
- `RoundDuration` the time interval between each round (in seconds)
- `SimulationSeed` 


## Blockcahin ##
There are 5 sheets, namely MarketMatching, FirstQueue, SecondQueue, and RoundTable, and PowerTable


- `MarketMatching`: the overall information about the market matching
    - about the servers: IP, 
    - about the contract: ID, duration, File size, and starting round#, isPublished (if a contract get expired, the column published is set to 0 until its poropose and commit transaction get submitted to the blockchain again)
- `PowerTable`: A matrix of each server's added power in each round
- `FirstQueue`: there are 5 types of trransactions in there
    - `propose contract': including the information of teh contract and the client's payment for it
    - `commit contract`: in which the server commits to the contract id already published by the client
    - `por`: for each active (not expired) contract each server issue ane por
    - `storage payment`: after the contract duration pass and a contract expires, this transaction is assued to pay for the service
- `SecondQueue`: the queue of regular payment transactions
- `RoundTable`: the overall information of the blockchain including:
    - each round's seed
    - the added block size
    - IP of the leader in each round
    - number of each transaction type that is submitted in each round
    - `TotalNumTxs`: total number of all submitted transactions in each round
    - the time that each round has started
    - `AveWait-RegPay` and `AveWait-OtherTxs`: the average wait time in each round for regular payment and other types of transactions[^2]
    - `RegPaySpaceFull` and `BlockSpaceFull`: 1 indicates the allocated space for regular payment is full /  the block space is full
- `Overall Evaluation`: the overall results up until each round[^3]


## Project Layout ##

`ChainBoost` is split into various subpackages.

The following packages provide core functionality to ..., as well as other tools and commands:

--------------------------------------------------------------------------------------------------
  - `crypto` contains the cryptographic constructions we're using for hashing,
    signatures, and VRFs. There are also some Algorand-specific details here
    about spending keys, protocols keys, one-time-use signing keys, and how they
    relate to each other.
  -   `...`
--------------------------------------------------------------------------------------------------

# Base Distributed File System Network

We used latest version of [Onet](https://github.com/dedis/onet/tree/v3.2.9) (v.3.2.9) for network, simulation, and communication modules 
as well as [Cosi](https://github.com/dedis/cothority) module from Cothority 

Onet's documents can be find under following link:
<https://github.com/dedis/onet/blob/master/README.md>

The Overlay-network (Onet) is a library for simulation and deployment of
decentralized, distributed protocols. This library offers a framework for
research, simulation, and deployment of crypto-related protocols with an emphasis
on decentralized, distributed protocols. It offers an abstraction for tree-based
communications between thousands of nodes and it is used both in research for
testing out new protocols and running simulations, as well as in production to
deploy those protocols as a service in a distributed manner.




<!--FootNote-->
[^1]: there may be some rounds that there is no leader for them, an empty block will be added to the blockchain in those rounds and the information of the root node (blockchain layer 1) is added (it can be removed) as the round leader but all the other columns are empty. in these rounds transactions will be added normally to the queue but no transaction is removed bcz the block is empty.
[^2]: when in a round, some transactions should wait in a queue (i.e. the allocated space for  that transaction is full) and are submitted in another round, the average wait of that queue in the round that those transactions get to be submitted increases.
[^3]: these sheets are updated each round so, after running the simulation, we can track the blockchain's progress while running. try opening the file and closing if you are using microsoft or just refreshing the file if you are opening it in visual studio code IDE.
<!--FootNote-->
