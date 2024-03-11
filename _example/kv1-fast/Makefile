PROJECTNAME := zongzi

DATA_DIR := /tmp/zongzi/example-kv1-fast

clean-data:
	@sudo mkdir -p $(DATA_DIR)
	@sudo chmod +w $(DATA_DIR)
	@rm -rf $(DATA_DIR)/node*
	@rm -rf $(DATA_DIR)/raft*

PEERS := 127.0.0.1:17013,127.0.0.1:17023,127.0.0.1:17033

test-node-1:
	@go run *.go -d $(DATA_DIR)/node1 -g 127.0.0.1:17011 -r 127.0.0.1:17012 -a 127.0.0.1:17013 -h :8001 -p $(PEERS) -z us-west-1a

test-node-2:
	@go run *.go -d $(DATA_DIR)/node2 -g 127.0.0.1:17021 -r 127.0.0.1:17022 -a 127.0.0.1:17023 -h :8002 -p $(PEERS) -z us-west-1c

test-node-3:
	@go run *.go -d $(DATA_DIR)/node3 -g 127.0.0.1:17031 -r 127.0.0.1:17032 -a 127.0.0.1:17033 -h :8003 -p $(PEERS) -z us-west-1f

test-node-4:
	@go run *.go -d $(DATA_DIR)/node4 -g 127.0.0.1:17041 -r 127.0.0.1:17042 -a 127.0.0.1:17043 -h :8004 -p $(PEERS) -z us-west-1a

test-node-5:
	@go run *.go -d $(DATA_DIR)/node5 -g 127.0.0.1:17051 -r 127.0.0.1:17052 -a 127.0.0.1:17053 -h :8005 -p $(PEERS) -z us-west-1c

test-node-6:
	@go run *.go -d $(DATA_DIR)/node6 -g 127.0.0.1:17061 -r 127.0.0.1:17062 -a 127.0.0.1:17063 -h :8006 -p $(PEERS) -z us-west-1f
