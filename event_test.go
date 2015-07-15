package incr

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

func now() Time {
	return Time(time.Now().Unix())
}

func TestTotalMeter(t *testing.T) {
	m := &totalMeter{}
	if len(m.Data()) != 1 {
		t.Error()
	}
	if d := m.Data(); d[0].Sum != 0 || d[0].Count != 0 {
		t.Error(d)
	}
	m.Add(now(), 1, "")
	if d := m.Data(); d[0].Sum != 1 || d[0].Count != 1 {
		t.Error(d)
	}
	m.Add(now(), 5.1, "")
	if d := m.Data(); d[0].Sum != 6.1 || d[0].Count != 2 {
		t.Error(d)
	}
}

func TestHistoryMeter(t *testing.T) {
	m := newHistoryMeter(1, 5)

	m.Add(3, 30, "")

	if d := m.Data(); reflect.DeepEqual(d, []EventData{
		{3, 30, 1, 0},
		{2, 0, 0, 0},
		{1, 0, 0, 0},
		{0, 0, 0, 0},
	}) == false {
		t.Error(d)
	}

	m.Add(2, 20, "")
	m.Add(4, 40, "")
	m.Add(0, 123, "")
	m.Add(2, 22, "")

	if d := m.Data(); reflect.DeepEqual(d, []EventData{
		{4, 40, 1, 0},
		{3, 30, 1, 0},
		{2, 42, 2, 0},
		{1, 0, 0, 0},
		{0, 123, 1, 0},
	}) == false {
		t.Error(d)
	}

	m.Add(1, 10, "")
	m.Add(5, 50, "")
	m.Add(2, -22, "")

	if d := m.Data(); reflect.DeepEqual(d, []EventData{
		{5, 50, 1, 0},
		{4, 40, 1, 0},
		{3, 30, 1, 0},
		{2, 20, 3, 0},
		{1, 10, 1, 0},
	}) == false {
		t.Error(d)
	}

	m.Add(7, 70, "")
	m.Add(6, 60, "")

	if d := m.Data(); reflect.DeepEqual(d, []EventData{
		{7, 70, 1, 0},
		{6, 60, 1, 0},
		{5, 50, 1, 0},
		{4, 40, 1, 0},
		{3, 30, 1, 0},
	}) == false {
		t.Error(d)
	}
}

func TestHLLMeter(t *testing.T) {
	m := newHLLMeter(1, 5, 8)
	m.Add(1, 10, "foo")
	m.Add(1, 20, "bar")
	m.Add(1, 20, "baz")
	m.Add(1, 20, "baz")
	m.Add(2, 0, "foo")
	m.Add(2, 0, "bar")
	m.Add(4, 0, "bar")
	if d := m.Data(); d[0].Unique != 1 || d[2].Unique != 2 || d[3].Unique != 3 {
		t.Error(d)
	}
}

func TestEvent(t *testing.T) {
	e := NewEvent()
	e.Add(0, 1, "foo")
	e.Add(0, 1, "bar")
	e.Add(1, 4, "foo")
	e.Add(2, 0, "baz")
	e.Add(LiveDuration, 1, "baz")
	e.Add(LiveDuration+1, 2, "baz")

	// Check total
	if !reflect.DeepEqual(e.Data(0), []EventData{{Sum: 9, Count: 6}}) {
		t.Error(e.Data(0))
	}

	// Check live, should be
	if !reflect.DeepEqual(e.Data(LiveDuration), []EventData{{T: LiveDuration, Sum: 3, Count: 2}, {T: 0, Sum: 6, Count: 4}}) {
		t.Error(e.Data(LiveDuration))
	}
	// TODO: test whole event
}

func TestEventGob(t *testing.T) {
	var e2 Event
	e1 := NewEvent()
	for i := 0; i < 100; i++ {
		e1.Add(Time(rand.Int()%(WeeklyDuration)), Value(rand.Float32()), "")
	}

	buf := bytes.Buffer{}
	if err := gob.NewEncoder(&buf).Encode(e1); err != nil {
		t.Error(err)
	}
	if err := gob.NewDecoder(&buf).Decode(&e2); err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(e1.Data(0), e2.Data(0)) {
		t.Error("event data differs")
	}
}

func BenchmarkEvent(b *testing.B) {
	e := NewEvent()
	for i := 0; i < b.N; i++ {
		e.Add(Time(rand.Int()%(DailyDuration)), Value(rand.Float32()), "")
	}
}

func BenchmarkEventHLL(b *testing.B) {
	e := NewEvent()
	for i := 0; i < b.N; i++ {
		e.Add(Time(rand.Int()%(DailyDuration)), Value(rand.Float32()), fmt.Sprintf("user%d", rand.Int()%10000))
	}
}

func BenchmarkEventData(b *testing.B) {
	e := NewEvent()
	for i := 0; i < 1000; i++ {
		e.Add(Time(rand.Int()%(DailyDuration)), Value(rand.Float32()), "")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Data(LiveDuration)
	}
}
