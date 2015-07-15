package incr

import (
	"bytes"
	"encoding/gob"
	"hash/fnv"

	"github.com/zserge/hyperloglog"
)

// You may change these to 64-bit values, but it will take more memory per event
type (
	Time  int64
	Value float32
	Int   int32
)

// Aggregated event data presented in JSON-friendly format
type EventData struct {
	T      Time  `json:"time"`   // Unix timestamp
	Sum    Value `json:"sum"`    // Accumulated value
	Count  Int   `json:"value"`  // How many times the event was modified
	Unique Int   `json:"unique"` // How many unique clients changed the event
}

// Aggregation of different meters per event
const (
	Total = iota
	Live
	Hourly
	Daily
	Weekly

	NumMeters

	LiveCount    = 10
	LiveDuration = 10 // 10 seconds

	HourlyCount    = 7
	HourlyDuration = 60 * 60 * 4 // 4 hours

	DailyCount        = 7
	DailyDuration     = 60 * 60 * 24 // 1 day
	DailyHLLPrecision = 7            // 2^7 buckets, 5..7% error

	WeeklyCount        = 4
	WeeklyDuration     = 60 * 60 * 24 * 7 // 1 week
	WeeklyHLLPrecision = 8                // 2^8 buckets, 3..5% error
)

//
// Event is a collection of meters for different time frames with different logic
//
type Event [NumMeters]meter

func NewEvent() *Event {
	var e Event = [NumMeters]meter{
		&totalMeter{},
		newHistoryMeter(LiveDuration, LiveCount),
		newHistoryMeter(HourlyDuration, HourlyCount),
		newHLLMeter(DailyDuration, DailyCount, DailyHLLPrecision),
		newHLLMeter(WeeklyDuration, WeeklyCount, WeeklyHLLPrecision),
	}
	return &e
}

// Record data in each meter
func (e *Event) Add(t Time, value Value, sender string) {
	for _, m := range e {
		m.Add(t, value, sender)
	}
}

// Return data from the most precise meter holding the given time
func (e *Event) Data(t Time) []EventData {
	switch {
	case t <= 0:
		return e[Total].Data()
	case t <= LiveDuration:
		return e[Live].Data()
	case t <= HourlyDuration:
		return e[Hourly].Data()
	case t <= DailyDuration:
		return e[Daily].Data()
	default:
		return e[Weekly].Data()
	}
}

func (e *Event) GobEncode() ([]byte, error) {
	b := bytes.Buffer{}
	enc := gob.NewEncoder(&b)
	err := gobenc(nil, enc, e[Total], e[Live], e[Hourly], e[Daily], e[Weekly])
	return b.Bytes(), err
}

func (e *Event) GobDecode(b []byte) error {
	*e = *(NewEvent())
	dec := gob.NewDecoder(bytes.NewBuffer(b))
	return gobdec(nil, dec, &e[Total], &e[Live], &e[Hourly], &e[Daily], &e[Weekly])
}

// Meter interface: can accumulate numerical values and report aggregated data
type meter interface {
	Add(t Time, value Value, sender string)
	Data() []EventData
	gob.GobEncoder
	gob.GobDecoder
}

// "Total" meter: keeps accumulated metrics for the whole event lifespan
type totalMeter struct {
	sum   Value
	count Int
}

func (m *totalMeter) Add(t Time, value Value, sender string) {
	m.count++
	m.sum = m.sum + value
}

func (m *totalMeter) Data() []EventData {
	return []EventData{{Sum: m.sum, Count: m.count}}
}

func (m *totalMeter) GobEncode() ([]byte, error) {
	buf := bytes.Buffer{}
	err := gobenc(nil, gob.NewEncoder(&buf), m.sum, m.count)
	return buf.Bytes(), err
}

func (m *totalMeter) GobDecode(b []byte) error {
	return gobdec(nil, gob.NewDecoder(bytes.NewBuffer(b)), &m.sum, &m.count)
}

// meter with history: keeps N most recent metrics
type historyMeter struct {
	data   []totalMeter
	period Time
	start  Time
	index  Int
}

func newHistoryMeter(period Time, backlog int) *historyMeter {
	return &historyMeter{
		data:   make([]totalMeter, backlog, backlog),
		period: period,
		index:  0,
	}
}

