package incr

import (
	"hash/fnv"
	"math"
	"strconv"

	hll "github.com/clarkduvall/hyperloglog"
)

type Metric interface {
	// Adds new value to the metric
	Submit(value string) error
	// Returns metric report
	Report() interface{}
	// Resets metric data preparing it for the next time interval
	Flush()
}

type Counter struct {
	Count int64 `json:"count"`
}

// Increments counter by the value specified by s.
func (c *Counter) Submit(s string) error {
	if incr, err := strconv.ParseInt(s, 10, 64); err != nil {
		return err
	} else {
		c.Count = c.Count + int64(incr)
	}
	return nil
}

// Reports accumulated count
func (c *Counter) Report() interface{} {
	return &Counter{c.Count}
}

// The next counter is reset to zero on each flush
func (c *Counter) Flush() {
	c.Count = 0
}

type Gauge struct {
	Count int64   `json:"count"`
	Value float64 `json:"value"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
}

func (g *Gauge) Submit(s string) error {
	if n, err := strconv.ParseInt(s, 10, 64); err != nil {
		return err
	} else {
		g.Count++
		if s[0] == '+' || s[0] == '-' {
			g.Value = g.Value + float64(n)
		} else {
			g.Value = float64(n)
		}
		if g.Value < g.Min || math.IsNaN(g.Min) {
			g.Min = g.Value
		}
		if g.Value > g.Max || math.IsNaN(g.Max) {
			g.Max = g.Value
		}
	}
	return nil
}

func (g *Gauge) Report() interface{} {
	return &Gauge{g.Count, g.Value, g.Min, g.Max}
}

func (g *Gauge) Flush() {
	g.Min = math.NaN()
	g.Max = math.NaN()
}

type Set struct {
	HLL          *hll.HyperLogLogPlus
	CachedReport SetReport
	Precision    byte
}

type SetReport struct {
	Count  int64 `json:"count"`
	Unique int64 `json:"uniq"`
}

func (s *Set) Submit(value string) error {
	s.CachedReport.Count++
	h := fnv.New64a()
	h.Write([]byte(value))
	s.HLL.Add(h)
	return nil
}

func (s *Set) Report() interface{} {
	s.CachedReport.Unique = int64(s.HLL.Count())
	return &SetReport{
		Count:  s.CachedReport.Count,
		Unique: s.CachedReport.Unique,
	}
}

func (s *Set) Flush() {
	s.HLL, _ = hll.NewPlus(s.Precision)
	s.CachedReport = SetReport{}
}