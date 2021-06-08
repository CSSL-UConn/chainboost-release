 # Base DFS protocol:
 
## Asumptions:
1) market matching is done
(for each client we have one server who has stored his client's file
and the storage contract is fixed for all - fixed payment)
2) clients have created their escrows -
a fixed amount of money is in some addresses (clients addresses)

## The BaseDFSProtocol goes as follows:
1- a server broadcast a por tx
(chanllenge's randomness comes from hash of the latest mined block)
2) all miners (who recieve this tx) verify the por tx + add a fixed payment tx +
		keep both in their current block
3) (each epoch) miners check the predefined leader election mechanism
		 to see if they are the leader
4) the leader broadcast his block, consisting por and payment tx.s
		and his election proof
5) all miners (who recieve this tx) verify proposed block and
		add it to their chain

### Types of Messages:
1) por tx
2) proposed block

### The simple version:
we assume one server broadcast his por tx and all servers on recieving this tx will verify and add it to their prepared block and then all server run leader election to check if they are leader, then the leader add his proof and broadcast his prepared block and all servers verify and append his block to their blockchain

# To Dos:
 - [ ] Clean Protocol structure
 - [ ] Resolve NewbaseDFSRootProtocol issue
 - [ ] onet.Broadcast returns []error, I want error!
 - [ ] Periodic action procedure (a simple storage server: por tx: each t1 + the miners: checkleadership: each epoch)
 - [ ] Now tree root is broadcasting a por tx, later we want peridic broadcasting of por txs by random servers in the roster
 - [ ] Creating por tx function (a simple storage server)
 - [ ] Verying por tx function (the miners)
 - [ ] Adding por tx to temp block (not new block)
 - [ ] Verifying block function (the miners)
 - [ ] Appending block function (the miners)
 - [ ] Creating leadership proof function (the leader)
 - [ ] Checking leadership function (the miners)
 - [ ] Creating new block and broadcasting it (the leader)
