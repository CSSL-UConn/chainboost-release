## Base DFS protocol ##


Types of Messages:
--------------------------------------------
- each server who is elected as leader (few servers in each round) send the following msg to root node: 
 ```
 &NewLeader{
	 Leaderinfo: bz.Name(), 
	 RoundNumber: bz.roundNumber
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
	RoundDuration:            bz.RoundDuration,
	BlockSize:                bz.BlockSize,
	SectorNumber:             bz.SectorNumber,
	NumberOfPayTXsUpperBound: bz.NumberOfPayTXsUpperBound,
}
```
- timeOut: terminate the simulation
