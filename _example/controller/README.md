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

This snapshot should include 3 replicas of the prime shard (`zongzi`) and 3 replicas of the guest shard (`banana`).

```
make test-node-4
```

Additional nodes added with duplicate zone should be added as non voting.

```json
{
    "Hosts": [
        {
            "id": "a74bf4be-df70-4c7a-9682-1458015d8c44",
            "created": 5,
            "updated": 16,
            "apiAddress": "127.0.0.1:17031",
            "meta": "eyJ6b25lIjoidXMtd2VzdC0xYyJ9",
            "shardTypes": [
                "github.com/logbn/zongzi-examples/controller"
            ],
            "status": "active"
        },
        {
            "id": "1fced3d3-91e9-44f3-830e-5a95b65a2287",
            "created": 6,
            "updated": 17,
            "apiAddress": "127.0.0.1:17021",
            "meta": "eyJ6b25lIjoidXMtd2VzdC0xYiJ9",
            "shardTypes": [
                "github.com/logbn/zongzi-examples/controller"
            ],
            "status": "active"
        },
        {
            "id": "96567b03-7269-4c1e-94fd-653ef54e3544",
            "created": 8,
            "updated": 18,
            "apiAddress": "127.0.0.1:17011",
            "meta": "eyJ6b25lIjoidXMtd2VzdC0xYSJ9",
            "shardTypes": [
                "github.com/logbn/zongzi-examples/controller"
            ],
            "status": "active"
        },
        {
            "id": "3de4eee9-d0d6-490e-a024-61108959b753",
            "created": 19,
            "updated": 24,
            "apiAddress": "127.0.0.1:17041",
            "meta": "eyJ6b25lIjoidXMtd2VzdC0xYSJ9",
            "shardTypes": [
                "github.com/logbn/zongzi-examples/controller"
            ],
            "status": "active"
        }
    ],
    "Index": 24,
    "ReplicaIndex": 8,
    "Replicas": [
        {
            "id": 1,
            "created": 11,
            "updated": 11,
            "hostID": "96567b03-7269-4c1e-94fd-653ef54e3544",
            "isNonVoting": false,
            "isWitness": false,
            "shardID": 0,
            "status": "active"
        },
        {
            "id": 2,
            "created": 12,
            "updated": 12,
            "hostID": "1fced3d3-91e9-44f3-830e-5a95b65a2287",
            "isNonVoting": false,
            "isWitness": false,
            "shardID": 0,
            "status": "active"
        },
        {
            "id": 3,
            "created": 13,
            "updated": 13,
            "hostID": "a74bf4be-df70-4c7a-9682-1458015d8c44",
            "isNonVoting": false,
            "isWitness": false,
            "shardID": 0,
            "status": "active"
        },
        {
            "id": 4,
            "created": 16,
            "updated": 16,
            "hostID": "a74bf4be-df70-4c7a-9682-1458015d8c44",
            "isNonVoting": false,
            "isWitness": false,
            "shardID": 1,
            "status": "active"
        },
        {
            "id": 5,
            "created": 17,
            "updated": 17,
            "hostID": "1fced3d3-91e9-44f3-830e-5a95b65a2287",
            "isNonVoting": false,
            "isWitness": false,
            "shardID": 1,
            "status": "active"
        },
        {
            "id": 6,
            "created": 18,
            "updated": 18,
            "hostID": "96567b03-7269-4c1e-94fd-653ef54e3544",
            "isNonVoting": false,
            "isWitness": false,
            "shardID": 1,
            "status": "active"
        },
        {
            "id": 7,
            "created": 20,
            "updated": 20,
            "hostID": "3de4eee9-d0d6-490e-a024-61108959b753",
            "isNonVoting": true,
            "isWitness": false,
            "shardID": 0,
            "status": "active"
        },
        {
            "id": 8,
            "created": 22,
            "updated": 22,
            "hostID": "3de4eee9-d0d6-490e-a024-61108959b753",
            "isNonVoting": true,
            "isWitness": false,
            "shardID": 1,
            "status": "active"
        }
    ],
    "ShardIndex": 1,
    "Shards": [
        {
            "id": 0,
            "created": 7,
            "updated": 20,
            "status": "new",
            "type": "github.com/logbn/zongzi/prime",
            "version": "v0.0.1"
        },
        {
            "id": 1,
            "created": 15,
            "updated": 22,
            "status": "new",
            "type": "github.com/logbn/zongzi-examples/controller",
            "version": "v0.0.1"
        }
    ]
}
```