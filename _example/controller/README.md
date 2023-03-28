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
{"index":31,"indexShard":1,"indexReplica":8,"countHost":4,"countShard":2,"countReplica":8}
{"id":"8f9a74ea-b354-4201-920d-ee433a2ce7c0","created":25,"updated":29,"meta":"eyJ6b25lIjoidXMtd2VzdC0xYSJ9","apiAddress":"127.0.0.1:17041","shardTypes":["github.com/logbn/zongzi-examples/controller"],"status":"active"}
{"id":"aa95f817-4c52-42a8-8634-d61952de186d","created":5,"updated":11,"meta":"eyJ6b25lIjoidXMtd2VzdC0xYiJ9","apiAddress":"127.0.0.1:17021","shardTypes":null,"status":"new"}
{"id":"bf5f9be4-ede3-4844-97b9-2179b2aae993","created":7,"updated":12,"meta":"eyJ6b25lIjoidXMtd2VzdC0xYyJ9","apiAddress":"127.0.0.1:17031","shardTypes":null,"status":"new"}
{"id":"d52cf51c-a2d3-4e9a-b1eb-3e566ef5ba26","created":10,"updated":16,"meta":"eyJ6b25lIjoidXMtd2VzdC0xYSJ9","apiAddress":"127.0.0.1:17011","shardTypes":["github.com/logbn/zongzi-examples/controller"],"status":"active"}
{"id":0,"created":9,"updated":9,"meta":null,"status":"new","type":"github.com/logbn/zongzi/prime","version":"v0.0.1"}
{"id":1,"created":18,"updated":18,"meta":null,"status":"new","type":"github.com/logbn/zongzi-examples/controller","version":"v0.0.1"}
{"id":1,"created":13,"updated":13,"meta":null,"hostID":"d52cf51c-a2d3-4e9a-b1eb-3e566ef5ba26","isNonVoting":false,"isWitness":false,"shardID":0,"status":"new"}
{"id":2,"created":14,"updated":14,"meta":null,"hostID":"aa95f817-4c52-42a8-8634-d61952de186d","isNonVoting":false,"isWitness":false,"shardID":0,"status":"new"}
{"id":3,"created":15,"updated":15,"meta":null,"hostID":"bf5f9be4-ede3-4844-97b9-2179b2aae993","isNonVoting":false,"isWitness":false,"shardID":0,"status":"new"}
{"id":4,"created":19,"updated":19,"meta":null,"hostID":"aa95f817-4c52-42a8-8634-d61952de186d","isNonVoting":false,"isWitness":false,"shardID":1,"status":"new"}
{"id":5,"created":20,"updated":20,"meta":null,"hostID":"bf5f9be4-ede3-4844-97b9-2179b2aae993","isNonVoting":false,"isWitness":false,"shardID":1,"status":"new"}
{"id":6,"created":21,"updated":21,"meta":null,"hostID":"d52cf51c-a2d3-4e9a-b1eb-3e566ef5ba26","isNonVoting":false,"isWitness":false,"shardID":1,"status":"new"}
{"id":7,"created":26,"updated":26,"meta":null,"hostID":"8f9a74ea-b354-4201-920d-ee433a2ce7c0","isNonVoting":true,"isWitness":false,"shardID":0,"status":"new"}
{"id":8,"created":28,"updated":28,"meta":null,"hostID":"8f9a74ea-b354-4201-920d-ee433a2ce7c0","isNonVoting":false,"isWitness":false,"shardID":1,"status":"new"}
```
