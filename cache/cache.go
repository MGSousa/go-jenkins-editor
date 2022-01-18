package cache

import (
	log "github.com/sirupsen/logrus"
)

type (
	// Driver interface to implement
	Driver interface {
		connect() error
		get(key string) (string, error)
		set(key string, value interface{}) (string, error)
		del(key string) (int64, error)
	}

	// Cache struct for type Driver
	Cache struct {
		driver Driver
	}
)

// Init provides internal cache
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
		c.driver = &BuntDB{
			Path: ":memory:",
		}
		if err := c.driver.connect(); err != nil {
			log.Fatalf("BuntDB: %s", err)
		}
	default:
		log.Warningf("Provider [%s] not implemented ", provider)
	}
}

// Set insert new key in cache
func (c *Cache) Set(key string, value interface{}) (string, error) {
	return c.driver.set(key, value)
}

// Get search for a key pattern
func (c *Cache) Get(key string) (string, error) {
	return c.driver.get(key)
}

// Del remove key from cache
func (c *Cache) Del(key string) (int64, error) {
	return c.driver.del(key)
}
