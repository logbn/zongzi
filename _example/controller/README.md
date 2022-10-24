# Zongzi - Controller Example

This example can be used to instantiate and initialize a 3 node cluster and add 3 more nodes as nonvoting.

## Work in progress

Note that this example represents a very crude proof of concept.  
Expect all aspects of this API to change.

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

Then in a fourth terminal, run:
```
make init-1
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
            "ID": "49e45c6a-ef2d-4a06-a1fe-a19f9f265b02",
            "Replicas": {
                "1": 1,
                "14": 13
            },
            "Meta": {
                "zone": "us-west-1a"
            },
            "Status": "new"
        },
        {
            "ID": "4c3e9c67-fdbe-47d3-a262-dc955f98bf86",
            "Replicas": {
                "15": 13,
                "2": 1
            },
            "Meta": {
                "zone": "us-west-1b"
            },
            "Status": "new"
        },
        {
            "ID": "0733a542-5460-4ae8-a6dc-badb118d2696",
            "Replicas": {
                "16": 13,
                "3": 1
            },
            "Meta": {
                "zone": "us-west-1c"
            },
            "Status": "new"
        },
        {
            "ID": "2329637b-9016-4baf-a17f-88d9e67cf0c1",
            "Replicas": {
                "19": 1,
                "21": 13
            },
            "Meta": {
                "zone": "us-west-1a"
            },
            "Status": "new"
        }
    ],
    "Shards": [
        {
            "ID": 1,
            "Replicas": {
                "1": "49e45c6a-ef2d-4a06-a1fe-a19f9f265b02",
                "19": "2329637b-9016-4baf-a17f-88d9e67cf0c1",
                "2": "4c3e9c67-fdbe-47d3-a262-dc955f98bf86",
                "3": "0733a542-5460-4ae8-a6dc-badb118d2696"
            },
            "Type": "zongzi",
            "Status": "new"
        },
        {
            "ID": 13,
            "Replicas": {
                "14": "49e45c6a-ef2d-4a06-a1fe-a19f9f265b02",
                "15": "4c3e9c67-fdbe-47d3-a262-dc955f98bf86",
                "16": "0733a542-5460-4ae8-a6dc-badb118d2696",
                "21": "2329637b-9016-4baf-a17f-88d9e67cf0c1"
            },
            "Type": "banana",
            "Status": "new"
        }
    ],
    "Replicas": [
        {
            "ID": 1,
            "ShardID": 1,
            "HostID": "49e45c6a-ef2d-4a06-a1fe-a19f9f265b02",
            "Status": "new",
            "IsNonVoting": false,
            "IsWitness": false
        },
        {
            "ID": 2,
            "ShardID": 1,
            "HostID": "4c3e9c67-fdbe-47d3-a262-dc955f98bf86",
            "Status": "new",
            "IsNonVoting": false,
            "IsWitness": false
        },
        {
            "ID": 3,
            "ShardID": 1,
            "HostID": "0733a542-5460-4ae8-a6dc-badb118d2696",
            "Status": "new",
            "IsNonVoting": false,
            "IsWitness": false
        },
        {
            "ID": 14,
            "ShardID": 13,
            "HostID": "49e45c6a-ef2d-4a06-a1fe-a19f9f265b02",
            "Status": "new",
            "IsNonVoting": false,
            "IsWitness": false
        },
        {
            "ID": 15,
            "ShardID": 13,
            "HostID": "4c3e9c67-fdbe-47d3-a262-dc955f98bf86",
            "Status": "new",
            "IsNonVoting": false,
            "IsWitness": false
        },
        {
            "ID": 16,
            "ShardID": 13,
            "HostID": "0733a542-5460-4ae8-a6dc-badb118d2696",
            "Status": "new",
            "IsNonVoting": false,
            "IsWitness": false
        },
        {
            "ID": 19,
            "ShardID": 1,
            "HostID": "2329637b-9016-4baf-a17f-88d9e67cf0c1",
            "Status": "new",
            "IsNonVoting": false,
            "IsWitness": false
        },
        {
            "ID": 21,
            "ShardID": 13,
            "HostID": "2329637b-9016-4baf-a17f-88d9e67cf0c1",
            "Status": "new",
            "IsNonVoting": true,
            "IsWitness": false
        }
    ]
}
```