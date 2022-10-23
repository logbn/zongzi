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
![image](https://user-images.githubusercontent.com/20638/197422450-8b033b3e-5c93-4cbb-875a-27246519ba4e.png)

Then in a fourth terminal, run:
```
make init-1
```

![image](https://user-images.githubusercontent.com/20638/197422392-42a4eb29-3231-4db8-a1ec-ac137b5d49d9.png)

If performed correctly, you should see a snapshot dumped as json to stdout on node 1.
