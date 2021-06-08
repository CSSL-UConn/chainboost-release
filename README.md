<!-- [![Build Status](https://travis-ci.org/dedis/onet.svg?branch=master)](https://travis-ci.org/dedis/onet)
[![Go Report Card](https://goreportcard.com/badge/github.com/dedis/onet)](https://goreportcard.com/report/github.com/dedis/onet)
[![Coverage Status](https://coveralls.io/repos/github/dedis/onet/badge.svg)](https://coveralls.io/github/dedis/onet)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/11ec79aa77fe41748edfdfcd55e92fab)](https://www.codacy.com/manual/nkcr/onet?utm_source=github.com&utm_medium=referral&utm_content=dedis/onet&utm_campaign=Badge_Grade) -->
 
# The Cothority Overlay Network Library - Onet

We used latest version of Onet (v.3.2.9) for network, simulation, and communication modules https://github.com/dedis/onet/tree/v3.2.9
as well as blockchain module from the ByzCoin Ng-first branch of cothority https://github.com/dedis/cothority/tree/byzcoin_ng_first/protocols/byzcoin/blockchain
and Cosi module from Cothority (it required some modification to work with Kyber and lastest Onet and with the blockchain module we used)

Onet's documents can be find under following link:
https://github.com/dedis/onet/blob/master/README.md
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
