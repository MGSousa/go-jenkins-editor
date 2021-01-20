package cache

import (
	log "github.com/sirupsen/logrus"
)

type (
	Driver interface {
		connect() error
		get(key string) (string, error)
		set(key string, value interface{}) (string, error)
		del(key string) (int64, error)
	}

	Cache struct {
		driver Driver
	}
)

// @TODO: add more kv providers
// Init internal cache
func (c *Cache) Init(provider string) {
	switch provider {
	case "redis":
		c.driver = &Redis{
			Address: "127.0.0.1:6379",
			DB:      0,
		}
		if err := c.driver.connect(); err != nil {
			log.Fatalf("Redis: %s", err)
		}
	case "buntdb":
	case "badger":
	default:
		log.Warningln("Not implemented")
	}
}

// Set
func (c *Cache) Set(key string, value interface{}) (string, error) {
	return c.driver.set(key, value)
}

// Keys search for a key pattern
func (c *Cache) Get(key string) (string, error) {
	return c.driver.get(key)
}

// Del remove key from cache
func (c *Cache) Del(key string) (int64, error) {
	return c.driver.del(key)
}