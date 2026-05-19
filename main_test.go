package main

import (
	"sort"
	"testing"
	"time"

	"jo-m.ch/go/nebeltracker/internal/db"
)

func TestKeepEarliestPerLocation(t *testing.T) {
	t0 := time.Date(2026, 5, 19, 4, 0, 0, 0, time.UTC)
	rows := []db.ForecastRow{
		{LocationID: 1, ParameterID: 7, Timestamp: t0.Add(2 * time.Hour), Value: 22},
		{LocationID: 1, ParameterID: 7, Timestamp: t0, Value: 10}, // earliest for loc 1
		{LocationID: 1, ParameterID: 7, Timestamp: t0.Add(time.Hour), Value: 15},
		{LocationID: 2, ParameterID: 7, Timestamp: t0.Add(3 * time.Hour), Value: 30},
		{LocationID: 2, ParameterID: 7, Timestamp: t0.Add(time.Hour), Value: 12}, // earliest for loc 2
		{LocationID: 3, ParameterID: 7, Timestamp: t0.Add(5 * time.Hour), Value: 99}, // only entry for loc 3
	}
	got := keepEarliestPerLocation(rows)

	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}

	sort.Slice(got, func(i, j int) bool { return got[i].LocationID < got[j].LocationID })

	want := []db.ForecastRow{
		{LocationID: 1, ParameterID: 7, Timestamp: t0, Value: 10},
		{LocationID: 2, ParameterID: 7, Timestamp: t0.Add(time.Hour), Value: 12},
		{LocationID: 3, ParameterID: 7, Timestamp: t0.Add(5 * time.Hour), Value: 99},
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("row %d = %+v, want %+v", i, got[i], w)
		}
	}
}

func TestKeepEarliestPerLocationEmpty(t *testing.T) {
	got := keepEarliestPerLocation(nil)
	if len(got) != 0 {
		t.Errorf("len(nil-input) = %d, want 0", len(got))
	}
}
