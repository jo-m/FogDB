package csvparse

import (
	"fmt"
	"io"
	"math"
)

// Parameter mirrors one row of ogd-local-forecasting_meta_parameters.csv.
type Parameter struct {
	Shortname     string
	DescriptionDE string
	DescriptionFR string
	DescriptionIT string
	DescriptionEN string
	GroupDE       string
	GroupFR       string
	GroupIT       string
	GroupEN       string
	Granularity   string
	Decimals      int
	Datatype      string
	Unit          string
}

// Location mirrors one row of ogd-local-forecasting_meta_point.csv.
// Numeric coordinate fields use math.NaN to signal "missing"; the db layer
// translates that to SQL NULL.
type Location struct {
	PointID     int64
	PointTypeID int64
	StationAbbr string
	PostalCode  string
	PointName   string
	PointTypeDE string
	PointTypeFR string
	PointTypeIT string
	PointTypeEN string
	HeightMASL  float64
	LV95East    float64
	LV95North   float64
	WGS84Lat    float64
	WGS84Lon    float64
}

// ParseParameters reads the meta-parameters CSV from r.
func ParseParameters(r io.Reader) ([]Parameter, error) {
	cr := newReader(r)
	header, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	idx := columnIndex(header)

	cols := map[string]int{}
	for _, name := range []string{
		"parameter_shortname",
		"parameter_description_de", "parameter_description_fr",
		"parameter_description_it", "parameter_description_en",
		"parameter_group_de", "parameter_group_fr",
		"parameter_group_it", "parameter_group_en",
		"parameter_granularity", "parameter_decimals",
		"parameter_datatype", "parameter_unit",
	} {
		i, err := requireCol(idx, name)
		if err != nil {
			return nil, err
		}
		cols[name] = i
	}

	var out []Parameter
	for lineNo := 2; ; lineNo++ {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read row %d: %w", lineNo, err)
		}
		dec, err := parseInt64(rec[cols["parameter_decimals"]])
		if err != nil {
			return nil, fmt.Errorf("row %d decimals: %w", lineNo, err)
		}
		out = append(out, Parameter{
			Shortname:     rec[cols["parameter_shortname"]],
			DescriptionDE: rec[cols["parameter_description_de"]],
			DescriptionFR: rec[cols["parameter_description_fr"]],
			DescriptionIT: rec[cols["parameter_description_it"]],
			DescriptionEN: rec[cols["parameter_description_en"]],
			GroupDE:       rec[cols["parameter_group_de"]],
			GroupFR:       rec[cols["parameter_group_fr"]],
			GroupIT:       rec[cols["parameter_group_it"]],
			GroupEN:       rec[cols["parameter_group_en"]],
			Granularity:   rec[cols["parameter_granularity"]],
			Decimals:      int(dec),
			Datatype:      rec[cols["parameter_datatype"]],
			Unit:          rec[cols["parameter_unit"]],
		})
	}
	return out, nil
}

// ParseLocations reads the meta-point CSV from r.
func ParseLocations(r io.Reader) ([]Location, error) {
	cr := newReader(r)
	header, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	idx := columnIndex(header)

	cols := map[string]int{}
	for _, name := range []string{
		"point_id", "point_type_id", "station_abbr", "postal_code", "point_name",
		"point_type_de", "point_type_fr", "point_type_it", "point_type_en",
		"point_height_masl",
		"point_coordinates_lv95_east", "point_coordinates_lv95_north",
		"point_coordinates_wgs84_lat", "point_coordinates_wgs84_lon",
	} {
		i, err := requireCol(idx, name)
		if err != nil {
			return nil, err
		}
		cols[name] = i
	}

	getFloat := func(rec []string, col string) (float64, error) {
		v, err := parseFloatOrNaN(rec[cols[col]])
		if err != nil {
			return math.NaN(), err
		}
		return v, nil
	}

	var out []Location
	for lineNo := 2; ; lineNo++ {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read row %d: %w", lineNo, err)
		}
		pid, err := parseInt64(rec[cols["point_id"]])
		if err != nil {
			return nil, fmt.Errorf("row %d point_id: %w", lineNo, err)
		}
		ptid, err := parseInt64(rec[cols["point_type_id"]])
		if err != nil {
			return nil, fmt.Errorf("row %d point_type_id: %w", lineNo, err)
		}
		hgt, err := getFloat(rec, "point_height_masl")
		if err != nil {
			return nil, fmt.Errorf("row %d height: %w", lineNo, err)
		}
		lvE, err := getFloat(rec, "point_coordinates_lv95_east")
		if err != nil {
			return nil, fmt.Errorf("row %d lv95_east: %w", lineNo, err)
		}
		lvN, err := getFloat(rec, "point_coordinates_lv95_north")
		if err != nil {
			return nil, fmt.Errorf("row %d lv95_north: %w", lineNo, err)
		}
		lat, err := getFloat(rec, "point_coordinates_wgs84_lat")
		if err != nil {
			return nil, fmt.Errorf("row %d wgs84_lat: %w", lineNo, err)
		}
		lon, err := getFloat(rec, "point_coordinates_wgs84_lon")
		if err != nil {
			return nil, fmt.Errorf("row %d wgs84_lon: %w", lineNo, err)
		}
		out = append(out, Location{
			PointID:     pid,
			PointTypeID: ptid,
			StationAbbr: rec[cols["station_abbr"]],
			PostalCode:  rec[cols["postal_code"]],
			PointName:   rec[cols["point_name"]],
			PointTypeDE: rec[cols["point_type_de"]],
			PointTypeFR: rec[cols["point_type_fr"]],
			PointTypeIT: rec[cols["point_type_it"]],
			PointTypeEN: rec[cols["point_type_en"]],
			HeightMASL:  hgt,
			LV95East:    lvE,
			LV95North:   lvN,
			WGS84Lat:    lat,
			WGS84Lon:    lon,
		})
	}
	return out, nil
}
