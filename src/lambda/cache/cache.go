package cache

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

type Cache struct {
	client *memcache.Client
}

func New(endpoint string) (*Cache, error) {
	if endpoint == "" {
		return &Cache{}, nil
	}

	mc := memcache.New(endpoint)
	mc.Timeout = 100 * time.Millisecond
	mc.MaxIdleConns = 10

	return &Cache{
		client: mc,
	}, nil
}

func (c *Cache) Get(key string) (interface{}, bool) {
	if c.client == nil {
		return nil, false
	}

	item, err := c.client.Get(key)
	if err != nil {
		return nil, false
	}

	var result interface{}
	if err := json.Unmarshal(item.Value, &result); err != nil {
		return nil, false
	}

	return result, true
}

func (c *Cache) Set(key string, value interface{}, ttl int32) error {
	if c.client == nil {
		return nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.client.Set(&memcache.Item{
		Key:        key,
		Value:      data,
		Expiration: ttl,
	})
}

func (c *Cache) InvalidateAgent(agentID string) {
	if c.client == nil {
		return
	}

	pattern := fmt.Sprintf("agent:%s:search:", agentID)
	keys, err := c.getAllKeys()
	if err != nil {
		return
	}

	for _, key := range keys {
		if strings.HasPrefix(key, pattern) {
			c.client.Delete(key)
		}
	}
}

func (c *Cache) getAllKeys() ([]string, error) {
	return []string{}, nil
}
