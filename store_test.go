package main

import (
	"fmt"
	"os"
	"testing"
	"time"
)

const TestDBPath = "test.db"

var seconds = 0

func init() {
	Now = func() time.Time {
		return time.Unix(int64(seconds), 0)
	}
}

func TestStoreSubmit(t *testing.T) {
	defer os.Remove(TestDBPath)
	s, err := NewStore(TestDBPath)
	if err != nil {
		t.Error(err)
	}

	if err := s.Submit("foo", "bar", 1); err != nil {
		t.Error(err)
	}
}

func TestStoreList(t *testing.T) {
	defer os.Remove(TestDBPath)
	s, _ := NewStore(TestDBPath)
	s.Submit("foo", "bar", 1)
	s.Submit("foo", "baz", 1)
	s.Submit("foo", "qux", 1)
	if items, _ := s.List("foo"); len(items) != 3 {
		t.Error(items)
	} else if items[0] != "bar" || items[1] != "baz" || items[2] != "qux" {
		t.Error(items)
	}
}

func TestStoreQuery(t *testing.T) {
	defer os.Remove(TestDBPath)
	s, _ := NewStore(TestDBPath)
	s.Submit("foo", "bar", 1)
	s.Submit("foo", "bar", 2)
	s.Submit("foo", "bar", 3)
	s.Submit("foo", "bar", 10)
	if c, err := s.Query("foo", "bar"); err != nil {
		t.Error(err)
	} else if c.Values[BucketIndex("total")][0].Count != 4 {
		t.Error(c.Values[BucketIndex("total")])
	} else if c.Values[BucketIndex("total")][0].Value != 1+2+3+10 {
		t.Error(c.Values[BucketIndex("total")])
	}
}

func TestStoreRolling(t *testing.T) {
	defer os.Remove(TestDBPath)
	s, _ := NewStore(TestDBPath)

	seconds = 0
	s.Submit("foo", "bar", 1)

	seconds = 5
	s.Submit("foo", "bar", 1)
	s.Submit("foo", "bar", 1)

	c, _ := s.Query("foo", "bar")

	if c.Values[BucketIndex("total")][0].Count != 3 {
		t.Error(c.Values[BucketIndex("total")])
	} else if c.Values[BucketIndex("realtime")][0].Count != 2 {
		t.Error(c.Values[BucketIndex("realtime")])
	} else if c.Values[BucketIndex("realtime")][1].Count != 0 {
		t.Error(c.Values[BucketIndex("realtime")])
	} else if c.Values[BucketIndex("realtime")][5].Count != 1 {
		t.Error(c.Values[BucketIndex("realtime")])
	}

	seconds = 6
	s.Submit("foo", "bar", 1)

	c, _ = s.Query("foo", "bar")
	if c.Values[BucketIndex("total")][0].Count != 4 {
		t.Error(c.Values[BucketIndex("total")])
	} else if c.Values[BucketIndex("realtime")][0].Count != 1 {
		t.Error(c.Values[BucketIndex("realtime")])
	} else if c.Values[BucketIndex("realtime")][1].Count != 2 {
		t.Error(c.Values[BucketIndex("realtime")])
	} else if c.Values[BucketIndex("realtime")][5].Count != 0 {
		t.Error(c.Values[BucketIndex("realtime")])
	} else if c.Values[BucketIndex("realtime")][6].Count != 1 {
		t.Error(c.Values[BucketIndex("realtime")])
	}

	seconds = 1000
	s.Submit("foo", "bar", 1)

	c, _ = s.Query("foo", "bar")
	if c.Values[BucketIndex("total")][0].Count != 5 {
		t.Error(c.Values[BucketIndex("total")])
	} else if c.Values[BucketIndex("realtime")][0].Count != 1 {
		t.Error(c.Values[BucketIndex("realtime")])
	} else if c.Values[BucketIndex("realtime")][1].Count != 0 {
		t.Error(c.Values[BucketIndex("realtime")])
	}
}

func BenchmarkStoreSubmit(b *testing.B) {
	defer os.Remove(TestDBPath)
	s, _ := NewStore(TestDBPath)
	for i := 0; i < b.N; i++ {
		seconds = i
		s.Submit("foo", fmt.Sprintf("bar%d", i), 1)
	}
	fi, _ := os.Stat(TestDBPath)
	b.Log(fi.Size(), b.N)
}

func BenchmarkStoreQuery(b *testing.B) {
	defer os.Remove(TestDBPath)
	s, _ := NewStore(TestDBPath)
	s.Submit("foo", "bar", 1)
	for i := 0; i < b.N; i++ {
		s.Query("foo", "bar")
	}
}

func BenchmarkStoreList(b *testing.B) {
	defer os.Remove(TestDBPath)
	s, _ := NewStore(TestDBPath)
	for i := 0; i < 500; i++ {
		s.Submit("bar", fmt.Sprintf("bar%d", i), 1)
	}
	for i := 0; i < 100; i++ {
		s.Submit("foo", fmt.Sprintf("bar%d", i), 1)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.List("foo")
	}
}
