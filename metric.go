package incr

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"hash/fnv"
	"math"
	"strconv"

	hll "github.com/clarkduvall/hyperloglog"
)

type MetricResult map[string]float64

type Metric interface {
	// Adds new value to the metric
	Submit(value string) error
	// Returns metric report
	Report() MetricResult
	// Resets metric data preparing it for the next time interval
	Flush()
}

type Counter struct {
	Count int64
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
func (c *Counter) Report() MetricResult {
	return MetricResult{"count": float64(c.Count)}
}

// The next counter is reset to zero on each flush
func (c *Counter) Flush() {
	c.Count = 0
}

type Gauge struct {
	Count int64
	Value float64
	Min   float64
	Max   float64
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

func (g *Gauge) Report() MetricResult {
	return MetricResult{
		"count": float64(g.Count),
		"value": g.Value,
		"min":   g.Min,
		"max":   g.Max,
	}
}

func (g *Gauge) Flush() {
	g.Min = math.NaN()
	g.Max = math.NaN()
}

type HLL struct {
	*hll.HyperLogLogPlus
}
type Set struct {
	HLL       HLL
	Count     int64
	Precision byte
}

type SetReport struct {
	Count  int64 `json:"count"`
	Unique int64 `json:"uniq"`
}

func (s *Set) Submit(value string) error {
	s.Count++
	h := fnv.New64a()
	h.Write([]byte(value))
	s.HLL.Add(h)
	return nil
}

func (s *Set) Report() MetricResult {
	return MetricResult{
		"count":  float64(s.Count),
		"unique": float64(s.HLL.Count()),
	}
}

var configHLLPrecision byte = 7

func (s *Set) Flush() {
	s.HLL.HyperLogLogPlus, _ = hll.NewPlus(configHLLPrecision)
	s.Count = 0
}

func (h *HLL) MarshalJSON() ([]byte, error) {
	b := &bytes.Buffer{}
	if err := gob.NewEncoder(b).Encode(h.HyperLogLogPlus); err != nil {
		return nil, err
	} else {
		s := base64.StdEncoding.EncodeToString(b.Bytes())
		return []byte(`"` + s + `"`), nil
	}
}

func (h *HLL) UnmarshalJSON(b64 []byte) error {
	// TODO assert for quotes
	b64 = b64[1 : len(b64)-1]
	if b, err := base64.StdEncoding.DecodeString(string(b64)); err != nil {
		return err
	} else {
		err = gob.NewDecoder(bytes.NewBuffer(b)).Decode(&h.HyperLogLogPlus)
		if err != nil {
			return err
		}
		return nil
	}
}
