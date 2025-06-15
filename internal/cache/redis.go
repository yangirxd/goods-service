package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yangirxd/goods-service/internal/models"
)

type GoodsCache struct {
	client *redis.Client
}

func NewGoodsCache(client *redis.Client) *GoodsCache {
	return &GoodsCache{client: client}
}

func (c *GoodsCache) Set(ctx context.Context, key string, good *models.Good) error {
	data, err := json.Marshal(good)
	if err != nil {
		return fmt.Errorf("marshal good: %w", err)
	}

	err = c.client.Set(ctx, key, data, time.Minute).Err()
	if err != nil {
		return fmt.Errorf("set cache: %w", err)
	}

	return nil
}

func (c *GoodsCache) Get(ctx context.Context, key string) (*models.Good, error) {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("get cache: %w", err)
	}

	var good models.Good
	if err := json.Unmarshal(data, &good); err != nil {
		return nil, fmt.Errorf("unmarshal good: %w", err)
	}

	return &good, nil
}

func (c *GoodsCache) Delete(ctx context.Context, key string) error {
	err := c.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("delete cache: %w", err)
	}

	return nil
}

func GoodKey(id int64) string {
	return fmt.Sprintf("good:%d", id)
}
