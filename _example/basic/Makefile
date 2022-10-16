PROJECTNAME := zongzi

DATA_DIR := /var/lib/zongzi


clean-data:
	@sudo mkdir -p $(DATA_DIR)
	@sudo chmod +w $(DATA_DIR)
	@rm -rf $(DATA_DIR)/node*
	@rm -rf $(DATA_DIR)/raft*

test-node-0:
	@go run main.go -c "test000" -u 127.0.0.1:10801 -i 1

TEST_BROADCAST := 239.108.0.1:10811,239.108.0.1:10821,239.108.0.1:10831
# TEST_BROADCAST := 127.0.0.1:10811,127.0.0.1:10821,127.0.0.1:10831

test-node-1:
	@go run main.go -c "test001" -d $(DATA_DIR)/node1 -r 127.0.0.1:10811 -g 127.0.0.1:10812 -u $(TEST_BROADCAST) -z us-west-1a -i 3

test-node-2:
	@go run main.go -c "test001" -d $(DATA_DIR)/node2 -r 127.0.0.1:10821 -g 127.0.0.1:10822 -u $(TEST_BROADCAST) -z us-west-1b

test-node-3:
	@go run main.go -c "test001" -d $(DATA_DIR)/node3 -r 127.0.0.1:10831 -g 127.0.0.1:10832 -u $(TEST_BROADCAST) -z us-west-1c
