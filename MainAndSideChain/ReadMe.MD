# Main and Side Chain #

Blockchain is stored in two layers: 
- in a file on hardware 
- and in the storage of a root node in the protocol. 
So when each node sends and receives a message from the root node, it is equal to accessing the blockchain [^1], reading from it or writing in it (in the case of the next block, if having a valid leadership proof).

## Main Chain protocol ##

Main Chain consensus:
--------------------------

1. On each round, all nodes check for their leadership by comparing their current power with **their own vrf output**.

2. if it was bigger than their random output they send a message to bc-layer1 announcing their leadership.

3. bc-layer 1 can verify their leadership argument [^2] and the first leader that has announced her leadership will get to be the round’s leader. [^3]

4. Bc-layer1 update the bc-layer2 using the config params, the selected leader’s info Etc.

5. once the round duration is finished, it announces the next round’s start by directly sending a message to all nodes. This is equal to having all nodes observe bc and once the new block is added (after round duration) they start the next round.

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
	MCRoundDuration:          bz.MCRoundDuration,
	MainChainBlockSize:       bz.MainChainBlockSize,
	SideChainBlockSize:		  bz.SideChainBlockSize,
	SectorNumber:             bz.SectorNumber,
	NumberOfPayTXsUpperBound: bz.NumberOfPayTXsUpperBound,
	SimulationSeed:			  bz.SimulationSeed,
	nbrSubTrees:			  bz.nbrSubTrees,
	threshold:				  bz.threshold,
	SCRoundDuration:		  bz.SCRoundDuration,
	CommitteeWindow:		  bz.CommitteeWindow,
	MCRoundPerEpoch:		  bz.MCRoundPerEpoch,
	SimState:				  bz.SimState,
}
```

## Side Chain protocol ## 
Side chain protocol is a BlS protocol that its code is brought from Cosi module from Cothority. [^4]

Types of Messages:
--------------------------------------------
Other than the internal messages that are sent through a round of CoSi (signs, announcments, final signature, etc.) , we have added 2 more messages in side chain protocol:

- after each full run of blscosi by a leader node, that node send a message to the root node announcing a successful completion of one round of side chain.
```
&LtRSideChainNewRound
```
- after finishing the prosseccing of each side chain's round completion by root node (such as updating blockchains or increasing round number or checking for next leader and committee), the root node send a message to next side chain's round, triggering him to run the next round.
```
&RtLSideChainNewRound
```


## Main Chain Blockcahin ##

- ServAgr stands for `service agreement`
- por stands for `proof of retrievibility`

There are 6 sheets, namely MarketMatching, FirstQueue, SecondQueue, RoundTable, PowerTable, and Overall Evaluation.

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
- `Overall Evaluation`: the overall results up until each round [^5]


## Side Chain Blockcahin ##

There are 3 sheets, namely FirstQueue, RoundTable, and Overall Evaluation.[^4]

### Assumptions, Simplifications ###
- [ ] we may need to say which model (account based or UTXO) are we adopting in the implmentation
- [ ] a simplifying assumption we have for committee is that in each committee, we have one instance (one node) for each committee member and not more! even if they have been powerfull and being selected as main chain's leader multiple time during last epoch.
- [ ] leader election in mainchain



<!--FootNote-->
[^1]: assumption: All nodes have a same view of the blockchain.
[^2]: verify vrf is implemented but it is not added here
[^3] if the SimState parameter in the config file be set to 1, the side chain protocol will not be run atall and the side chain blockchain file will be empty (i.e. there will be no side chain blockchain)
[^4] these sheets are updated each round so, after running the simulation, we can track the blockchain's progress while running. try opening the file and closing if you are using microsoft or just refreshing the file if you are opening it in visual studio code IDE.
[^5] the simplifying assumption here is that in each round multiple nodes may have bigger power than their vrf output hence get selected, different systems have different methods to resolve this multiple branch issue, some are trying to have a single winning node, some define a rule to pick one of them. Here for simpllicity we accept thefirst announcement (the first leader that has announce his leadership) as the round's leader.
<!--FootNote-->
