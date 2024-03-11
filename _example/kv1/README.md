# Zongzi - In-Memory KV Example

This example can be used to instantiate and initialize a 3 node cluster and add 3 more nodes as nonvoting.

It illustrates use of optimistic write locks to implement a consistent, persistent, in-memory finite state machine.

The example starts an HTTP server which performs queries on GET and proposes updates on PUT.

Any proposed update with an invalid version will be rejected.

From the command line:

```
> curl -X PUT "http://localhost:8001/testkey?val=testvalue"
{"key":"/testkey","ver":6,"val":"testvalue"}

> curl -X PUT "http://localhost:8001/testkey?val=testvalue2"
Version mismatch (0 != 6)

> curl -X PUT "http://localhost:8001/testkey?val=testvalue2&ver=6"
{"key":"/testkey","ver":8,"val":"testvalue2"}

> curl -X PUT "http://localhost:8001/testkey?val=testvalue3&ver=6"
Version mismatch (6 != 8)

> curl -X PUT "http://localhost:8001/testkey?val=testvalue3&ver=8"
{"key":"/testkey","ver":10,"val":"testvalue3"}
```

Or in a browser console:

Visit http://localhost:8001/testkey2 then:
```
await fetch(window.location.href, {method: 'PUT', body: new URLSearchParams('val=testval')})
await fetch(window.location.href, {method: 'PUT', body: new URLSearchParams('val=testval2&ver=6')})
```

## Startup

Run this command each time you want to start a new cluster (clears existing data):
```
make clean-data
```

In 3 separate terminals on the same machine, run the following commands:

```
make test-node-1
```
```
make test-node-2
```
```
make test-node-3
```

If performed correctly, you should see a snapshot dumped to stdout on whichever is voted the leader.

This snapshot should include 3 replicas of the prime shard and 3 replicas of the guest shard.

```
make test-node-4
```

Additional nodes added with duplicate zone should be added as non voting.

