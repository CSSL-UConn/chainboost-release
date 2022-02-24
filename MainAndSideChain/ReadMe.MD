## Base DFS protocol ##


Types of Messages:
--------------------------------------------
- each server who is elected as leader (few servers in each round) send the following msg to root node: 
 ```
 &NewLeader{
	 Leaderinfo: bz.Name(), 
	 MCRoundNumber: bz.MCRoundNumber
 } 
 ```
- each round, the root node send the following msg to all servers including the target server's power
```
&NewRound{
	Seed:  seed, 
	Power: power
}
```
- in the bootstrapping phase, the root node send the following message to all servers containing protocol config parameters.
```	
&HelloBaseDFS{
	Timeout:                  bz.timeout,
	PercentageTxPay:          bz.PercentageTxPay,
	MCRoundDuration:            bz.MCRoundDuration,
	BlockSize:                bz.BlockSize,
	SectorNumber:             bz.SectorNumber,
	NumberOfPayTXsUpperBound: bz.NumberOfPayTXsUpperBound,
	SimulationSeed:			  bz.SimulationSeed,
	nbrSubTrees:			  bz.nbrSubTrees,
	threshold:				  bz.threshold,
	SCRoundDuration:			  bz.SCRoundDuration,
	CommitteeWindow:		  bz.CommitteeWindow,
	EpochCount:					bz.EpochCount,
	SimState:					bz.SimState,
}
```
- timeOut: terminate the simulation


## Blockcahin ##

- ServAgr stands for `service agreement`
- por stands for `proof of retrievibility`

There are 5 sheets, namely MarketMatching, FirstQueue, SecondQueue, and RoundTable, and PowerTable


- `MarketMatching`: the overall information about the market matching
    - about the servers: IP, 
    - about the ServAgr: ID, duration, File size, and starting round#, isPublished (if a ServAgr get expired, the column published is set to 0 until its poropose and commit transaction get submitted to the blockchain again)
- `PowerTable`: A matrix of each server's added power in each round
- `FirstQueue`: there are 5 types of trransactions in there
    - `propose ServAgr': including the information of teh ServAgr and the client's payment for it
    - `commit ServAgr`: in which the server commits to the ServAgr id already published by the client
    - `por`: for each active (not expired) ServAgr each server issue ane por
    - `storage payment`: after the ServAgr duration pass and a ServAgr expires, this transaction is assued to pay for the service
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





<!--FootNote-->
[^3]: these sheets are updated each round so, after running the simulation, we can track the blockchain's progress while running. try opening the file and closing if you are using microsoft or just refreshing the file if you are opening it in visual studio code IDE.
<!--FootNote-->