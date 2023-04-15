# Zongzi - In-Memory KV Example

This example can be used to instantiate and initialize a 3 node cluster and add 3 more nodes as nonvoting.

It illustrates use of optimistic write locks to implement a consistent, persistent, in-memory finite state machine.

The example starts an HTTP server which performs queries on GET and proposes updates on PUT.

Any proposed update with an invalid version will be rejected.

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
{"index":60,"hosts":3,"shards":5,"replicas":18,"lastReplicaID":18,"lastShardID":4,"lastHostID":"e15b7e9d-6a74-4bcb-b43f-6a4cadb4615d"}
{"id":"77a67fd5-a460-4379-80d3-3ebdcaff9a28","created":7,"updated":53,"status":"active","tags":{"geo:region":"us-central1","geo:zone":"us-west-1a"},"apiAddress":"127.0.0.1:17011","raftAddress":"127.0.0.1:17012","shardTypes":["zongzi://github.com/logbn/zongzi/_examples/kv1"]}
{"id":"a06c2f96-0075-4d4b-98a6-83f8246c8d70","created":9,"updated":60,"status":"active","tags":{"geo:region":"us-central1","geo:zone":"us-west-1c"},"apiAddress":"127.0.0.1:17021","raftAddress":"127.0.0.1:17022","shardTypes":["zongzi://github.com/logbn/zongzi/_examples/kv1"]}
{"id":"e15b7e9d-6a74-4bcb-b43f-6a4cadb4615d","created":10,"updated":58,"status":"active","tags":{"geo:region":"us-central1","geo:zone":"us-west-1f"},"apiAddress":"127.0.0.1:17031","raftAddress":"127.0.0.1:17032","shardTypes":["zongzi://github.com/logbn/zongzi/_examples/kv1"]}
{"id":0,"created":5,"updated":32,"status":"","tags":null,"type":"zongzi://github.com/logbn/zongzi","name":"zongzi"}
{"id":1,"created":20,"updated":56,"status":"new","tags":{"placement:member":"3;geo:region=us-central1","placement:vary":"geo:zone"},"type":"zongzi://github.com/logbn/zongzi/_examples/kv1","name":"kv1-00000"}
{"id":2,"created":23,"updated":58,"status":"new","tags":{"placement:member":"3;geo:region=us-central1","placement:vary":"geo:zone"},"type":"zongzi://github.com/logbn/zongzi/_examples/kv1","name":"kv1-00001"}
{"id":3,"created":25,"updated":59,"status":"new","tags":{"placement:member":"3;geo:region=us-central1","placement:vary":"geo:zone"},"type":"zongzi://github.com/logbn/zongzi/_examples/kv1","name":"kv1-00002"}
{"id":4,"created":26,"updated":60,"status":"new","tags":{"placement:member":"3;geo:region=us-central1","placement:vary":"geo:zone"},"type":"zongzi://github.com/logbn/zongzi/_examples/kv1","name":"kv1-00003"}
{"id":1,"created":12,"updated":12,"status":"active","tags":null,"hostID":"77a67fd5-a460-4379-80d3-3ebdcaff9a28","isNonVoting":false,"isWitness":false,"shardID":0}
{"id":2,"created":13,"updated":13,"status":"active","tags":null,"hostID":"a06c2f96-0075-4d4b-98a6-83f8246c8d70","isNonVoting":false,"isWitness":false,"shardID":0}
{"id":3,"created":15,"updated":15,"status":"active","tags":null,"hostID":"e15b7e9d-6a74-4bcb-b43f-6a4cadb4615d","isNonVoting":false,"isWitness":false,"shardID":0}
{"id":4,"created":17,"updated":17,"status":"joining","tags":null,"hostID":"77a67fd5-a460-4379-80d3-3ebdcaff9a28","isNonVoting":false,"isWitness":false,"shardID":0}
{"id":5,"created":19,"updated":19,"status":"joining","tags":null,"hostID":"a06c2f96-0075-4d4b-98a6-83f8246c8d70","isNonVoting":false,"isWitness":false,"shardID":0}
{"id":6,"created":21,"updated":21,"status":"joining","tags":null,"hostID":"e15b7e9d-6a74-4bcb-b43f-6a4cadb4615d","isNonVoting":false,"isWitness":false,"shardID":0}
{"id":7,"created":37,"updated":37,"status":"active","tags":null,"hostID":"77a67fd5-a460-4379-80d3-3ebdcaff9a28","isNonVoting":false,"isWitness":false,"shardID":1}
{"id":8,"created":38,"updated":38,"status":"active","tags":null,"hostID":"a06c2f96-0075-4d4b-98a6-83f8246c8d70","isNonVoting":false,"isWitness":false,"shardID":1}
{"id":9,"created":39,"updated":39,"status":"active","tags":null,"hostID":"e15b7e9d-6a74-4bcb-b43f-6a4cadb4615d","isNonVoting":false,"isWitness":false,"shardID":1}
{"id":10,"created":40,"updated":40,"status":"active","tags":null,"hostID":"77a67fd5-a460-4379-80d3-3ebdcaff9a28","isNonVoting":false,"isWitness":false,"shardID":2}
{"id":11,"created":41,"updated":41,"status":"active","tags":null,"hostID":"a06c2f96-0075-4d4b-98a6-83f8246c8d70","isNonVoting":false,"isWitness":false,"shardID":2}
{"id":12,"created":42,"updated":42,"status":"active","tags":null,"hostID":"e15b7e9d-6a74-4bcb-b43f-6a4cadb4615d","isNonVoting":false,"isWitness":false,"shardID":2}
{"id":13,"created":43,"updated":43,"status":"active","tags":null,"hostID":"77a67fd5-a460-4379-80d3-3ebdcaff9a28","isNonVoting":false,"isWitness":false,"shardID":3}
{"id":14,"created":44,"updated":44,"status":"active","tags":null,"hostID":"a06c2f96-0075-4d4b-98a6-83f8246c8d70","isNonVoting":false,"isWitness":false,"shardID":3}
{"id":15,"created":45,"updated":45,"status":"active","tags":null,"hostID":"e15b7e9d-6a74-4bcb-b43f-6a4cadb4615d","isNonVoting":false,"isWitness":false,"shardID":3}
{"id":16,"created":47,"updated":47,"status":"active","tags":null,"hostID":"77a67fd5-a460-4379-80d3-3ebdcaff9a28","isNonVoting":false,"isWitness":false,"shardID":4}
{"id":17,"created":48,"updated":48,"status":"active","tags":null,"hostID":"a06c2f96-0075-4d4b-98a6-83f8246c8d70","isNonVoting":false,"isWitness":false,"shardID":4}
{"id":18,"created":49,"updated":49,"status":"active","tags":null,"hostID":"e15b7e9d-6a74-4bcb-b43f-6a4cadb4615d","isNonVoting":false,"isWitness":false,"shardID":4}
```

## Read Benchmark

```
> ab -n 10000 -c 16 "http://localhost:8001/testkey"

