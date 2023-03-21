# ADR: Raft Sessions

Do we need to support raft client sessions?

> To achieve linearizability in Raft, servers must filter out duplicate requests. The basic idea is
that servers save the results of client operations and use them to skip executing the same request
multiple times. To implement this, each client is given a unique identifier, and clients assign unique
serial numbers to every command. Each server’s state machine maintains a session for each client.
The session tracks the latest serial number processed for the client, along with the associated response.
If a server receives a command whose serial number has already been executed, it responds
immediately without re-executing the request.
>
> -- Ongaro PhD (ch6.3)

## Use Case

Client request timeout causes proposal duplication on retry.

1. Client issues non-idempotent proposal
1. Client request times out
1. Proposal is committed
1. Client never receives response
1. Client re-sends proposal
1. Non-idempotent proposal is applied twice

This may be an issue in message bus architecture.

## Scope

### External

Sessions may need to trace all the way up the stack outside Zongzi to the application client in order for proposal deduplication to function correctly in this manner. Basically anything that might retry a request will need to maintain a session reference in order to prevent duplicate proposal.

### Internal

Given that every shard client would need a separate session, exposing this outside Zongzi may not make sense. Internal sessions could work if Zongzi operations are addressed to shards rather than replicas. However, application clients may want to issue commands and queries to specific replicas which (for instance) reside in the same cloud availability zone as the client. Addressing by shardID and treating all replicas as equally viable candidates for command/query processing may not provide the level of granularity desired by application clients.

Nonsense. Any ReplicaID can be derefenced to a ShardID. The client can easily maintain a shard session cache even if commands and queries are addressed by ReplicaID.

## Counter case

- Could developers just code their components in a way that makes every request idempotent?
    - Yes, but dedupe would occur on apply, adding redundant log entries and potentially increasing write amplification to maintain a dedupe index. Dedupe indexes not written to disk would need to be included in snapshots which could increase snapshot size.
- Sessions are not even supported by on disk state machines. What are we optimizing for here?
    - Given that on disk state machines will need to implement their own deduplication mechanisms anyways, the only value here appears to be reduced duplicate commit volume for non-disk based state machines.

## Conclusion

 The additional interface complexity of session juggling doesn't seem worthwhile given that disk-based state machines (the primary use case) can't benefit from use of Dragonboat sessions.

 Leave it out of the interace for now in alignment with design principle #1

 > [...] prefer basic interfaces and take great care in their design.

## Mitigation

- Provide examples to illustrate how applications can perform efficient message deduplication
    - Shard scoped
    - Client scoped

## History

- 2023-03-09 - KBurns - Proposed
