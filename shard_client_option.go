package zongzi

type ClientOption func(*client) error

func WithRetries(retries int) ClientOption {
	return func(c *client) error {
		c.retries = retries
		return nil
	}
}

func WithWriteToLeader() ClientOption {
	return func(c *client) error {
		c.writeToLeader = true
		return nil
	}
}
