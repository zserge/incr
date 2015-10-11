package incr

import (
	"encoding/json"
	"math"
	"testing"
)

func TestCounter(t *testing.T) {
	c := &Counter{}
	c.Flush()

	if c.Count != 0 {
		t.Error(c.Count)
	}

	if err := c.Submit("10"); err != nil || c.Count != 10 {
		t.Error(c.Count, err)
	}
	if err := c.Submit("-1"); err != nil || c.Count != 9 {
		t.Error(c.Count, err)
	}
	if err := c.Submit("+2"); err != nil || c.Count != 11 {
		t.Error(c.Count, err)
	}
	if err := c.Submit("5"); err != nil || c.Count != 16 {
		t.Error(c.Count, err)
	}
	c.Flush()
	if c.Count != 0 {
		t.Error(c.Count)
	}
	if err := c.Submit("5"); err != nil || c.Count != 5 {
		t.Error(c.Count, err)
	}
	if err := c.Submit("-foo"); err == nil || c.Count != 5 {
		t.Error(c.Count, err)
	}

	report := c.Report()
	if r, ok := report.(*Counter); !ok {
		t.Error("bad report type")
	} else if r.Count != 5 {
		t.Error(r.Count)
	}
}

func TestGauge(t *testing.T) {
	g := &Gauge{}
	g.Flush()

	if g.Count != 0 || g.Value != 0.0 ||
		!math.IsNaN(g.Min) || !math.IsNaN(g.Max) {
		t.Error(g)
	}

	if err := g.Submit("2"); err != nil || g.Count != 1 || g.Value != 2.0 {
		t.Error(g, err)
	}
	if err := g.Submit("5"); err != nil || g.Count != 2 || g.Value != 5.0 {
		t.Error(g, err)
	}
	if err := g.Submit("+2"); err != nil || g.Count != 3 || g.Value != 7.0 {
		t.Error(g, err)
	}
	if err := g.Submit("-0"); err != nil || g.Count != 4 || g.Value != 7.0 {
		t.Error(g, err)
	}
	if err := g.Submit("-4"); err != nil || g.Count != 5 || g.Value != 3.0 {
		t.Error(g, err)
	}
	if g.Min != 2.0 || g.Max != 7.0 {
		t.Error(g)
	}

	if err := g.Submit("-foo"); err == nil || g.Count != 5 {
		t.Error(g, err)
	}

	report := g.Report()
	if r, ok := report.(*Gauge); !ok {
		t.Error("bad report type")
	} else if r.Value != 3.0 || r.Min != 2.0 || r.Max != 7.0 {
		t.Error(r)
	}
	g.Flush()

	if !math.IsNaN(g.Min) || !math.IsNaN(g.Max) {
		t.Error(g)
	} else if g.Value != 3.0 || g.Count != 5 {
		t.Error(g)
	}
}

func TestSet(t *testing.T) {
	s := &Set{Precision: 7}
	s.Flush()
	for _, word := range []string{"foo", "bar", "baz", "foo", "foo"} {
		if err := s.Submit(word); err != nil {
			t.Error(err)
		}
	}
	if r, ok := s.Report().(*SetReport); !ok {
		t.Error("bad report type")
	} else if r.Count != 5 || r.Unique != 3 {
		t.Error(r)
	}

	s.Flush()
	if r, ok := s.Report().(*SetReport); !ok {
		t.Error("bad report type")
	} else if r.Count != 0 || r.Unique != 0 {
		t.Error(r)
	}
}

func TestCounterPersist(t *testing.T) {
	c := &Counter{}
	c.Flush()
	c.Submit("1")
	c.Submit("2")

	b, err := json.Marshal(c)
	if err != nil {
		t.Error(err)
	}

	var res Counter
	if err := json.Unmarshal(b, &res); err != nil {
		t.Error(err)
	}

	if res.Count != 3 {
		t.Error(res)
	}
}

func TestGaugePersist(t *testing.T) {
	g := &Gauge{}
	g.Flush()
	g.Submit("0")
	g.Submit("-2")
	g.Submit("7")
	g.Submit("+5")

	b, err := json.Marshal(g)
	if err != nil {
		t.Error(err)
	}

	var res Gauge
	if err := json.Unmarshal(b, &res); err != nil {
		t.Error(err)
	}

	if res.Count != 4 || res.Value != 12.0 || res.Min != -2.0 || res.Max != 12.0 {
		t.Error(res)
	}
}

func TestSetPersist(t *testing.T) {
	s := &Set{Precision: 7}
	s.Flush()
	for _, word := range []string{"foo", "bar", "baz", "foo", "foo"} {
		if err := s.Submit(word); err != nil {
			t.Error(err)
		}
	}

	b, err := json.Marshal(s)
	if err != nil {
		t.Error(err)
	}

	var res Set
	if err := json.Unmarshal(b, &res); err != nil {
		t.Error(err)
	}

	if r, ok := res.Report().(*SetReport); !ok {
		t.Error("bad report type")
	} else if r.Count != 5 || r.Unique != 3 {
		t.Error(r)
	}

	res.Submit("bar")
	res.Submit("hello world")

	if r, ok := res.Report().(*SetReport); !ok {
		t.Error("bad report type")
	} else if r.Count != 7 || r.Unique != 4 {
		t.Error(r)
	}
}