```json
{"index":54,"hosts":3,"shards":5,"replicas":15,"lastReplicaID":15,"lastShardID":4,"lastHostID":"69f63dd0-50cb-4410-bebc-6442d2c151af"}
{"id":"69f63dd0-50cb-4410-bebc-6442d2c151af","created":9,"updated":50,"status":"active","tags":{"geo:region":"us-central1","geo:zone":"us-west-1a"},"apiAddress":"127.0.0.1:17011","raftAddress":"127.0.0.1:17012","shardTypes":["zongzi://github.com/logbn/zongzi/_examples/kv1"]}
{"id":"bd789e7c-aa86-4e71-8aeb-f64f7d3ebf93","created":7,"updated":54,"status":"active","tags":{"geo:region":"us-central1","geo:zone":"us-west-1c"},"apiAddress":"127.0.0.1:17021","raftAddress":"127.0.0.1:17022","shardTypes":["zongzi://github.com/logbn/zongzi/_examples/kv1"]}
{"id":"dd2872e7-28ae-443c-8277-72e5b6883ca7","created":8,"updated":46,"status":"active","tags":{"geo:region":"us-central1","geo:zone":"us-west-1f"},"apiAddress":"127.0.0.1:17031","raftAddress":"127.0.0.1:17032","shardTypes":["zongzi://github.com/logbn/zongzi/_examples/kv1"]}
{"id":0,"created":6,"updated":26,"status":"","tags":null,"type":"zongzi://github.com/logbn/zongzi","name":"zongzi"}
{"id":1,"created":15,"updated":51,"status":"new","tags":{"placement:member":"3;geo:region=us-central1","placement:vary":"geo:zone"},"type":"zongzi://github.com/logbn/zongzi/_examples/kv1","name":"kv1-00001"}
{"id":2,"created":16,"updated":52,"status":"new","tags":{"placement:member":"3;geo:region=us-central1","placement:vary":"geo:zone"},"type":"zongzi://github.com/logbn/zongzi/_examples/kv1","name":"kv1-00002"}
{"id":3,"created":17,"updated":53,"status":"new","tags":{"placement:member":"3;geo:region=us-central1","placement:vary":"geo:zone"},"type":"zongzi://github.com/logbn/zongzi/_examples/kv1","name":"kv1-00003"}
{"id":4,"created":18,"updated":54,"status":"new","tags":{"placement:member":"3;geo:region=us-central1","placement:vary":"geo:zone"},"type":"zongzi://github.com/logbn/zongzi/_examples/kv1","name":"kv1-00004"}
{"id":1,"created":10,"updated":10,"status":"active","tags":null,"hostID":"69f63dd0-50cb-4410-bebc-6442d2c151af","isNonVoting":false,"isWitness":false,"shardID":0}
{"id":2,"created":11,"updated":11,"status":"active","tags":null,"hostID":"bd789e7c-aa86-4e71-8aeb-f64f7d3ebf93","isNonVoting":false,"isWitness":false,"shardID":0}
{"id":3,"created":12,"updated":12,"status":"active","tags":null,"hostID":"dd2872e7-28ae-443c-8277-72e5b6883ca7","isNonVoting":false,"isWitness":false,"shardID":0}
{"id":4,"created":31,"updated":31,"status":"active","tags":null,"hostID":"69f63dd0-50cb-4410-bebc-6442d2c151af","isNonVoting":false,"isWitness":false,"shardID":1}
{"id":5,"created":32,"updated":32,"status":"active","tags":null,"hostID":"bd789e7c-aa86-4e71-8aeb-f64f7d3ebf93","isNonVoting":false,"isWitness":false,"shardID":1}
{"id":6,"created":33,"updated":33,"status":"active","tags":null,"hostID":"dd2872e7-28ae-443c-8277-72e5b6883ca7","isNonVoting":false,"isWitness":false,"shardID":1}
{"id":7,"created":34,"updated":34,"status":"active","tags":null,"hostID":"69f63dd0-50cb-4410-bebc-6442d2c151af","isNonVoting":false,"isWitness":false,"shardID":2}
{"id":8,"created":35,"updated":35,"status":"active","tags":null,"hostID":"bd789e7c-aa86-4e71-8aeb-f64f7d3ebf93","isNonVoting":false,"isWitness":false,"shardID":2}
{"id":9,"created":36,"updated":36,"status":"active","tags":null,"hostID":"dd2872e7-28ae-443c-8277-72e5b6883ca7","isNonVoting":false,"isWitness":false,"shardID":2}
{"id":10,"created":37,"updated":37,"status":"active","tags":null,"hostID":"69f63dd0-50cb-4410-bebc-6442d2c151af","isNonVoting":false,"isWitness":false,"shardID":3}
{"id":11,"created":38,"updated":38,"status":"active","tags":null,"hostID":"bd789e7c-aa86-4e71-8aeb-f64f7d3ebf93","isNonVoting":false,"isWitness":false,"shardID":3}
{"id":12,"created":39,"updated":39,"status":"active","tags":null,"hostID":"dd2872e7-28ae-443c-8277-72e5b6883ca7","isNonVoting":false,"isWitness":false,"shardID":3}
{"id":13,"created":40,"updated":40,"status":"active","tags":null,"hostID":"69f63dd0-50cb-4410-bebc-6442d2c151af","isNonVoting":false,"isWitness":false,"shardID":4}
{"id":14,"created":41,"updated":41,"status":"active","tags":null,"hostID":"bd789e7c-aa86-4e71-8aeb-f64f7d3ebf93","isNonVoting":false,"isWitness":false,"shardID":4}
{"id":15,"created":42,"updated":42,"status":"active","tags":null,"hostID":"dd2872e7-28ae-443c-8277-72e5b6883ca7","isNonVoting":false,"isWitness":false,"shardID":4}
```

## Trivial Read Benchmark

These benchmarks are run in Ubuntu 22.04 in WSL2 on a 4Ghz i5-12600k desktop w/ 32GB DDR5 5600 and 2TB Samsung 980 Pro.

The load generator and the 3 node cluster are running in the same vm so YMMV.

These benchmarks all hit a single key so do not benefit from sharding.

### Concurrency 64 (follower)

80k rps @ 1.5ms p99 latency (to follower)

