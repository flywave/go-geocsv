package geocsv

import (
	"encoding/csv"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/flywave/go-geom"
	"github.com/flywave/go-geom/general"
	"github.com/flywave/go-geom/wkt"

	"golang.org/x/text/encoding/simplifiedchinese"
)

var defaultCoordValue = float64(-9999)

type GeoCSV struct {
	file    *os.File
	reader  io.Reader
	headers []string
	rows    [][]string
	options GeoCSVOptions
}

type GeoCSVOptions struct {
	Fields   []string
	XField   string
	YField   string
	WKTField string
}

func NewGeoCSV() (gc *GeoCSV) {
	gc = &GeoCSV{}
	return
}

func (gc *GeoCSV) getReader() io.Reader {
	if gc.file != nil {
		return gc.file
	}
	return gc.reader
}

func (gc *GeoCSV) readRecords() (err error) {
	if gc.getReader() == nil {
		err = errors.New("file is nil")
		return
	}
	headerRead := false
	gbkDecoder := simplifiedchinese.GBK.NewDecoder()
	reader := csv.NewReader(gc.getReader())
	for {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			err = readErr
			return
		}
		encodeValues := make([]string, 0, len(record))
		for _, value := range record {
			var encodeValue string
			coding := GetStringEncoding(value)
			switch coding {
			case UTF8:
				encodeValue = value
			case GBK:
				encodingString, _ := gbkDecoder.Bytes([]byte(value))
				encodeValue = string(encodingString)
			default:
				if encodingString, decodeError := gbkDecoder.Bytes([]byte(value)); decodeError == nil {
					encodeValue = string(encodingString)
				} else {
					err = errors.New("file encoding is not supported")
					return
				}
			}
			encodeValue = strings.TrimSpace(encodeValue)

			encodeValue = strings.ReplaceAll(encodeValue, "\uFEFF", "")
			encodeValue = strings.TrimSpace(encodeValue)
			encodeValues = append(encodeValues, encodeValue)
		}
		if !headerRead {
			headerRead = true
			gc.headers = encodeValues
		} else {
			gc.rows = append(gc.rows, encodeValues)
		}
	}
	return
}

func (gc *GeoCSV) Valid() bool {
	if len(gc.headers) == 0 || gc.RowCount() == 0 {
		return false
	}
	if len(gc.options.WKTField) > 0 {
		for _, fieldName := range gc.headers {
			if fieldName == gc.options.WKTField {
				return true
			}
		}
	}
	if len(gc.options.XField) > 0 && len(gc.options.YField) > 0 {
		for _, fieldName := range gc.headers {
			xfield := false
			yfield := false
			if len(gc.options.XField) > 0 && fieldName == gc.options.XField {
				xfield = true
			} else if len(gc.options.YField) > 0 && fieldName == gc.options.YField {
				yfield = true
			}
			return xfield && yfield
		}
	}
	return false
}

func (gc *GeoCSV) RowCount() int {
	return len(gc.rows)
}

func (gc *GeoCSV) Feature(i int) *geom.Feature {
	if i < gc.RowCount() {
		var (
			lng      = defaultCoordValue
			lat      = defaultCoordValue
			geometry geom.Geometry
		)
		properties := map[string]interface{}{}

		for j, cell := range gc.rows[i] {
			fieldName := gc.headers[j]
			if len(gc.options.WKTField) > 0 && fieldName == gc.options.WKTField {
				if wktGeometry, _, wktError := wkt.DecodeWKT([]byte(cell)); wktError == nil {
					geometry = general.GeometryDataAsGeometry(wktGeometry)
				}
			} else if len(gc.options.XField) > 0 && fieldName == gc.options.XField {
				lng, _ = strconv.ParseFloat(cell, 64)
			} else if len(gc.options.YField) > 0 && fieldName == gc.options.YField {
				lat, _ = strconv.ParseFloat(cell, 64)
			}
			properties[fieldName] = cell
		}
		if geometry == nil && lng != defaultCoordValue && lat != defaultCoordValue {
			geometry = general.NewPoint([]float64{lng, lat})
		}
		if geometry != nil {
			feature := geom.NewFeature(geometry)
			feature.Properties = properties
			return feature
		}
	}
	return nil
}

func ReadFile(filePath string, options GeoCSVOptions) (gc *GeoCSV, err error) {
	gc = NewGeoCSV()
	gc.options = options
	if gc.file, err = os.Open(filePath); err != nil {
		return
	}
	defer gc.file.Close()
	if err = gc.readRecords(); err != nil {
		return
	}
	return
}

func Read(reader io.Reader, options GeoCSVOptions) (gc *GeoCSV, err error) {
	gc = NewGeoCSV()
	gc.options = options
	gc.reader = reader
	if err = gc.readRecords(); err != nil {
		return
	}
	return
}

func (gc *GeoCSV) ToFeatureCollection() (features *geom.FeatureCollection) {
	features = geom.NewFeatureCollection()
	for _, row := range gc.rows {
		var (
			lng      = defaultCoordValue
			lat      = defaultCoordValue
			geometry geom.Geometry
		)
		properties := map[string]interface{}{}

		for j, cell := range row {
			fieldName := gc.headers[j]
			if len(gc.options.WKTField) > 0 && fieldName == gc.options.WKTField {
				if wktGeometry, _, wktError := wkt.DecodeWKT([]byte(cell)); wktError == nil {
					geometry = general.GeometryDataAsGeometry(wktGeometry)
				}
			} else if len(gc.options.XField) > 0 && fieldName == gc.options.XField {
				lng, _ = strconv.ParseFloat(cell, 64)
			} else if len(gc.options.YField) > 0 && fieldName == gc.options.YField {
				lat, _ = strconv.ParseFloat(cell, 64)
			}
			properties[fieldName] = cell
		}
		if geometry == nil && lng != defaultCoordValue && lat != defaultCoordValue {
			geometry = general.NewPoint([]float64{lng, lat})
		}
		if geometry != nil {
			feature := geom.NewFeature(geometry)
			feature.Properties = properties
			features.Features = append(features.Features, feature)
		}
	}
	return
}
