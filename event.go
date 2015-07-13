package incr

import (
	"hash/fnv"

	"github.com/clarkduvall/hyperloglog"
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

// Meter interface: can accumulate numerical values and report aggregated data
type meter interface {
	Add(t Time, value Value, sender string)
	Data() []EventData
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

// meter with history: keeps N most recent metrics
type historyMeter struct {
	data   []counter
	period Time
	start  Time
	index  Int
}
type counter struct {
	sum   Value
	count Int
}

func newHistoryMeter(period Time, backlog int) *historyMeter {
	return &historyMeter{
		data:   make([]counter, backlog, backlog),
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
		for m.start < t {
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

// History meter with HyperLogLog++ counter
type hllMeter struct {
	data   []uniqcounter
	index  Int
	start  Time
	period Time
	hll    *hyperloglog.HyperLogLogPlus
}
type uniqcounter struct {
	counter
	unique Int
}

func newHLLMeter(period Time, backlog int) *hllMeter {
	return &hllMeter{
		data:   make([]uniqcounter, backlog, backlog),
		period: period,
		start:  0,
		index:  0,
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
				m.hll, _ = hyperloglog.NewPlus(7)
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

	DailyCount    = 7
	DailyDuration = 60 * 60 * 24 // 1 day

	WeeklyCount    = 4
	WeeklyDuration = 60 * 60 * 24 * 7 // 1 week
)

type Event [NumMeters]meter

func NewEvent() Event {
	return [NumMeters]meter{
		&totalMeter{},
		newHistoryMeter(0, 10),
		newHistoryMeter(0, 10),
		newHLLMeter(0, 10),
		newHLLMeter(0, 10),
	}
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
	case t < LiveCount*LiveDuration:
		return e[Live].Data()
	case t < HourlyCount*HourlyDuration:
		return e[Hourly].Data()
	case t < DailyCount*DailyDuration:
		return e[Daily].Data()
	default:
		return e[Weekly].Data()
	}
}
