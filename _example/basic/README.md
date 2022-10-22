# Zongzi - Basic Example

This basic example can be used to instantiate and initialize a 3 node cluster.

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

If performed correctly, you should see a snapshot dumped to stdout on node 1.
