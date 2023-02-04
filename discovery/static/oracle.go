package static

import (
	"github.com/logbn/zongzi"
	"github.com/logbn/zongzi/util"
)

type oracle struct {
	peers map[string]string
}

func NewOracle(peers map[string]string) *oracle {
	return &oracle{peers}
}

func (o *oracle) GetSeedList(agent zongzi.Agent) (seedList []string, err error) {
	return util.Keys(o.peers), nil
}

func (o *oracle) Peers() map[string]string {
	return o.peers
}
