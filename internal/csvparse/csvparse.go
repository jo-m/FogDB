// Package csvparse parses the MeteoSwiss open-data CSV files for the
// local-forecasting collection (parameter metadata, point metadata, and
// per-parameter forecast values).
//
// All files are semicolon-separated, Latin1 (ISO-8859-1) encoded, with CRLF
// line terminators. See https://opendatadocs.meteoswiss.ch/e-forecast-data/e4-local-forecast-data
package csvparse

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// newReader wraps r in a Latin1->UTF-8 decoder and returns a semicolon-aware
// csv.Reader configured to tolerate the MeteoSwiss file conventions.
func newReader(r io.Reader) *csv.Reader {
	decoded := transform.NewReader(r, charmap.ISO8859_1.NewDecoder())
	cr := csv.NewReader(decoded)
	cr.Comma = ';'
	cr.LazyQuotes = true
	cr.FieldsPerRecord = -1 // tolerate any column count; we validate per-row
	cr.ReuseRecord = true
	return cr
}

// parseFloatOrNaN parses s as a float; an empty string yields NaN, which the
// db layer translates to SQL NULL.
func parseFloatOrNaN(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return math.NaN(), nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("parse float %q: %w", s, err)
	}
	return v, nil
}

// parseInt64 parses s as an int64. An empty string returns 0.
func parseInt64(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse int %q: %w", s, err)
	}
	return v, nil
}

// utf8BOM is the UTF-8 byte-order mark; some sources prepend it to the first
// header cell.
const utf8BOM = "\uFEFF"

// columnIndex builds a map from header name to column index.
func columnIndex(header []string) map[string]int {
	out := make(map[string]int, len(header))
	for i, h := range header {
		out[strings.TrimSpace(strings.TrimPrefix(h, utf8BOM))] = i
	}
	return out
}

// requireCol returns the index of col in idx or an error if missing.
func requireCol(idx map[string]int, col string) (int, error) {
	i, ok := idx[col]
	if !ok {
		return 0, fmt.Errorf("missing required column %q", col)
	}
	return i, nil
}
