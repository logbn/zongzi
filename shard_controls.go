package zongzi

type Controls interface {
	Create(hostID string, shardID uint64, isNonVoting bool) (id uint64, err error)
	Delete(replicaID uint64) error
}

func newControls(a *Agent) *controls {
	return &controls{a, false}
}

type controls struct {
	agent   *Agent
	updated bool
}

func (sc *controls) Create(hostID string, shardID uint64, isNonVoting bool) (id uint64, err error) {
	id, err = sc.agent.replicaCreate(hostID, shardID, isNonVoting)
	if err == nil {
		sc.updated = true
	}
	return
}

func (sc *controls) Delete(replicaID uint64) (err error) {
	err = sc.agent.replicaDelete(replicaID)
	if err == nil {
		sc.updated = true
	}
	return
}
