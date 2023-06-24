package zongzi

import (
	"fmt"
	"strings"
)

type ShardOption func(*Shard) error

func WithName(name string) ShardOption {
	return func(s *Shard) error {
		s.Name = name
		return nil
	}
}

func WithPlacementMembers(n int, tags ...string) ShardOption {
	return func(s *Shard) error {
		s.Tags[`placement:member`] = fmt.Sprintf(`%d;%s`, n, strings.Join(tags, ";"))
		return nil
	}
}

func WithPlacementReplicas(group string, n int, tags ...string) ShardOption {
	return func(s *Shard) error {
		s.Tags[`placement:replica:`+group] = fmt.Sprintf(`%d;%s`, n, strings.Join(tags, ";"))
		return nil
	}
}

func WithPlacementVary(tagKeys ...string) ShardOption {
	return func(s *Shard) error {
		s.Tags[`placement:vary`] = strings.Join(tagKeys, ";")
		return nil
	}
}
