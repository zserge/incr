package incr

import (
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
