# Zongzi

<img alt="chibi style zongzi rice pyramid" src="https://user-images.githubusercontent.com/20638/228140399-d7b5cb5f-b2e4-4452-8de0-3decbe35bb36.png" width="220" height="180" align="right"/>

A cluster coordinator for Dragonboat

[![Go Reference](https://godoc.org/github.com/logbn/zongzi?status.svg)](https://godoc.org/github.com/logbn/zongzi)
[![Go Report Card](https://goreportcard.com/badge/github.com/logbn/zongzi?4)](https://goreportcard.com/report/github.com/logbn/zongzi)
[![Go Coverage](https://github.com/logbn/zongzi/wiki/coverage.svg)](https://raw.githack.com/wiki/logbn/zongzi/coverage.html)

This package provides a centralized coordination layer for Dragonboat multi-group Raft consesus clusters.

### Components

- Registry
  - Stores desired state of all hosts, shards and replicas in the cluster
  - Cluster state stored in raft shard zero (shardID: 0)
  - All cluster state changes pass through the Zongzi agent
- Message Bus
  - Internal gPRC
  - Facilitates cluster boostrap
  - Forwards proposals and queries between nodes
- Host Controller
  - Manages replicas on all hosts (start, stop, recover, delete, etc)
  - Responds automatically to changes in cluster state registry
- Shard Controller
  - Reads placement policies from shard tags
  - Creates and destroys replicas to reconcile cluster state
  - You can replace the default shard controller with your own custom version
- Host Client
  - Used to make proposals and queries to replicas on specific hosts
  - gRPC over message bus
- Shard Client
  - Intelligently routes proposals and queries to active shard replicas
  - Selects nearest replica based on ping
  - Load balances stale reads across replicas

## Usage

The Zongzi Agent simplifies multi-host operations using an internal API that automatically coordinates the
necessary multi-host actions required to achieve the desired cluster state.

1. Call `zongzi.(*Agent).CreateReplica` from any host in the cluster
2. The desired replica state will be stored in the registry
3. The responsible host controller will start the replica on the desired host

The cluster state registry is replicated to every host in the cluster so every host always has an eventually consistent
snapshot of the cluster topology for command/query forwarding. Changes to the cluster can be proposed via internal API
so cluster changes are as simple as writing to the registry and the host controllers will reconcile the difference.

There is no need to import the dragonboat package or interact with the dragonboat host directly.

## Architectural Constraints

Zongzi imposes some additional architectural constraints on top of Dragonboat:

1. Although [Dragonboat statemachine reads](https://pkg.go.dev/github.com/lni/dragonboat/v4#NodeHost.ReadLocalNode)
accept and return `interface{}`, all queries and responses sent through Zongzi must be expressed as `[]byte`, just
like command proposals. This serialization overhead is necessary for request forwarding because empty interfaces are
not serializable. See [ADR: Message Bus](/docs/adr/sessions.md)

2. Although [Dragonboat replica IDs](https://pkg.go.dev/github.com/lni/dragonboat/v4#NodeHost.HasNodeInfo) are unique
per shard, Zongzi replica IDs are unique per cluster. Having independent replica ids simplifies many replica operations
which may have previously required both a shard id and replica id to be passed together up and down the callstack. The
loss of address space (`uint64 * uint64` vs `uint64`) is not expected to be a concern as 18.4 quintillion is still
an astonomically large number. Having a materialized view of the global cluster state replicated to every host in the
cluster makes it simple and efficient to derefence a replicaID to the correct host and shard. Dragonboat can't do this
alone because its cluster state is decentralized.

3. Any host may have at most one active replica of any shard. This aligns with many Dragonboat host operations like
[(*NodeHost).StopShard](https://pkg.go.dev/github.com/lni/dragonboat/v4#NodeHost.StopShard) which assumes no more than
one active replica per shard per host.

4. Dragonboat sessions are not supported (they didn't work with on disk state machines anyways). Proposal deduplication
is delegated to the developer. See [ADR: Raft Sessions](/docs/adr/raft_sessions.md)

5. Initializing a Zongzi cluster requires at least 3 peer agents. Single host clusters are not supported.

# Setup

## Requirements for `make gen`

1. Install [protoc](https://grpc.io/docs/protoc-installation/)
2. Install [protoc-gen-go](https://grpc.io/docs/languages/go/quickstart/)

# Project Status

This library is in alpha (`v0.0.x`). Expect regular breaking changes to interfaces and serious defects outside the green
path. When the API is stable and the code is well tested, beta releases will begin (`v0.1.0`). General availability
will come after the library has been thoroughly tested in at least one production deployment for an extended period of
time (`v1.0.0`).
