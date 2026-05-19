package csvparse

import (
	"fmt"
	"io"
	"time"
)

// ForecastRow is one (point, valid-time, value) tuple from a per-parameter
// forecast CSV. The parameter is identified by the file/asset name, not the
// row, and is carried separately.
type ForecastRow struct {
	PointID     int64
	PointTypeID int64
	ValidTime   time.Time // UTC
	Value       float64
}

// ParseForecast parses a per-parameter forecast CSV. The value column is
// determined by the 4th header column (the file's parameter shortname). Rows
// whose value cell is empty are skipped (missing forecast value).
//
// All "Date" timestamps in the source file are UTC (per the open-data docs).
func ParseForecast(r io.Reader) ([]ForecastRow, error) {
	cr := newReader(r)
	header, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	if len(header) < 4 {
		return nil, fmt.Errorf("forecast header has %d columns, expected >= 4", len(header))
	}
	idx := columnIndex(header)
	pointIDCol, err := requireCol(idx, "point_id")
	if err != nil {
		return nil, err
	}
	pointTypeCol, err := requireCol(idx, "point_type_id")
	if err != nil {
		return nil, err
	}
	dateCol, err := requireCol(idx, "Date")
	if err != nil {
		return nil, err
	}
	// The value column is the 4th one; its name is the parameter shortname.
	valueCol := 3

	var out []ForecastRow
	for lineNo := 2; ; lineNo++ {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read row %d: %w", lineNo, err)
		}
		if len(rec) <= valueCol {
			return nil, fmt.Errorf("row %d has %d cols, expected >= %d", lineNo, len(rec), valueCol+1)
		}
		// Skip rows with no value.
		if rec[valueCol] == "" {
			continue
		}
		pid, err := parseInt64(rec[pointIDCol])
		if err != nil {
			return nil, fmt.Errorf("row %d point_id: %w", lineNo, err)
		}
		ptid, err := parseInt64(rec[pointTypeCol])
		if err != nil {
			return nil, fmt.Errorf("row %d point_type_id: %w", lineNo, err)
		}
		ts, err := parseMeteoDate(rec[dateCol])
		if err != nil {
			return nil, fmt.Errorf("row %d date: %w", lineNo, err)
		}
		v, err := parseFloatOrNaN(rec[valueCol])
		if err != nil {
			return nil, fmt.Errorf("row %d value: %w", lineNo, err)
		}
		out = append(out, ForecastRow{
			PointID:     pid,
			PointTypeID: ptid,
			ValidTime:   ts,
			Value:       v,
		})
	}
	return out, nil
}

// parseMeteoDate parses a "YYYYMMDDHHMM" UTC timestamp from the forecast CSVs.
func parseMeteoDate(s string) (time.Time, error) {
	t, err := time.ParseInLocation("200601021504", s, time.UTC)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse meteo date %q: %w", s, err)
	}
	return t, nil
}
