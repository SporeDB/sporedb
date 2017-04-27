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

## Overview

The SporeDB architecture is represented in the following figure :
* SporeDB nodes are connected through a P2P network using a home-made "Mycelium" protocol ;
* Every node store a whole copy of the database, ensuring data security ;
* Write operations are applied by the cluster with the SporeDB Consensus Algorithm, presented (and hopefully proved) in the whitepaper ;
* Nodes expose a GRPC API server that can be used by clients to access the data and to submit transactions.

![overview](doc/overview.png)

## Installation

### Easy installation (Docker)

```bash
$ docker pull registry.gitlab.com/sporedb/sporedb
```

We suggest that you use docker volumes to preserve SporeDB states, like this:

```bash
$ docker run --rm -it -e PASSWORD=******* -v $PWD:/sporedb registry.gitlab.com/sporedb/sporedb --help
```

### Build environement

The following requirements are needed before building SporeDB:

* [Go 1.8+](https://golang.org/dl/) properly configured
* [RocksDB v5.1.x](https://github.com/facebook/rocksdb.git)
* [Protoc v3.1.x](https://github.com/google/protobuf.git)
  * With the [Protoc-Gen-Go](https://github.com/golang/protobuf) plugin

**To setup a compilation environement please refer to the up-to-date [continuous integration](ci/Dockerfile) Dockerfile.**
After this setup your can build the project with
```bash
$ make
```

## Configuration

### Writing main configuration file

SporeDB needs a YAML configuration file.
An example is available in [sporedb.yaml](sporedb.yaml) and can be used as-is after edition of `identity` field that will identify you in your network.

You might also want to add some information about peers to connect to in this configuration file.

*Please note that right now, network topology is mainly static. This will be upgraded to a full gossip network soon.*

### Setting-up crypto credentials

You will need some credentials to build a Trust Network.
SporeDB uses a system very similar to OpenPGP, with some specific modifications.
Basically, each node of the network holds a public/private Ed25519 keys pair for integrity verification.

First of all, you must set the `PASSWORD` environment variable.
This password will be used to encrypt your private key.
You might then want to send your public key to the other peers of your network.

```bash
$ export PASSWORD=********
$ sporedb keys init   # Will create your credentials
$ sporedb keys export # Will export your public key
```

You might also want to import other's public key with a specific trust level in your keyring.
For example, the following command imports the Alice's public key, stored in `alice.pem`, with a High trust level.

```
$ cat alice.pem > sporedb keys import alice -t high
```

For more information and advanced features (like key's signatures), see `sporedb keys -h`.

### Creating a policy

Policies define what nodes can and cannot do accross a network of nodes ("Mycelium").
We encourage you to read the [full whitepaper](https://static.lesterpig.com/sporedb.pdf) to fully understand how policy are designed.

Policies are stored in JSON files, and can be created with a wizzard:

```bash
$ sporedb keys ls
+----------+----------+-----------+----------------+
| Identity |  Trust   | Certified |  Fingerprint   |
+----------+----------+-----------+----------------+
| <self>   | ultimate | ✔️️ yes     | 63:29:41:4A:B9 |
+----------+----------+-----------+----------------+
| bob      | high     | ✔️️ yes     | B1:D3:CD:91:07 |
+----------+----------+-----------+----------------+
| carol    | high     | ✔️️ yes     | 09:0F:86:26:E0 |
+----------+----------+-----------+----------------+

$ sporedb policy create
Name of the policy [6cfeddad-eaec-4e0b-abf8-4658f0297402]:
Comment []: A test policy
Shall this node be considered as an endorser? [y/n] [y]: y
Endorser #1 (blank to skip) []: bob
Endorser #2 (blank to skip) []: carol
Endorser #3 (blank to skip) []:
Maximum number of byzantine (faulty) endorsers [1]: 1
Quorum [3]: 3
```

The previous dialog will create a policy that will allow the current node, bob and carol to endorse (validate) spore submissions in the network.

## Usage

Each SporeDB node will offer a GRPC API server, enabling Clients to connect to it.
Right now, it is possible to send basic instructions to one Node using embedded client.

On first terminal:

```bash
$ sporedb server
Successfully loaded policy test
SporeDB is running on localhost:4000
```

On second terminal:

```bash
$ sporedb client -s localhost:4000 -p test
Now using policy test
SporeDB client is connected and ready to execute your luscious instructions!
localhost:4000> SET key value
Transaction: bbd5aa6b-7b56-4ce9-926a-ea0ce6175ca0
localhost:4000> GET key
value
```

Documentation is being written about client capabilities.
You can though check the available commands [here](db/client/cli.go).

## Acknowledgements

SporeDB should **NOT** be used in production yet.
It is very new and not stable enough.

Feedbacks about the project and the whitepaper will be very much appreciated! 😘

## Implemented features

* Basic database management
  * RocksDB Layer
  * SET operation
  * CONCAT operation
  * ADD operation (float)
  * MUL operation (float)
  * SADD operation (set)
  * SREM operation (set)
* Basic policy management
* Endorsement algorithm
* Integrity with Ed25519 signatures
  * CLI KeyRing management similar to OpenPGP
* P2P network
* GRPC Server / Client API
* Database GRPC client
* Recovery after failures
  * Single state-transfer (with version comparison)
