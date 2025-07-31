package state

import (
	"encoding/json"
	"fmt"

	bolt "go.etcd.io/bbolt"
)

type Store struct {
	db *bolt.DB
}

func NewStore(path string) (*Store, error) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Namespace(name string) (*Namespace, error) {
	if err := s.db.Update(func(tx *bolt.Tx) error {
		_, e := tx.CreateBucketIfNotExists([]byte(name))
		return e
	}); err != nil {
		return nil, err
	}
	return &Namespace{s.db, []byte(name)}, nil
}

type Namespace struct {
	db     *bolt.DB
	bucket []byte
}

func (n *Namespace) Get(key string) ([]byte, error) {
	var v []byte
	err := n.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(n.bucket)
		if b == nil {
			return fmt.Errorf("bucket %q missing", n.bucket)
		}
		val := b.Get([]byte(key))
		if val != nil {
			v = append([]byte(nil), val...)
		}
		return nil
	})
	return v, err
}

func (n *Namespace) Put(key string, value []byte) error {
	return n.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(n.bucket)
		if b == nil {
			return fmt.Errorf("bucket %q missing", n.bucket)
		}
		return b.Put([]byte(key), value)
	})
}

func (n *Namespace) Delete(key string) error {
	return n.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(n.bucket)
		if b == nil {
			return fmt.Errorf("bucket %q missing", n.bucket)
		}
		return b.Delete([]byte(key))
	})
}

func (n *Namespace) GetString(key string) (string, error) {
	v, err := n.Get(key)
	if err != nil || v == nil {
		return "", err
	}
	return string(v), nil
}

func (n *Namespace) PutString(key, val string) error {
	return n.Put(key, []byte(val))
}

func (n *Namespace) GetJSON(key string, dest any) error {
	data, err := n.Get(key)
	if err != nil || data == nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func (n *Namespace) PutJSON(key string, src any) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return n.Put(key, data)
}
