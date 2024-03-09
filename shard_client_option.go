package zongzi

type ShardClientOption func(*shardClient) error

func WithRetries(retries int) ShardClientOption {
	return func(c *shardClient) error {
		c.retries = retries
		return nil
	}
}
