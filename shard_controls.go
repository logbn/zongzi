package zongzi

type ShardControls interface {
	ReplicaCreate(hostID string, shardID uint64, isNonVoting bool) (id uint64, err error)
	ReplicaDelete(replicaID uint64) error
}

func newShardControls(a *Agent) *shardControls {
	return &shardControls{a, false}
}

type shardControls struct {
	agent   *Agent
	updated bool
}

func (sc *shardControls) ReplicaCreate(hostID string, shardID uint64, isNonVoting bool) (id uint64, err error) {
	id, err = sc.agent.replicaCreate(hostID, shardID, isNonVoting)
	if err == nil {
		sc.updated = true
	}
	return
}

func (sc *shardControls) ReplicaDelete(replicaID uint64) (err error) {
	err = sc.agent.replicaDelete(replicaID)
	if err == nil {
		sc.updated = true
	}
	return
}
