# ADR: Message Bus

Should state machine registration require an API that receives messages and is responsible for issuing proposals and queries to the local state machine?

### Advantages

1. No need to add constraints for query serialization.
    - Native `ReadLocal(any) any` just works
    - May reduce seralization overhead for queries
2. May allow greater flexibility for some workloads
    - State machine can remain simple
    - API can compose queries and proposals into complex actions
3. May provide some performance improvement opportunities (buffered writes)

### Disadvantages

1. One more required interface for clients to implement
    - State machine
    - Controller
    - Client
2. Use case might not warrant complexity
    - Does anyone really need this?
    - Does everyone really need this?
    - Can't this logic be embedded in the state machine?

### Mitigation 1 - Implement some useful default clients like `passthrough.Client`
- Requires an envelope format to route messages to:
    - Proposal vs Query
    - Linear vs Local
- Still requires all Queries to be serializable

### Use case 1 - Colocated components

A stream and a dedupe index are partitioned alike. Writes to the stream shard are guarded by the dedupe shard.

- If two components are going to be partitioned alike and colocated then why not combine them into one state machine?
- Should a replica API even be capable of accesssing other replicas?

### Use case 2 - Materialized views

A shard is replicated to every host in the cluster. This shard is frequently referenced for authorization, joining, filtering, etc.

- Will anyone really want this?
    - Probably. Map/Reduce is a hell of a drug.

### Anti-use case

Any state machine API could be implemented as a concurrent state machine Query.
It may read local for settings and query local replicas through the agent.
A reference to the agent would be passed into the factory constructor.

- This is the way.

## Conclusion

1. Every piece of data passed through the message bus is either a command or a query.
2. Any sort of api required to access multiple components locally should be implemented as a concurrent state machine and provisioned as a separate replica.
3. Commands and Queries are byte slices.

## History

- 2023-03-09 - KBurns - Proposed
