package ingest

import (
	"sort"
	"testing"
	"time"

	"jo-m.ch/go/fogdb/internal/db"
)

func TestKeepAtReferenceTime(t *testing.T) {
	t0 := time.Date(2026, 5, 19, 4, 0, 0, 0, time.UTC)
	rows := []db.ForecastRow{
		{LocationID: 1, ParameterID: 7, Timestamp: t0.Add(2 * time.Hour), Value: 22},
		{LocationID: 1, ParameterID: 7, Timestamp: t0.Add(-1 * time.Hour), Value: 9},
		{LocationID: 1, ParameterID: 7, Timestamp: t0, Value: 10}, // now-cast for loc 1
		{LocationID: 1, ParameterID: 7, Timestamp: t0.Add(time.Hour), Value: 15},
		{LocationID: 2, ParameterID: 7, Timestamp: t0.Add(3 * time.Hour), Value: 30},
		{LocationID: 2, ParameterID: 7, Timestamp: t0.Add(time.Hour), Value: 12}, // earliest >= ref for loc 2
		{LocationID: 3, ParameterID: 7, Timestamp: t0.Add(-2 * time.Hour), Value: 70},
		{LocationID: 3, ParameterID: 7, Timestamp: t0.Add(-5 * time.Hour), Value: 99}, // all before ref; least stale wins
		{LocationID: 4, ParameterID: 7, Timestamp: t0.Add(5 * time.Hour), Value: 99},  // only entry for loc 4
	}
	got := keepAtReferenceTime(rows, t0)

	if len(got) != 4 {
		t.Fatalf("len = %d, want 4", len(got))
	}

	sort.Slice(got, func(i, j int) bool { return got[i].LocationID < got[j].LocationID })

	want := []db.ForecastRow{
		{LocationID: 1, ParameterID: 7, Timestamp: t0, Value: 10},
		{LocationID: 2, ParameterID: 7, Timestamp: t0.Add(time.Hour), Value: 12},
		{LocationID: 3, ParameterID: 7, Timestamp: t0.Add(-2 * time.Hour), Value: 70},
		{LocationID: 4, ParameterID: 7, Timestamp: t0.Add(5 * time.Hour), Value: 99},
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("row %d = %+v, want %+v", i, got[i], w)
		}
	}
}

func TestKeepAtReferenceTimeEmpty(t *testing.T) {
	got := keepAtReferenceTime(nil, time.Now())
	if len(got) != 0 {
		t.Errorf("len(nil-input) = %d, want 0", len(got))
	}
}

func TestKeepAtReferenceTimePicksLeadZeroAcrossHorizon(t *testing.T) {
	// A single location with an hourly horizon spanning before and after the
	// reference time: the row at the reference time must win.
	t0 := time.Date(2026, 5, 19, 4, 0, 0, 0, time.UTC)
	rows := make([]db.ForecastRow, 0, 27)
	for h := -2; h <= 24; h++ {
		rows = append(rows, db.ForecastRow{
			LocationID:  1,
			ParameterID: 7,
			Timestamp:   t0.Add(time.Duration(h) * time.Hour),
			Value:       float64(h),
		})
	}
	got := keepAtReferenceTime(rows, t0)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Value != 0 || !got[0].Timestamp.Equal(t0) {
		t.Errorf("got %+v, want value 0 at reference time %v", got[0], t0)
	}
}