```
> hey -n 1000000 -c 64 "http://localhost:8001/testkey"

Summary:
  Total:        12.5200 secs
  Slowest:      0.0087 secs
  Fastest:      0.0002 secs
  Average:      0.0008 secs
  Requests/sec: 79872.1881

  Total data:   19000000 bytes
  Size/request: 19 bytes

Response time histogram:
  0.000 [1]      |
  0.001 [877521] |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  0.002 [118124] |■■■■■
  0.003 [3666]   |
  0.004 [401]    |
  0.004 [119]    |
  0.005 [122]    |
  0.006 [42]     |
  0.007 [2]      |
  0.008 [0]      |
  0.009 [2]      |


Latency distribution:
  10% in 0.0005 secs
  25% in 0.0007 secs
  50% in 0.0008 secs
  75% in 0.0009 secs
  90% in 0.0011 secs
  95% in 0.0011 secs
  99% in 0.0015 secs

Details (average, fastest, slowest):
  DNS+dialup:   0.0000 secs, 0.0002 secs, 0.0087 secs
  DNS-lookup:   0.0000 secs, 0.0000 secs, 0.0035 secs
  req write:    0.0000 secs, 0.0000 secs, 0.0028 secs
  resp wait:    0.0007 secs, 0.0002 secs, 0.0087 secs
  resp read:    0.0000 secs, 0.0000 secs, 0.0036 secs

Status code distribution:
  [200] 1000000 responses

 ```

### Concurrency 64 (leader)

120k rps @ 1.2ms p99 latency (to leader)

```
> hey -n 1000000 -c 64 "http://localhost:8003/testkey"

Summary:
  Total:        8.3917 secs
  Slowest:      0.1053 secs
  Fastest:      0.0001 secs
  Average:      0.0005 secs
  Requests/sec: 119165.5451

  Total data:   19000000 bytes
  Size/request: 19 bytes

Response time histogram:
  0.000 [1]      |
  0.011 [999948] |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  0.021 [0]      |
  0.032 [0]      |
  0.042 [0]      |
  0.053 [0]      |
  0.063 [0]      |
  0.074 [0]      |
  0.084 [0]      |
  0.095 [0]      |
  0.105 [51]     |


Latency distribution:
  10% in 0.0003 secs
  25% in 0.0004 secs
  50% in 0.0005 secs
  75% in 0.0006 secs
  90% in 0.0008 secs
  95% in 0.0009 secs
  99% in 0.0012 secs

Details (average, fastest, slowest):
  DNS+dialup:   0.0000 secs, 0.0001 secs, 0.1053 secs
  DNS-lookup:   0.0000 secs, 0.0000 secs, 0.0012 secs
  req write:    0.0000 secs, 0.0000 secs, 0.0043 secs
  resp wait:    0.0005 secs, 0.0001 secs, 0.1030 secs
  resp read:    0.0000 secs, 0.0000 secs, 0.0061 secs

Status code distribution:
  [200] 1000000 responses
```

### Concurrency 64 (stale)

300k rps @ 1.3ms p99 latency (stale read)

```
> hey -n 1000000 -c 64 "http://localhost:8001/testkey?stale=true"

Summary:
  Total:        3.3367 secs
  Slowest:      0.0162 secs
  Fastest:      0.0000 secs
  Average:      0.0002 secs
  Requests/sec: 299693.4242

  Total data:   19000000 bytes
  Size/request: 19 bytes

Response time histogram:
  0.000 [1]      |
  0.002 [993727] |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  0.003 [5493]   |
  0.005 [352]    |
  0.006 [280]    |
  0.008 [125]    |
  0.010 [18]     |
  0.011 [2]      |
  0.013 [0]      |
  0.015 [1]      |
  0.016 [1]      |


Latency distribution:
  10% in 0.0000 secs
  25% in 0.0001 secs
  50% in 0.0001 secs
  75% in 0.0003 secs
  90% in 0.0005 secs
  95% in 0.0006 secs
  99% in 0.0013 secs

Details (average, fastest, slowest):
  DNS+dialup:   0.0000 secs, 0.0000 secs, 0.0162 secs
  DNS-lookup:   0.0000 secs, 0.0000 secs, 0.0048 secs
  req write:    0.0000 secs, 0.0000 secs, 0.0060 secs
  resp wait:    0.0001 secs, 0.0000 secs, 0.0153 secs
  resp read:    0.0001 secs, 0.0000 secs, 0.0089 secs

Status code distribution:
  [200] 1000000 responses

```