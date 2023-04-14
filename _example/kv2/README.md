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
{"index":37,"shardID":1,"replicaID":8,"hosts":4,"shards":2,"replicas":8}
{"id":"2350f6b8-244d-4e4a-8ce6-97bfb4d2f2c2","meta":"eyJ6b25lIjoidXMtd2VzdC0xYyJ9","status":"active","created":13,"updated":33,"apiAddress":"127.0.0.1:17031","raftAddress":"","shardTypes":["github.com/logbn/zongzi-examples/kv-in-memory"]}
{"id":"36748fc1-d9c0-4bd4-a376-0201fcf8bbdb","meta":"eyJ6b25lIjoidXMtd2VzdC0xYSJ9","status":"active","created":12,"updated":20,"apiAddress":"127.0.0.1:17011","raftAddress":"","shardTypes":["github.com/logbn/zongzi-examples/kv-in-memory"]}
{"id":"6ea1cd18-805b-4a4d-a192-4d8b0f9d7509","meta":"eyJ6b25lIjoidXMtd2VzdC0xYSJ9","status":"active","created":25,"updated":35,"apiAddress":"127.0.0.1:17041","raftAddress":"","shardTypes":["github.com/logbn/zongzi-examples/kv-in-memory"]}
{"id":"ddd73798-31ec-4c49-98fe-679119e8f770","meta":"eyJ6b25lIjoidXMtd2VzdC0xYiJ9","status":"active","created":7,"updated":21,"apiAddress":"127.0.0.1:17021","raftAddress":"","shardTypes":["github.com/logbn/zongzi-examples/kv-in-memory"]}
{"id":0,"meta":null,"status":"new","created":5,"updated":34,"type":"github.com/logbn/zongzi/prime","version":"v0.0.1"}
{"id":1,"meta":null,"status":"new","created":18,"updated":37,"type":"github.com/logbn/zongzi-examples/kv-in-memory","version":"v0.0.1"}
{"id":1,"meta":null,"status":"active","created":9,"updated":9,"hostID":"36748fc1-d9c0-4bd4-a376-0201fcf8bbdb","isNonVoting":false,"isWitness":false,"shardID":0}
{"id":2,"meta":null,"status":"active","created":10,"updated":10,"hostID":"ddd73798-31ec-4c49-98fe-679119e8f770","isNonVoting":false,"isWitness":false,"shardID":0}
{"id":3,"meta":null,"status":"active","created":11,"updated":11,"hostID":"2350f6b8-244d-4e4a-8ce6-97bfb4d2f2c2","isNonVoting":false,"isWitness":false,"shardID":0}
{"id":4,"meta":null,"status":"active","created":19,"updated":19,"hostID":"2350f6b8-244d-4e4a-8ce6-97bfb4d2f2c2","isNonVoting":false,"isWitness":false,"shardID":1}
{"id":5,"meta":null,"status":"active","created":20,"updated":20,"hostID":"36748fc1-d9c0-4bd4-a376-0201fcf8bbdb","isNonVoting":false,"isWitness":false,"shardID":1}
{"id":6,"meta":null,"status":"active","created":21,"updated":21,"hostID":"ddd73798-31ec-4c49-98fe-679119e8f770","isNonVoting":false,"isWitness":false,"shardID":1}
{"id":7,"meta":null,"status":"active","created":26,"updated":26,"hostID":"6ea1cd18-805b-4a4d-a192-4d8b0f9d7509","isNonVoting":true,"isWitness":false,"shardID":0}
{"id":8,"meta":null,"status":"active","created":35,"updated":35,"hostID":"6ea1cd18-805b-4a4d-a192-4d8b0f9d7509","isNonVoting":true,"isWitness":false,"shardID":1}
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
