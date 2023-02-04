PROJECTNAME := zongzi

DATA_DIR := /var/lib/zongzi


clean-data:
	@sudo mkdir -p $(DATA_DIR)
	@sudo chmod +w $(DATA_DIR)
	@rm -rf $(DATA_DIR)/node*
	@rm -rf $(DATA_DIR)/raft*

MULTICAST := 239.108.0.1:10813,239.108.0.1:10823,239.108.0.1:10833

SECRETS := df9feaeab4102aa677d38c75a64a56da6e069c31a7894d4dc0f05e034c969bb3

test-node-1:
	@go run *.go -n "test001" -d $(DATA_DIR)/node1 -r 127.0.0.1:10811 -g 127.0.0.1:10812 -dl 239.108.0.1:10813 -dm $(MULTICAST) -z us-west-1a -s $(SECRETS)

test-node-2:
	@go run *.go -n "test001" -d $(DATA_DIR)/node2 -r 127.0.0.1:10821 -g 127.0.0.1:10822 -dl 239.108.0.1:10823 -dm $(MULTICAST) -z us-west-1b -s $(SECRETS)

test-node-3:
	@go run *.go -n "test001" -d $(DATA_DIR)/node3 -r 127.0.0.1:10831 -g 127.0.0.1:10832 -dl 239.108.0.1:10833 -dm $(MULTICAST) -z us-west-1c -s $(SECRETS)

test-node-4:
	@go run *.go -n "test001" -d $(DATA_DIR)/node4 -r 127.0.0.1:10841 -g 127.0.0.1:10842 -dl 239.108.0.1:10843 -dm $(MULTICAST) -z us-west-1a -s $(SECRETS)

test-node-5:
	@go run *.go -n "test001" -d $(DATA_DIR)/node4 -r 127.0.0.1:10851 -g 127.0.0.1:10852 -dl 239.108.0.1:10853 -dm $(MULTICAST) -z us-west-1b -s $(SECRETS)

test-node-6:
	@go run *.go -n "test001" -d $(DATA_DIR)/node4 -r 127.0.0.1:10861 -g 127.0.0.1:10862 -dl 239.108.0.1:10863 -dm $(MULTICAST) -z us-west-1c -s $(SECRETS)

init-1:
# @bash -c 'echo "zongzi test001 INIT" > /dev/udp/239.108.0.1/10811'
	@bash -c 'echo "zongzi test001 INIT" > /dev/udp/127.0.0.1/10811'

init-2:
# @bash -c 'echo "zongzi test001 INIT" > /dev/udp/239.108.0.1/10821'
	@bash -c 'echo "zongzi test001 INIT" > /dev/udp/127.0.0.1/10821'

init-3:
# @bash -c 'echo "zongzi test001 INIT" > /dev/udp/239.108.0.1/10831'
	@bash -c 'echo "zongzi test001 INIT" > /dev/udp/127.0.0.1/10831'
