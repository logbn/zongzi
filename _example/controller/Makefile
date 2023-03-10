PROJECTNAME := zongzi

DATA_DIR := /var/lib/zongzi

clean-data:
	@sudo mkdir -p $(DATA_DIR)
	@sudo chmod +w $(DATA_DIR)
	@rm -rf $(DATA_DIR)/node*
	@rm -rf $(DATA_DIR)/raft*

PEERS := 127.0.0.1:10811,127.0.0.1:10821,127.0.0.1:10831

SECRETS := df9feaeab4102aa677d38c75a64a56da6e069c31a7894d4dc0f05e034c969bb3

test-node-1:
	@go run *.go -n "test001" -d $(DATA_DIR)/node1 -l 127.0.0.1:10811 -g 127.0.0.1:10812 -r 127.0.0.1:10813 -p $(PEERS) -z us-west-1a -s $(SECRETS)

test-node-2:
	@go run *.go -n "test001" -d $(DATA_DIR)/node2 -l 127.0.0.1:10821 -g 127.0.0.1:10822 -r 127.0.0.1:10823 -p $(PEERS) -z us-west-1b -s $(SECRETS)

test-node-3:
	@go run *.go -n "test001" -d $(DATA_DIR)/node3 -l 127.0.0.1:10831 -g 127.0.0.1:10832 -r 127.0.0.1:10833 -p $(PEERS) -z us-west-1c -s $(SECRETS)

test-node-4:
	@go run *.go -n "test001" -d $(DATA_DIR)/node4 -l 127.0.0.1:10841 -g 127.0.0.1:10842 -r 127.0.0.1:10843 -p $(PEERS) -z us-west-1a -s $(SECRETS)

test-node-5:
	@go run *.go -n "test001" -d $(DATA_DIR)/node5 -l 127.0.0.1:10851 -g 127.0.0.1:10852 -r 127.0.0.1:10853 -p $(PEERS) -z us-west-1b -s $(SECRETS)

test-node-6:
	@go run *.go -n "test001" -d $(DATA_DIR)/node6 -l 127.0.0.1:10861 -g 127.0.0.1:10862 -r 127.0.0.1:10863 -p $(PEERS) -z us-west-1c -s $(SECRETS)
