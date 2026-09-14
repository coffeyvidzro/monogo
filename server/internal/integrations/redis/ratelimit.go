package redis

import (
	"github.com/ulule/limiter/v3"
	limiterredis "github.com/ulule/limiter/v3/drivers/store/redis"
)

func (c *Client) NewRateLimitStore() (limiter.Store, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}

	return limiterredis.NewStoreWithOptions(c.client, limiter.StoreOptions{
		Prefix: "leamout:rate-limit",
	})
}
