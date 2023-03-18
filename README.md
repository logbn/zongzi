# zongzi

A cluster coordinator for Dragonboat.

The primary goal of this package is to completely wrap dragonboat behind a facade that presents a simpler interface
with a lot of the complex tasks handled automatically using gRPC and good defaults.

- Initialize cluster
- Auto-join new hosts
- Store desired cluster state
- Replicate cluster state to all hosts
- Reconcile desired and actual cluster state
- Provide cluster-wide gRPC message bus

In order to add a replica to a shard in dragonboat, you must:

1. Call `dragonboat.(*NodeHost).SyncRequestAddReplica` on a host that is already a member of the shard
2. Call `dragonboat.(*NodeHost).StartReplica` on the host that wishes to join the shard

The zongzi Agent simplifies these multi-host operations with a local agent API that automatically coordinates the
necessary multi-host actions required to achieve the desired result.

1. Call `zongzi.Agent.CreateReplica` from any host and the replica will eventually be added and started

## Constraints

1. Although [Dragonboat statemachine reads](https://pkg.go.dev/github.com/lni/dragonboat/v4#NodeHost.ReadLocalNode) 
accept and return `interface{}`, all queries and responses sent through the Agent must be expressed as `[]byte`, just
like command proposals. This serialization overhead is necessary for request forwarding because empty interfaces are
not serializable.
    ```go
    Propose(ctx context.Context, replicaID uint64, cmd []byte, linear bool) (value uint64, data []byte, err error)
    Query(ctx context.Context, replicaID uint64, query []byte, linear bool) (value uint64, data []byte, err error)
    ```

2. Although [Dragonboat replica IDs](https://pkg.go.dev/github.com/lni/dragonboat/v4#NodeHost.HasNodeInfo) are unique
per shard, Zongzi replica IDs are unique per cluster. Having independent replica ids simplifies many replica operations
which may have previously required both a shard id and replica id to be passed together up and down the callstack. The
loss of address space (`uint64 * uint64` vs `uint64`) is not expected to be a concern as 18.4 quintillion is still 
an astonomically large number. Having a consistent, locally replicated view of the global cluster state on every host
makes these lookups simple and efficient.

3. Any host may have at most one active replica of any shard. No host may ever have more than one active replica of
a shard. A host may have any number of inactive replicas for any shard. This aligns with some Dragonboat host operations
such as [(*NodeHost).StopShard](https://pkg.go.dev/github.com/lni/dragonboat/v4#NodeHost.StopShard) which assumes one
active replica per shard per host. A replica can never be reactivated after being marked inactive.

# Setup

## Requirements for `make gen`

1. Install [protoc](https://grpc.io/docs/protoc-installation/)
2. Install [protoc-gen-go](https://grpc.io/docs/languages/go/quickstart/)
