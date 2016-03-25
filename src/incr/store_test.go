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

func TestStoreIncr(t *testing.T) {
	defer os.Remove(TestDBPath)
	s, err := NewStore(TestDBPath)
	if err != nil {
		t.Error(err)
	}

	if err := s.Incr("foo", "bar"); err != nil {
		t.Error(err)
	}
}

func TestStoreList(t *testing.T) {
	defer os.Remove(TestDBPath)
	s, _ := NewStore(TestDBPath)
	s.Incr("foo", "bar")
	s.Incr("foo", "baz")
	s.Incr("foo", "qux")
	if items, _ := s.List("foo"); len(items) != 3 {
		t.Error(items)
	} else if items[0] != "bar" || items[1] != "baz" || items[2] != "qux" {
		t.Error(items)
	}
	if items, _ := s.List("badlist"); items == nil {
		t.Error(items)
	} else if len(items) != 0 {
		t.Error(items)
	}
}

func TestStoreQuery(t *testing.T) {
	defer os.Remove(TestDBPath)
	s, _ := NewStore(TestDBPath)
	s.Incr("foo", "bar")
	s.Incr("foo", "bar")
	s.Incr("foo", "bar")
	s.Incr("foo", "bar")
	if c, err := s.Query("foo", "bar"); err != nil {
		t.Error(err)
	} else if c.Values[BucketIndex("total")][0] != 4 {
		t.Error(c.Values[BucketIndex("total")])
	}
}

func TestStoreRolling(t *testing.T) {
	defer os.Remove(TestDBPath)
	s, _ := NewStore(TestDBPath)

	seconds = 0
	s.Incr("foo", "bar")

	seconds = 5
	s.Incr("foo", "bar")
	s.Incr("foo", "bar")

	c, _ := s.Query("foo", "bar")

	if c.Values[BucketIndex("total")][0] != 3 {
		t.Error(c.Values[BucketIndex("total")])
	} else if c.Values[BucketIndex("realtime")][0] != 2 {
		t.Error(c.Values[BucketIndex("realtime")])
	} else if c.Values[BucketIndex("realtime")][1] != 0 {
		t.Error(c.Values[BucketIndex("realtime")])
	} else if c.Values[BucketIndex("realtime")][5] != 1 {
		t.Error(c.Values[BucketIndex("realtime")])
	}

	seconds = 6
	s.Incr("foo", "bar")

	c, _ = s.Query("foo", "bar")
	if c.Values[BucketIndex("total")][0] != 4 {
		t.Error(c.Values[BucketIndex("total")])
	} else if c.Values[BucketIndex("realtime")][0] != 1 {
		t.Error(c.Values[BucketIndex("realtime")])
	} else if c.Values[BucketIndex("realtime")][1] != 2 {
		t.Error(c.Values[BucketIndex("realtime")])
	} else if c.Values[BucketIndex("realtime")][5] != 0 {
		t.Error(c.Values[BucketIndex("realtime")])
	} else if c.Values[BucketIndex("realtime")][6] != 1 {
		t.Error(c.Values[BucketIndex("realtime")])
	}

	seconds = 1000
	s.Incr("foo", "bar")

	c, _ = s.Query("foo", "bar")
	if c.Values[BucketIndex("total")][0] != 5 {
		t.Error(c.Values[BucketIndex("total")])
	} else if c.Values[BucketIndex("realtime")][0] != 1 {
		t.Error(c.Values[BucketIndex("realtime")])
	} else if c.Values[BucketIndex("realtime")][1] != 0 {
		t.Error(c.Values[BucketIndex("realtime")])
	}
}

func TestStoreHourly(t *testing.T) {
	defer os.Remove(TestDBPath)
	s, _ := NewStore(TestDBPath)

	seconds = 0
	s.Incr("foo", "bar")
	seconds = 5
	s.Incr("foo", "bar")
	seconds = 60
	s.Incr("foo", "bar")
	c, _ := s.Query("foo", "bar")
	if c.Values[BucketIndex("day")][0] != 3 {
		t.Error(c.Values[BucketIndex("day")])
	}
	seconds = 3600
	s.Incr("foo", "bar")

	c, _ = s.Query("foo", "bar")
	if c.Values[BucketIndex("day")][0] != 1 {
		t.Error(c.Values[BucketIndex("day")])
	} else if c.Values[BucketIndex("day")][1] != 3 {
		t.Error(c.Values[BucketIndex("day")])
	}
}

func TestStoreDaily(t *testing.T) {
	defer os.Remove(TestDBPath)
	s, _ := NewStore(TestDBPath)

	seconds = 0
	s.Incr("foo", "bar")
	seconds = 5
	s.Incr("foo", "bar")
	seconds = 60
	s.Incr("foo", "bar")
	c, _ := s.Query("foo", "bar")
	if c.Values[BucketIndex("month")][0] != 3 {
		t.Error(c.Values[BucketIndex("month")])
	}
	seconds = 3600
	s.Incr("foo", "bar")

	seconds = 24 * 3600
	s.Incr("foo", "bar")

	seconds = 48 * 3600
	s.Incr("foo", "bar")
	s.Incr("foo", "bar")

	c, _ = s.Query("foo", "bar")
	if c.Values[BucketIndex("month")][0] != 2 {
		t.Error(c.Values[BucketIndex("month")])
	} else if c.Values[BucketIndex("month")][1] != 1 {
		t.Error(c.Values[BucketIndex("month")])
	} else if c.Values[BucketIndex("month")][2] != 4 {
		t.Error(c.Values[BucketIndex("month")])
	}
}

func TestStoreMonthly(t *testing.T) {
	defer os.Remove(TestDBPath)
	s, _ := NewStore(TestDBPath)

	seconds = 0
	s.Incr("foo", "bar")
	seconds = 5
	s.Incr("foo", "bar")
	seconds = 60
	s.Incr("foo", "bar")
	c, _ := s.Query("foo", "bar")
	if c.Values[BucketIndex("year")][0] != 3 {
		t.Error(c.Values[BucketIndex("year")])
	}
	seconds = 24 * 3600
	s.Incr("foo", "bar")
	seconds = 48 * 3600
	s.Incr("foo", "bar")

	seconds = 40 * 24 * 3600
	s.Incr("foo", "bar")
	s.Incr("foo", "bar")

	c, _ = s.Query("foo", "bar")
	if c.Values[BucketIndex("year")][0] != 2 {
		t.Error(c.Values[BucketIndex("year")])
	} else if c.Values[BucketIndex("year")][1] != 5 {
		t.Error(c.Values[BucketIndex("year")])
	}
}

func BenchmarkStoreIncr(b *testing.B) {
	defer os.Remove(TestDBPath)
	s, _ := NewStore(TestDBPath)
	for i := 0; i < b.N; i++ {
		seconds = i
		s.Incr("foo", fmt.Sprintf("bar%d", i))
	}
	fi, _ := os.Stat(TestDBPath)
	b.Log(fi.Size(), b.N)
}

func BenchmarkStoreQuery(b *testing.B) {
	defer os.Remove(TestDBPath)
	s, _ := NewStore(TestDBPath)
	s.Incr("foo", "bar")
	for i := 0; i < b.N; i++ {
		s.Query("foo", "bar")
	}
}

func BenchmarkStoreList(b *testing.B) {
	defer os.Remove(TestDBPath)
	s, _ := NewStore(TestDBPath)
	for i := 0; i < 500; i++ {
		s.Incr("bar", fmt.Sprintf("bar%d", i))
	}
	for i := 0; i < 100; i++ {
		s.Incr("foo", fmt.Sprintf("bar%d", i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.List("foo")
	}
}
