PROJECTNAME := zongzi

DATA_DIR := /var/lib/zongzi

clean-data:
	@sudo mkdir -p $(DATA_DIR)
	@sudo chmod +w $(DATA_DIR)
	@rm -rf $(DATA_DIR)/node*
	@rm -rf $(DATA_DIR)/raft*

PEERS := 127.0.0.1:17011,127.0.0.1:17021,127.0.0.1:17031

SECRETS := df9feaeab4102aa677d38c75a64a56da6e069c31a7894d4dc0f05e034c969bb3

test-node-1:
	@go run *.go -n "test001" -d $(DATA_DIR)/node1 -a 127.0.0.1:17011 -r 127.0.0.1:17012 -g 127.0.0.1:17013 -h :8001 -p $(PEERS) -z us-west-1a -s $(SECRETS)

test-node-2:
	@go run *.go -n "test001" -d $(DATA_DIR)/node2 -a 127.0.0.1:17021 -r 127.0.0.1:17022 -g 127.0.0.1:17023 -h :8002 -p $(PEERS) -z us-west-1b -s $(SECRETS)

test-node-3:
	@go run *.go -n "test001" -d $(DATA_DIR)/node3 -a 127.0.0.1:17031 -r 127.0.0.1:17032 -g 127.0.0.1:17033 -h :8003 -p $(PEERS) -z us-west-1c -s $(SECRETS)

test-node-4:
	@go run *.go -n "test001" -d $(DATA_DIR)/node4 -a 127.0.0.1:17041 -r 127.0.0.1:17042 -g 127.0.0.1:17043 -h :8004 -p $(PEERS) -z us-west-1a -s $(SECRETS)

test-node-5:
	@go run *.go -n "test001" -d $(DATA_DIR)/node5 -a 127.0.0.1:17051 -r 127.0.0.1:17052 -g 127.0.0.1:17053 -h :8005 -p $(PEERS) -z us-west-1b -s $(SECRETS)

test-node-6:
	@go run *.go -n "test001" -d $(DATA_DIR)/node6 -a 127.0.0.1:17061 -r 127.0.0.1:17062 -g 127.0.0.1:17063 -h :8006 -p $(PEERS) -z us-west-1c -s $(SECRETS)
