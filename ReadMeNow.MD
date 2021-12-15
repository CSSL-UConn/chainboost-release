ChainBoost
====================
ChainBoost's official implementation in Go.

note: running on an OS other than IOS needs a change in c extention config code

## Getting Started ##
1- Install Go
2- Clone or Downloade the ChainBoost's source code from Git <https://github.com/chainBstSc/basedfs>
3- Open a terminal in the directory where the folder basedfs is located
4- run the following command: 
    - "/usr/local/go/bin/go test -timeout 50000s -run ^TestSimulation$ github.com/basedfs/simul/manage/simulation"
    - this will call the TestSimulation function in the file: ([simul_test.go](https://github.com/chainBstSc/basedfs/blob/master/simul/manage/simulation/simul_test.go))
5- the stored blockchain in Excel file "centralbc.xlsx"  can be found under the `build` directory that is going to be created after simulation run



## Config File ##

Config File "BaseDFS.toml" is located under the following directory:
([BaseDFS.toml](https://github.com/chainBstSc/basedfs/blob/master/simul/manage/simulation/BaseDFS.toml))


This is how config file looks like:
----------------------------------------------------


Servers = 1 
Simulation = "BaseDFS"
BF = 2
Rounds = 0
RunWait = "600000s"
Suite = "Ed25519"
PreScript = "testPreScript.toml"
RoundDuration = 5
ProtocolTimeout = 100
timeout = 500
PercentageTxPay = 30
DistributionMeanFileSize = 100
DistributionVarianceFileSize = 5
DistributionMeanContractDuration = 10
DistributionVarianceContractDuration = 50
DistributionMeanInitialPower = 900000000000000000
DistributionVarianceInitialPower = 9000000000000000
Nodes = 50
BlockSize = 100000
SectorNumber = 1
NumberOfPayTXsUpperBound = 100

Hosts
50


----------------------------------------------------
## To Change the Configs ##
- to change number of servers, change two values: 1- `Hosts` and 2- `Nodes` - with a same number :)
- `BlockSize` is the maximum block size (in Byte) allowed in each round (the submitted block may be less than this size based on the available transactions in the queues)
- `DistributionMeanFileSize` and `DistributionVarianceFileSize` are specifying the mean and variance of the Normal distribution used to generate file-sizes in the contracts
- `DistributionMeanContractDuration` and `DistributionVarianceContractDuration` is the same for contracts' duration
- `DistributionMeanInitialPower` and DistributionVarianceInitialPower is the same for the intial power we assign to each server
- `SectorNumber` is the number of sectors in each block of file with impact the por transaction size
- `PercentageTxPay` the block size percentage allocated for regular payment transactions (if regular payment txs are less, other types of txs will take its space)
- `NumberOfPayTXsUpperBound` the upper bound for a random number of regular payment transactions issued in each round
- `ProtocolTimeout` is the time that we want the protocol to stop after it (in seconds)
- `RoundDuration` the time interval between each round (in seconds)


## Blockcahin ##
There are 5 sheets, namely MarketMatching, FirstQueue, SecondQueue, and RoundTable, and PowerTable


- `MarketMatching`: the overall information about the market matching 
    - about the servers: IP, about the contract: ID, duration, File size, and starting round#, isPublished (if a contract get expired, the column published is set to 0 until its poropose and commit transaction get submitted to the blockchain again)
- `PowerTable`
    - A matrix of each server's added power in each round
-`FirstQueue`
    - there are 5 types of trransactions in there
        - propose contract (including the information of teh contract and the client's payment for it)
        - commit contract: in which the server commits to the contract id already published by the client
        - por: for each active (not expired) contract each server issue ane por
        - storage payment: after the contract duration pass and a contract expires, this transaction is assued to pay for the service
-`SecondQueue`
    - the queue of regular payment transactions
-`RoundTable`
    - the overall information of the blockchain including:
        - each round's seed
        - the added block size
        - IP of the leader in each round
        - number of each transaction type that is submitted in each round
        - `TotalNumTxs`: total number of all submitted transactions in each round
        - the time that each round has started
        - `AveWait-RegPay` and `AveWait-OtherTxs`: the average wait time in each round for regular payment and other types of transactions
        - `RegPaySpaceFull` and `BlockSpaceFull`: 1 indicates the allocated space for regular payment is full /  the block space is full

Note that:
------------
- there may be some rounds that there is no leader for them, an empty block will be added to the blockchain in those rounds and the information of the root node (blockchain layer 1) is added (it can be removed) as the round leader but all the other columns are empty. in these rounds transactions will be added normally to the queue but no transaction is removed bcz the block is empty.
- when in a round, some transactions should wait in a queue (i.e. the allocated space for  that transaction is full) and are submitted in another round, the average wait of that queue in the round that those transactions get to be submitted increases.


