package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	"math"
	"time"

	"github.com/boltdb/bolt"
)

type Number float32

var ErrLimit = errors.New("limit exceeded")
var ErrNotFound = errors.New("not found")

var Now = time.Now

var IncrBucket = []byte("incr")

type Store interface {
	Submit(ns, name string, value Number) error
	List(ns string) ([]string, error)
	Query(ns, name string) (*Counter, error)
}

type store struct {
	db *bolt.DB
}

type Bucket struct {
	Name   string
	Period time.Duration
	Size   int
}

var Buckets = []Bucket{
	{"realtime", time.Second, 60},
	{"hour", time.Second * 60, 60},
	{"day", time.Second * 60 * 60, 24},
	{"month", time.Second * 60 * 60 * 24, 30},
	{"year", time.Second * 60 * 60 * 24 * 30, 12},
	{"total", time.Duration(math.MaxInt64), 1},
}

func BucketIndex(bucket string) int {
	switch bucket {
	case "realtime":
		return 0
	case "hour":
		return 1
	case "day":
		return 2
	case "month":
		return 3
	case "year":
		return 4
	case "total":
		return 5
	default:
		return -1
	}
}

type Counter struct {
	Atime  time.Time
	Values [][]Value
}

type Value struct {
	Value Number `json:"n"`
	Count int    `json:"c"`
}

func NewStore(path string) (Store, error) {
	if db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 5 * time.Second}); err != nil {
		return nil, err
	} else if err := db.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists(IncrBucket)
		return err
	}); err != nil {
		return nil, err
	} else {
		return &store{db: db}, nil
	}
}

func (s *store) Submit(ns, name string, value Number) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(IncrBucket)
		cnt := NewCounter(b.Get([]byte(ns + ":" + name)))
		cnt.Submit(value)
		return b.Put([]byte(ns+":"+name), cnt.Bytes())
	})
}

func (s *store) List(ns string) (list []string, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(IncrBucket).Cursor()
		prefix := []byte(ns + ":")
		for k, _ := c.Seek(prefix); bytes.HasPrefix(k, prefix); k, _ = c.Next() {
			list = append(list, string(bytes.TrimPrefix(k, prefix)))
		}
		return nil
	})
	return list, err
}

func (s *store) Query(ns, name string) (counter *Counter, err error) {
	err = s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(IncrBucket)
		counter = NewCounter(b.Get([]byte(ns + ":" + name)))
		return b.Put([]byte(ns+":"+name), counter.Bytes())
	})
	return counter, err
}

func NewCounter(data []byte) *Counter {
	c := Counter{}
	if data != nil {
		b := bytes.NewBuffer(data)
		gob.NewDecoder(b).Decode(&c)
	} else {
		c.Atime = Now()
		c.Values = [][]Value{}
		for _, bucket := range Buckets {
			c.Values = append(c.Values, make([]Value, bucket.Size))
		}
	}

	// Change atime
	atime := c.Atime
	c.Atime = Now()

	// Roll values
	for i, bucket := range Buckets {
		roll := int((c.Atime.Sub(atime)) / bucket.Period)
		if roll > 0 {
			if roll >= bucket.Size {
				c.Values[i] = make([]Value, bucket.Size)
			} else {
				c.Values[i] = append(make([]Value, roll), c.Values[i]...)[:bucket.Size]
			}
		}
	}

	return &c
}

func (c *Counter) Submit(n Number) {
	for i, _ := range Buckets {
		c.Values[i][0].Value = c.Values[i][0].Value + n
		c.Values[i][0].Count++
	}
}

func (c *Counter) Bytes() []byte {
	b := &bytes.Buffer{}
	if err := gob.NewEncoder(b).Encode(c); err != nil {
		panic(err)
	}
	return b.Bytes()
}
