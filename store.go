package redirect

import (
	"encoding/json"
	"errors"
	"go.etcd.io/bbolt"
	"log"
	"time"
)

const bucket = "redirects"

type Store struct {
	db *bbolt.DB
}

type RedirectDestination struct {
	URL       string           `json:"url"`
	Options   *RedirectOptions `json:"options"`
	UpdatedAt time.Time        `json:"updated_at"`
}

type RedirectOptions struct {
}

func OpenStore(path string) (*Store, error) {
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}
	if err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucket))
		return err
	}); err != nil {
		return nil, err
	}
	return &Store{db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Upsert(key string, dest RedirectDestination) error {
	dest.UpdatedAt = time.Now()
	return s.db.Update(func(tx *bbolt.Tx) error {
		destJSON, err := json.Marshal(dest)
		if err != nil {
			return err
		}
		return tx.Bucket([]byte(bucket)).Put([]byte(key), destJSON)
	})
}

var ErrNotFound = errors.New("no such redirect")

func (s *Store) Get(key string) (dest *RedirectDestination, err error) {
	err = s.db.View(func(tx *bbolt.Tx) error {
		rawDest := tx.Bucket([]byte(bucket)).Get([]byte(key))
		if rawDest == nil {
			return ErrNotFound
		}
		if err := json.Unmarshal(rawDest, &dest); err != nil {
			return err
		}
		return nil
	})
	return dest, err
}

func (s *Store) Delete(key string) {
	_ = s.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte(bucket)).Delete([]byte(key))
	})
}

type Redirect struct {
	Key string
	RedirectDestination
}

func (s *Store) List(start string, limit int) (dests []Redirect, err error) {
	err = s.db.View(func(tx *bbolt.Tx) error {
		cur := tx.Bucket([]byte(bucket)).Cursor()
		for key, dest := cur.Seek([]byte(start)); key != nil; key, dest = cur.Next() {
			var parsed RedirectDestination
			if err := json.Unmarshal(dest, &parsed); err != nil {
				log.Println("unable to parse record for key", key)
				continue
			}
			dests = append(dests, Redirect{
				Key:                 string(key),
				RedirectDestination: parsed,
			})
			if limit > 0 && len(dests) == limit {
				break
			}
		}
		return nil
	})
	return dests, err
}
