# Zongzi - Controller Example

This example can be used to instantiate and initialize a 3 node cluster and add 3 more nodes as nonvoting.

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
{"index":31,"shardID":1,"replicaID":8,"hosts":4,"shards":2,"replicas":8}
{"id":"0ab8a726-b125-493e-a3c3-d69545a633dd","created":5,"updated":8,"meta":"eyJ6b25lIjoidXMtd2VzdC0xYSJ9","apiAddress":"127.0.0.1:17011","shardTypes":["github.com/logbn/zongzi-examples/controller"],"status":"new"}
{"id":"0e12309e-4887-474c-9ec1-03894d84ac3c","created":10,"updated":16,"meta":"eyJ6b25lIjoidXMtd2VzdC0xYyJ9","apiAddress":"127.0.0.1:17031","shardTypes":["github.com/logbn/zongzi-examples/controller"],"status":"active"}
{"id":"7507a475-7825-4205-b377-77305cfe4f6c","created":25,"updated":29,"meta":"eyJ6b25lIjoidXMtd2VzdC0xYSJ9","apiAddress":"127.0.0.1:17041","shardTypes":["github.com/logbn/zongzi-examples/controller"],"status":"active"}
{"id":"dfd29e6f-49c4-4f9b-ba46-d9d6c777a497","created":9,"updated":14,"meta":"eyJ6b25lIjoidXMtd2VzdC0xYiJ9","apiAddress":"127.0.0.1:17021","shardTypes":["github.com/logbn/zongzi-examples/controller"],"status":"active"}
{"id":0,"created":7,"updated":7,"meta":null,"status":"new","type":"github.com/logbn/zongzi/prime","version":"v0.0.1"}
{"id":1,"created":18,"updated":18,"meta":null,"status":"new","type":"github.com/logbn/zongzi-examples/controller","version":"v0.0.1"}
{"id":1,"created":11,"updated":11,"meta":null,"hostID":"0ab8a726-b125-493e-a3c3-d69545a633dd","isNonVoting":false,"isWitness":false,"shardID":0,"status":"new"}
{"id":2,"created":12,"updated":12,"meta":null,"hostID":"dfd29e6f-49c4-4f9b-ba46-d9d6c777a497","isNonVoting":false,"isWitness":false,"shardID":0,"status":"new"}
{"id":3,"created":13,"updated":13,"meta":null,"hostID":"0e12309e-4887-474c-9ec1-03894d84ac3c","isNonVoting":false,"isWitness":false,"shardID":0,"status":"new"}
{"id":4,"created":19,"updated":19,"meta":null,"hostID":"0ab8a726-b125-493e-a3c3-d69545a633dd","isNonVoting":false,"isWitness":false,"shardID":1,"status":"new"}
{"id":5,"created":20,"updated":20,"meta":null,"hostID":"0e12309e-4887-474c-9ec1-03894d84ac3c","isNonVoting":false,"isWitness":false,"shardID":1,"status":"new"}
{"id":6,"created":21,"updated":21,"meta":null,"hostID":"dfd29e6f-49c4-4f9b-ba46-d9d6c777a497","isNonVoting":false,"isWitness":false,"shardID":1,"status":"new"}
{"id":7,"created":26,"updated":26,"meta":null,"hostID":"7507a475-7825-4205-b377-77305cfe4f6c","isNonVoting":true,"isWitness":false,"shardID":0,"status":"new"}
{"id":8,"created":28,"updated":28,"meta":null,"hostID":"7507a475-7825-4205-b377-77305cfe4f6c","isNonVoting":true,"isWitness":false,"shardID":1,"status":"new"}
```