This is ApacheBench, Version 2.3 <$Revision: 1843412 $>
Copyright 1996 Adam Twiss, Zeus Technology Ltd, http://www.zeustech.net/
Licensed to The Apache Software Foundation, http://www.apache.org/

Benchmarking localhost (be patient)
Completed 1000 requests
Completed 2000 requests
Completed 3000 requests
Completed 4000 requests
Completed 5000 requests
Completed 6000 requests
Completed 7000 requests
Completed 8000 requests
Completed 9000 requests
Completed 10000 requests
Finished 10000 requests


Server Software:
Server Hostname:        localhost
Server Port:            8001

Document Path:          /testkey
Document Length:        24 bytes

Concurrency Level:      16
Time taken for tests:   0.262 seconds
Complete requests:      10000
Failed requests:        0
Total transferred:      1410000 bytes
HTML transferred:       240000 bytes
Requests per second:    38173.77 [#/sec] (mean)
Time per request:       0.419 [ms] (mean)
Time per request:       0.026 [ms] (mean, across all concurrent requests)
Transfer rate:          5256.35 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    0   0.1      0       1
Processing:     0    0   0.2      0       3
Waiting:        0    0   0.2      0       3
Total:          0    0   0.2      0       3

Percentage of the requests served within a certain time (ms)
  50%      0
  66%      0
  75%      0
  80%      1
  90%      1
  95%      1
  98%      1
  99%      1
 100%      3 (longest request)
 ```