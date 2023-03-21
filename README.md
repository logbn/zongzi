# Zongzi

A cluster coordinator for Dragonboat.

[![Go Reference](https://godoc.org/github.com/logbn/zongzi?status.svg)](https://godoc.org/github.com/logbn/zongzi)
[![Go Report Card](https://goreportcard.com/badge/github.com/logbn/zongzi?1)](https://goreportcard.com/report/github.com/logbn/zongzi)
[![Go Coverage](https://github.com/logbn/zongzi/wiki/coverage.svg)](https://raw.githack.com/wiki/logbn/zongzi/coverage.html)

The primary goal of this package is to completely wrap Dragonboat behind a facade that presents a simpler interface
with a lot of the complex tasks handled automatically using centralized cluster state management, host controllers
and an internal gRPC API.

- Initialize cluster
- Auto-join new hosts
- Store desired cluster state
- Replicate cluster state to all hosts
- Reconcile desired and actual cluster state
- Provide cluster-wide gRPC message bus

In order to add a replica to a shard using Dragonboat, you must:

1. Call `dragonboat.(*NodeHost).SyncRequestAddReplica` from a host that is already a member of the shard
2. Call `dragonboat.(*NodeHost).StartReplica` from the host that wishes to join the shard

The zongzi Agent simplifies this sort of multi-host operation using an internal API that automatically coordinates the
necessary multi-host actions required to achieve the desired cluster state.

1. Call `zongzi.Agent.CreateReplica` from any host in the cluster and the replica will eventually be added and started
on the desired host

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

3. Any host may have at most one active replica of any shard. No host may ever have more than one active replica of
any shard. A host may have any number of inactive replicas for any shard. This aligns with many Dragonboat host
operations like [(*NodeHost).StopShard](https://pkg.go.dev/github.com/lni/dragonboat/v4#NodeHost.StopShard) which
assumes one active replica per shard per host. A replica can never be reactivated after being marked inactive.

4. Although Dragonboat supports multiple types of statemachines (
[IStateMachine](https://pkg.go.dev/github.com/lni/dragonboat/v4@v4.0.0-20230202152124-023bafb8e648/statemachine#IStateMachine),
[IConcurrentStateMachine](https://pkg.go.dev/github.com/lni/dragonboat/v4@v4.0.0-20230202152124-023bafb8e648/statemachine#IConcurrentStateMachine),
[IOnDiskStateMachine](https://pkg.go.dev/github.com/lni/dragonboat/v4@v4.0.0-20230202152124-023bafb8e648/statemachine#IOnDiskStateMachine)),
they can all be considered subsets of `IOnDiskStateMachine`. To keep things simple, only one type of state machine and
state machine factory is supported with no support for `IExtended`.

5. Dragonboat sessions are not supported (they didn't work with on disk state machines anyways). Proposal deduplication
is delegated to the developer. See [ADR: Raft Sessions](/docs/adr/raft_sessions.md)

6. Initializing a Zongzi cluster requires at least 3 peer agents. Single node clusters are not supported.

# Setup

## Requirements for `make gen`

1. Install [protoc](https://grpc.io/docs/protoc-installation/)
2. Install [protoc-gen-go](https://grpc.io/docs/languages/go/quickstart/)
