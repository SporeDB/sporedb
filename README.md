# 🍄 SporeDB [![build status](https://gitlab.com/SporeDB/sporedb/badges/master/build.svg)](https://gitlab.com/SporeDB/sporedb/commits/master)

SporeDB is a work-in-progress to a highly scalable, fast, resilient, decentralized and flexible database engine, named by analogy with the Mycology science.

## Idea

*Extract from the [full whitepaper](https://static.lesterpig.com/sporedb.pdf)*

Distributed databases are very popular when it comes to service scalability and high availability.
Such databases, like Apache Cassandra, MongoDB or Redis are able to handle node or network failures, but cannot handle nodes acting in a byzantine way.

Solving the Byzantine problem usually involves complex, costly or non-scalable consensus algorithms.
We can mention the well-known PBFT, the Bitcoin Proof-of-Work , the Tendermint protocol or the Stellar CP among many others.
Generally, these protocols require some strong coordination between nodes (leadership for example), or are mostly designed for a specific application (crypto-currencies for example).
This strong coordination reduces scalability and performance of the global system.

We introduce SporeDB as a way to solve these problems using simple (but powerful) techniques.

## ToDo

* Recovery after failures
  * Full state-transfer
  * Single state-transfer (with version comparison)
* Gossip network
* RSA+AES Encryption
* QUIC protocol (UDP) in P2P network
* Integration tests

## Done

* Basic database management
  * RocksDB Layer
  * SET operation
  * CONCAT operation
  * ADD operation (float)
  * MUL operation (float)
* Basic policy management
* Endorsement algorithm
* Integrity with Ed25519 signatures
  * CLI KeyRing management similar to OpenPGP
* P2P network
* Database GRPC client
