package cache

import (
	"github.com/tidwall/buntdb"
)

type BuntDB struct {
	client *buntdb.DB

	Path string
}

var err error

// connect to a new BuntDB instance
func (db *BuntDB) connect() error {
	if db.client == nil {
		if db.client, err = buntdb.Open(db.Path); err != nil {
			return err
		}
	}

	return nil
}

// set new key
func (db *BuntDB) set(key string, value interface{}) (res string, err error) {
	if err = db.client.Update(func(tx *buntdb.Tx) error {
		if _, _, err = tx.Set(key, value.(string), nil); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return "", err
	}
	return "Update succeded", nil
}

// get search for a key pattern
func (db *BuntDB) get(key string) (res string, err error) {
	db.client.View(func(tx *buntdb.Tx) error {
		if res, err = tx.Get(key, true); err != nil {
			return err
		}
		return nil
	})
	return
}

// del remove key from cache
func (db *BuntDB) del(key string) (i int64, err error) {
	return i, db.client.DropIndex(key)
}
