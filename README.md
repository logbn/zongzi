# Zongzi

A cluster coordinator for Dragonboat.

The primary goal of this package is to completely wrap dragonboat behind a facade that presents a simpler interface
with a lot of the complex tasks handled automatically using gRPC and good defaults.

- [ ] Initialize cluster
- [ ] Auto-join new hosts
- [ ] Store desired cluster state
- [ ] Replicate cluster state to all hosts
- [ ] Reconcile desired and actual cluster state
- [ ] Provide cluster-wide gRPC message bus

In order to add a replica to a shard in dragonboat, you must:

1. Call `SyncRequestAddReplica` on a host that is already a member of the shard
2. Call `StartReplica` on the host that wishes to join the shard

The Zongzi Agent simplifies these multi-host operations with a local API that automatically coordinates the necessary
multi-host actions required to achieve the desired result.

## Constraints

1. Although [dragonboat statemachine reads](https://pkg.go.dev/github.com/lni/dragonboat/v4#NodeHost.ReadLocalNode) 
accept and return `interface{}`, all queries and responses sent through the Agent must be expressed as `[]byte`, just
like command proposals. This serialization overhead is necessary for request forwarding because empty interfaces are 
not serializable.
