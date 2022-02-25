
# ChainBoost #
<p align="center"><img width="650" height="200" src="./MainAndSideChain/chainboost.png" alt="ChainBoost logo"></p>

ChainBoost's official implementation in Go.

ChainBoost is a ...

## Getting Started ##
note: running on an OS other than IOS needs a change in c extention config code

- Install Go
- Clone or Downloade the ChainBoost's source code from Git <https://github.com/chainBstSc/basedfs>
- Open a terminal in the directory where the folder basedfs is located
- run the following command: 
```
/usr/local/go/bin/go test -timeout 50000s -run ^TestSimulation$ github.com/basedfs/simul/manage/simulation
```

- this will call the TestSimulation function in the file: ([simul_test.go](https://github.com/chainBstSc/basedfs/blob/master/simul/manage/simulation/simul_test.go))


raha@R-MacBook-Pro basedfs % /usr/local/go/bin/go test -timeout 300000s -run ^TestSimulation$ github.com/basedfs/simul/manage/simulation


- the stored blockchain in Excel file "mainchainbc.xlsx"  can be found under the `build` directory that is going to be created after simulation run[^3]
- in the case of debugging the following code in ([simul_test.go](https://github.com/chainBstSc/basedfs/blob/master/simul/manage/simulation/simul_test.go)) indicates the debug logging level, with 0 being the least logging and 5 being the most (every tiny detail is logged in this level)
```
log.SetDebugVisible(1)
```

## Config File ##

Config File "BaseDFS.toml" is located under the following directory:
([BaseDFS.toml](https://github.com/chainBstSc/basedfs/blob/master/simul/manage/simulation/BaseDFS.toml))


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


We used latest version of [Onet](https://github.com/dedis/onet/tree/v3.2.9) (v.3.2.9) at the time for network, simulation, and communication modules 
as well as [Cosi](https://github.com/dedis/cothority) module from Cothority. 
We used [Kyber](https://github.com/dedis/kyber) for advanced cryptographic primitives.


<!--FootNote-->
[^1]: there may be some rounds that there is no leader for them, an empty block will be added to the blockchain in those rounds and the information of the root node (blockchain layer 1) is added (it can be removed) as the round leader but all the other columns are empty. in these rounds transactions will be added normally to the queue but no transaction is removed bcz the block is empty.
[^2]: when in a round, some transactions should wait in a queue (i.e. the allocated space for  that transaction is full) and are submitted in another round, the average wait of that queue in the round that those transactions get to be submitted increases.
<!--FootNote-->


- we may need to say which model (account based or UTXO) are we adopting in the implmentation
- a simplifying assumption we have for committee is that in each committee, we have one instance (one note) for each committee member and not more! even if they have been powerfuil and being selected as main chain's leader multiple time during last epoch
