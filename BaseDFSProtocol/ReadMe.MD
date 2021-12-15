## Base DFS protocol ##


Types of Messages:
--------------------------------------------
- each server who is elected as leader (few servers in each round) send a msg of &NewLeader{Leaderinfo: bz.Name(), RoundNumber: bz.roundNumber} to root node.
- each round, the root node send a msg of &NewRound{Seed:  seed, Power: power} to all servers including the target server's power
- in the bootstrapping phase, the root node send a message to all servers containing protocol config parameters.
	&HelloBaseDFS{
					Timeout:                  bz.timeout,
					PercentageTxPay:          bz.PercentageTxPay,
					RoundDuration:            bz.RoundDuration,
					BlockSize:                bz.BlockSize,
					SectorNumber:             bz.SectorNumber,
					NumberOfPayTXsUpperBound: bz.NumberOfPayTXsUpperBound,
				}
- timeOut
