
for now check ([ReadMeNow.MD].../basedfs/ReadMeNow.MD) file.



<!-- [![Build Status](https://travis-ci.org/dedis/onet.svg?branch=master)](https://travis-ci.org/dedis/onet)
[![Go Report Card](https://goreportcard.com/badge/github.com/dedis/onet)](https://goreportcard.com/report/github.com/dedis/onet)
[![Coverage Status](https://coveralls.io/repos/github/dedis/onet/badge.svg)](https://coveralls.io/github/dedis/onet)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/11ec79aa77fe41748edfdfcd55e92fab)](https://www.codacy.com/manual/nkcr/onet?utm_source=github.com&utm_medium=referral&utm_content=dedis/onet&utm_campaign=Badge_Grade) -->

ChainBoost
====================
ChainBoost's official implementation in Go.

ChainBoost is a ...

## Getting Started ##
...

## Project Layout ##

`ChainBoost` is split into various subpackages.

The following packages provide core functionality to ..., as well as other tools and commands:

--------------------------------------------------------------------------------------------------
  - `crypto` contains the cryptographic constructions we're using for hashing,
    signatures, and VRFs. There are also some Algorand-specific details here
    about spending keys, protocols keys, one-time-use signing keys, and how they
    relate to each other.
  - `config` holds configuration parameters.  These include parameters used
    locally by the node as well as parameters that must be agreed upon by the
    protocol.
  - `data` defines various types used throughout the codebase.
     - `basics` hold basic types such as MicroAlgos, account data, and
       addresses.
     - `account` defines accounts, including "root" accounts (which can
       spend money) and "participation" accounts (which can participate in
       the agreement protocol).
     - `transactions` define transactions that accounts can issue against
       the Algorand state.  These include standard payments and also
       participation key registration transactions.
     - `bookkeeping` defines blocks, which are batches of transactions
       atomically committed to Algorand.
     - `pools` implement the transaction pool.  The transaction pool holds
       transactions seen by a node in memory before they are proposed in a
       block.
     - `committee` implements the credentials that authenticate a
       participating account's membership in the agreement protocol.
  - `ledger` ([README](ledger/README.md)) contains the Algorand Ledger state
    machine, which holds the sequence of blocks.  The Ledger executes the state
    transitions that result from applying these blocks.  It answers queries on
    blocks (e.g., what transactions were in the last committed block?) and on
    accounts (e.g., what is my balance?).
  - `protocol` declares constants used to identify protocol versions, tags for
    routing network messages, and prefixes for domain separation of
    cryptographic inputs.  It also implements the canonical encoder.
  - `network` contains the code for participating in a mesh network based on
    WebSockets. Maintains connection to some number of peers, (optionally)
    accepts connections from peers, sends point to point and broadcast messages,
    and receives messages routing them to various handler code
    (e.g. agreement/gossip/network.go registers three handlers).
     - `rpcs` contains the HTTP RPCs used by `algod` processes to query one
       another.
  - `agreement` ([README](agreement/README.md)) contains the agreement service,
    which implements Algorand's Byzantine Agreement protocol.  This protocol
    allows participating accounts to quickly confirm blocks in a fork-safe
    manner, provided that sufficient account stake is correctly executing the
    protocol.
  - `node` integrates the components above and handles initialization and
    shutdown.  It provides queries into these components.

`daemon` defines the two daemons which provide Algorand clients with services:

  - `daemon/algod` holds the `algod` daemon, which implements a participating
    node.  `algod` allows a node to participate in the agreement protocol,
    submit and confirm transactions, and view the state of the Algorand Ledger.
     - `daemon/algod/api` ([README](daemon/algod/api/README.md)) is the REST
       interface used for interactions with algod.
  - `daemon/kmd` ([README](daemon/kmd/README.md)) holds the `kmd` daemon.  This
    daemon allows a node to sign transactions.  Because `kmd` is separate from
    `algod`, `kmd` allows a user to sign transactions on an air-gapped computer.

The following packages allow developers to interface with the Algorand system:

  - `cmd` holds the primary commands defining entry points into the system.
     - `cmd/catchupsrv` ([README](cmd/catchupsrv/README.md)) is a tool to
       assist with processing historic blocks on a new node.
  - `libgoal` exports a Go interface useful for developers of Algorand clients.
  - `debug` holds secondary commands which assist developers during debugging.

The following packages contain tools to help Algorand developers deploy networks
of their own:

  - `nodecontrol`
  - `tools`
  - `docker`
  - `commandandcontrol` ([README](test/commandandcontrol/README.md)) is a tool to
    automate a network of algod instances.
  - `components`
  - `netdeploy`

A number of packages provide utilities for the various components:

  - `logging` is a wrapper around `logrus`.
  - `util` contains a variety of utilities, including a codec, a SQLite wrapper,
    a goroutine pool, a timer interface, node metrics, and more.

`test` ([README](test/README.md)) contains end-to-end tests and utilities for the above components.

--------------------------------------------------------------------------------------------------

# Base Distributed File System Network

We used latest version of Onet (v.3.2.9) for network, simulation, and communication modules <https://github.com/dedis/onet/tree/v3.2.9>
as well as blockchain module from the ByzCoin Ng-first branch of cothority <https://github.com/dedis/cothority/tree/byzcoin_ng_first/protocols/byzcoin/blockchain>
and Cosi module from Cothority (it required some modification to work with Kyber and lastest Onet and with the blockchain module we used)

Onet's documents can be find under following link:
<https://github.com/dedis/onet/blob/master/README.md>
The Overlay-network (Onet) is a library for simulation and deployment of
decentralized, distributed protocols. This library offers a framework for
research, simulation, and deployment of crypto-related protocols with an emphasis
on decentralized, distributed protocols. It offers an abstraction for tree-based
communications between thousands of nodes and it is used both in research for
testing out new protocols and running simulations, as well as in production to
deploy those protocols as a service in a distributed manner.

**Onet** is developed by [DEDIS/EFPL](http://dedis.epfl.ch) as part of the
[Cothority](https://github.com/dedis/cothority) project that aims to deploy a
large number of nodes for distributed signing and related projects. In
cothority, nodes are commonly named "conodes". A collective authority
(cothority) is a set of conodes that work together to handle a distributed,
decentralized task.