func (m *historyMeter) Add(t Time, value Value, sender string) {
	size := Int(len(m.data))
	if t < m.start {
		for i := Int(0); i < size; i++ {
			if m.start-m.period*Time(i) <= t {
				index := Int((m.index + size - i) % size)
				m.data[index].count++
				m.data[index].sum += value
				break
			}
		}
	} else {
		for m.start+m.period-1 < t {
			m.index = (m.index + 1) % size
			m.start = m.start + m.period
			m.data[m.index].count = 0
			m.data[m.index].sum = 0
		}
		m.data[m.index].count++
		m.data[m.index].sum += value
	}
}

func (m *historyMeter) Data() []EventData {
	e := []EventData{}
	for i := 0; i < len(m.data); i++ {
		index := (int(m.index) + len(m.data) - i) % len(m.data)
		t := m.start - Time(i)*m.period
		if t < 0 {
			break
		}
		e = append(e, EventData{
			T:     t,
			Sum:   m.data[index].sum,
			Count: m.data[index].count,
		})
	}
	return e
}

func (m *historyMeter) GobEncode() ([]byte, error) {
	buf := bytes.Buffer{}
	err := gobenc(nil, gob.NewEncoder(&buf), m.period, m.start, m.index, m.data)
	return buf.Bytes(), err
}

func (m *historyMeter) GobDecode(b []byte) error {
	return gobdec(nil, gob.NewDecoder(bytes.NewBuffer(b)), &m.period, &m.start, &m.index, &m.data)
}

// History meter with HyperLogLog++ counter
type hllMeter struct {
	data   []uniqcounter
	index  Int
	start  Time
	period Time

	hll       *hyperloglog.HyperLogLogPlus
	precision byte
}
type uniqcounter struct {
	sum    Value
	count  Int
	unique Int
}

func newHLLMeter(period Time, backlog int, precision byte) *hllMeter {
	return &hllMeter{
		data:      make([]uniqcounter, backlog, backlog),
		precision: precision,
		period:    period,
		start:     0,
		index:     0,
	}
}

func (m *hllMeter) Add(t Time, value Value, sender string) {
	// Number of unique senders for older events is already stored as a single
	// number, so it can't be adjusted
	// That's why we process only newer events
	if t >= m.start {
		for m.start < t {
			m.index = (m.index + 1) % Int(len(m.data))
			m.start = m.start + m.period
			m.data[m.index].count = 0
			m.data[m.index].sum = 0
			m.data[m.index].unique = 0
			m.hll = nil
		}
		m.data[m.index].count++
		m.data[m.index].sum += value
		if sender != "" {
			if m.hll == nil {
				m.hll, _ = hyperloglog.NewPlus(m.precision)
			}
			h := fnv.New64a()
			h.Write([]byte(sender))
			m.hll.Add(h)
		}
		// FIXME This might be too slow, we should call Count() only when going to
		// the next time frame
		if m.hll != nil {
			m.data[m.index].unique = Int(m.hll.Count())
		}
	}
}

func (m *hllMeter) Data() []EventData {
	e := []EventData{}
	for i := 0; i < len(m.data); i++ {
		index := (int(m.index) + len(m.data) - i) % len(m.data)
		t := m.start - Time(i)*m.period
		if t < 0 {
			break
		}
		e = append(e, EventData{
			T:      t,
			Sum:    m.data[index].sum,
			Count:  m.data[index].count,
			Unique: m.data[index].unique,
		})
	}
	return e
}

func (m *hllMeter) GobEncode() ([]byte, error) {
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	err := gobenc(nil, enc, m.index, m.start, m.period, m.precision)
	if m.hll != nil {
		err = gobenc(err, enc, true, m.hll)
	} else {
		err = gobenc(err, enc, false)
	}
	return buf.Bytes(), err
}

func (m *hllMeter) GobDecode(b []byte) error {
	dec := gob.NewDecoder(bytes.NewBuffer(b))
	hasHLL := false
	err := gobdec(nil, dec, &m.index, &m.start, &m.period, &m.precision, &hasHLL)
	if hasHLL {
		err = gobdec(err, dec, m.hll)
	}
	return err
}

// Helper gob functions
func gobenc(err error, enc *gob.Encoder, v ...interface{}) error {
	if err != nil {
		return err
	}
	for _, field := range v {
		if err = enc.Encode(field); err != nil {
			return err
		}
	}
	return nil
}

func gobdec(err error, dec *gob.Decoder, v ...interface{}) error {
	if err != nil {
		return err
	}
	for _, field := range v {
		if err = dec.Decode(field); err != nil {
			return err
		}
	}
	return nil
}
