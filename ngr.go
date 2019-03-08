// Package for use with Ordnance Survey grid references.
// Uses integer rather than floating point.
//
// https://en.wikipedia.org/wiki/Ordnance_Survey_National_Grid
//
// https://www.ordnancesurvey.co.uk/resources/maps-and-geographic-resources/the-national-grid.html
//
// GridCoord grid references are the measured from the southwest corner of the
// SV square - the British National Grid origin at 49° N, 2° W
// "an offshore point in the English Channel which lies between the island of
// Jersey and the French port of St. Malo"
//
// NGR uses the grid system. Within each square, easting and northing from
// the south west corner of the square are given numerically.
//
// Various agencies have assets in places like the Channel Islands, the
// North Sea & Atlantic Ocean - this explains the myriads outside of the
// normal Ordnance Survey grid.
//
// There are also assets in the North Sea. These assets are generally on oil
// rigs.
//
// Terminology
//  Myriad 100km × 100km square
//  Hectad 10km × 10km square
//  Tetrad 2km × 2km square
package ngr

import (
	"errors"
	"fmt"
	errors2 "github.com/pkg/errors"
	"github.com/recombinant/go-geodesy"
	"log"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// GridCoord coordinate version of GridRef
type GridCoord struct {
	Easting, Northing int
}

// GridRef contains the national grid reference with the myriad and offset
// from south west corner of said myriad. Use NewGridRefFromString() to construct.
type GridRef struct {
	myriad   string
	easting  string
	northing string
}

// NewGridRefFromString creates a GridRef from an NGR string.
func NewGridRefFromString(value string) (*GridRef, error) {
	gridRef := new(GridRef)

	if utf8.RuneCountInString(value) == 2 {
		gridRef.myriad = value // myriad only
	} else {
		match := ngrCre.FindStringSubmatch(value)
		// It must look like an NGR.
		if len(match) == 0 {
			return nil, errors.New(fmt.Sprintf("badly formatted NGR \"%s\"", value))
		} else {
			for i, name := range ngrCre.SubexpNames() {
				if i != 0 && name != "" && match[i] != "" {
					if name == "myriad" {
						gridRef.myriad = match[i]
					} else if strings.HasPrefix(name, "easting") {
						gridRef.easting = match[i]
					} else if strings.HasPrefix(name, "northing") {
						gridRef.northing = match[i]
					}
				}
			}
		}
	}

	// The myriad must exist.
	if _, ok := myriadOffsets[gridRef.myriad]; !ok {
		return nil, errors.New(fmt.Sprintf("unknown myriad \"%s\"", value[:2]))
	}

	if len(gridRef.easting) != len(gridRef.northing) {
		return nil, errors.New(fmt.Sprintf("mismatched GridRef easting=\"%s\" northing=\"%s\" lengths", gridRef.easting, gridRef.northing))
	}

	// gridRef{myriad, easting, northing}
	return gridRef, nil // Ok
}

func (ngr *GridRef) String() string {
	if ngr.easting != "" {
		return fmt.Sprintf("%s %s %s", ngr.myriad, ngr.easting, ngr.northing)
	} else {
		return ngr.myriad
	}
}

func (ngr *GridRef) DigitResolution() int {
	return len(ngr.easting)
}

// ngrCre is the compiled regular expression to match a Grid Reference.
// First is a legitimate myriad followed by an optional pair of numbers of matching length.
var ngrCre = regexp.MustCompile(`^(?P<myriad>([JH][F-H])|([NS][A-H])|([HNS][J-Z])|([OTY][A-C])|([OT][F-H])|([JOT][L-N])|([JOT][Q-S])|([JOT][V-X])|(X[A-E])) ?((?P<easting1>\d) ?(?P<northing1>\d)|(?P<easting2>\d{2}) ?(?P<northing2>\d{2})|(?P<easting3>\d{3}) ?(?P<northing3>\d{3})|(?P<easting4>\d{4}) ?(?P<northing4>\d{4})|(?P<easting5>\d{5}) ?(?P<northing5>\d{5}))?$`)

func (ngr *GridRef) ToLatLon() (*GridCoord, error) {
	myriadOffset, ok := myriadOffsets[ngr.myriad]
	if !ok {
		return nil, errors.New(fmt.Sprintf("unknown GridRef myriad \"%s\"", ngr.myriad))
	}

	if len(ngr.easting) != len(ngr.northing) {
		return nil, errors.New(fmt.Sprintf("mismatched GridRef easting=\"%s\" northing=\"%s\" lengths", ngr.easting, ngr.northing))
	}

	var easting, northing int
	var err error

	if len(ngr.easting) > 0 {
		easting, err = strconv.Atoi(ngr.easting)
		if err != nil {
			return nil, errors2.Wrap(err, "invalid digits in NGR easting")
		}
		northing, err = strconv.Atoi(ngr.northing)
		if err != nil {
			return nil, errors2.Wrap(err, "invalid digits in NGR northing")
		}
	}
	factor := int(math.Pow(10, float64(5-len(ngr.easting))))

	gridCoord := new(GridCoord)
	gridCoord.Easting = easting*factor + myriadOffset.Easting
	gridCoord.Northing = northing*factor + myriadOffset.Northing
	return gridCoord, nil // Ok
}

func (ngr *GridRef) ToWGS84() geodesy.LatLon {
	// integer
	ngrLatLon, err := ngr.ToLatLon()
	if err != nil {
		log.Fatalf("%v", errors2.Wrap(err, "could not convert WGS84 to LatLon"))
	}

	// floating point
	geodesyLatLon := geodesy.OsGridRef{Easting: float64(ngrLatLon.Easting), Northing: float64(ngrLatLon.Northing)}
	return *geodesyLatLon.OsGridToLatLon(geodesy.WGS84)
}

var formatLookup = map[int]struct {
	format string
	factor float64
}{
	0: {"", 0.0}, // Special case. Just the myriad. Factor irrelevant.
	1: {"%01d", 10000.0},
	2: {"%02d", 1000.0},
	3: {"%03d", 100.0},
	4: {"%04d", 10.0},
	5: {"%05d", 1.0},
}

func (coord GridCoord) ToGridRef(digitResolution int) (*GridRef, error) {

	config, ok := formatLookup[digitResolution]
	if !ok {
		return nil, errors.New(fmt.Sprintf("digitResolution should be 0, 1, 2, 3, 4 or 5 (not %d)", digitResolution))
	}

	// Indices into myriadTable (south west corner of myriads)
	// Compensate for Channel Islands being south of official grid origin
	i := coord.Easting
	if i < 0 {
		return nil, errors.New("outside UK - west of extents (northing untested)")
	}

	i /= 100000
	if i >= len(myriadTable) {
		return nil, errors.New("outside UK - east of extents (northing untested)")
	}

	// Offset for additional row in grid hack for Channel Islands
	j := coord.Northing + 100000
	if j < 0 {
		return nil, errors.New("outside UK - south of extents (Easting Ok)")
	}

	j /= 100000
	if j >= len(myriadTable[i]) {
		return nil, errors.New("outside UK - north of extents (Easting Ok)")
	}

	gridRef := new(GridRef)
	gridRef.myriad = myriadTable[i][j]

	if config.factor == 0.0 {
		return gridRef, nil // Ok, gridRef.easting == gridRef.northing == ""
	}

	eastingAsInt := int(math.Floor(float64(coord.Easting%100000) / config.factor))
	northingAsInt := int(math.Floor(float64((coord.Northing+100000)%100000) / config.factor))

	gridRef.easting = fmt.Sprintf(config.format, eastingAsInt)
	gridRef.northing = fmt.Sprintf(config.format, northingAsInt)

	return gridRef, nil // Ok
}

func (coord GridCoord) ToWGS84() geodesy.LatLon {
	// floating point
	latlon := geodesy.OsGridRef{Easting: float64(coord.Easting), Northing: float64(coord.Northing)}
	return *latlon.OsGridToLatLon(geodesy.WGS84)
}

//// This was used to create myriadOffsets
//func printMyriadOffsets() {
//	var a []string
//	for i1 := range myriadTable {
//		for j1 := range myriadTable[i1] {
//			i2 := 100000 * i1
//			j2 := 100000 * (j1 - 1)
//			s := fmt.Sprintf("\"%s\": {%d, %d},\n", myriadTable[i1][j1], i2, j2)
//			a = append(a, s)
//		}
//	}
//	sort.Strings(a)
//	for _, s := range a {
//		fmt.Print(s)
//	}
//}

// myriadOffsets was generated from simpler 2D slice myriadTable below.
// It gives the South West corner of the respective myriad.
var myriadOffsets = map[string]GridCoord{
	"HF": {0, 1300000}, "HG": {100000, 1300000}, "HH": {200000, 1300000}, "HJ": {300000, 1300000}, "HK": {400000, 1300000},
	"HL": {0, 1200000}, "HM": {100000, 1200000}, "HN": {200000, 1200000}, "HO": {300000, 1200000}, "HP": {400000, 1200000},
	"HQ": {0, 1100000}, "HR": {100000, 1100000}, "HS": {200000, 1100000}, "HT": {300000, 1100000}, "HU": {400000, 1100000},
	"HV": {0, 1000000}, "HW": {100000, 1000000}, "HX": {200000, 1000000}, "HY": {300000, 1000000}, "HZ": {400000, 1000000}, "JF": {500000, 1300000}, "JG": {600000, 1300000}, "JH": {700000, 1300000}, "JL": {500000, 1200000}, "JM": {600000, 1200000}, "JN": {700000, 1200000}, "JQ": {500000, 1100000}, "JR": {600000, 1100000}, "JS": {700000, 1100000}, "JV": {500000, 1000000}, "JW": {600000, 1000000}, "JX": {700000, 1000000},
	"NA": {0, 900000}, "NB": {100000, 900000}, "NC": {200000, 900000}, "ND": {300000, 900000}, "NE": {400000, 900000},
	"NF": {0, 800000}, "NG": {100000, 800000}, "NH": {200000, 800000}, "NJ": {300000, 800000}, "NK": {400000, 800000},
	"NL": {0, 700000}, "NM": {100000, 700000}, "NN": {200000, 700000}, "NO": {300000, 700000}, "NP": {400000, 700000},
	"NQ": {0, 600000}, "NR": {100000, 600000}, "NS": {200000, 600000}, "NT": {300000, 600000}, "NU": {400000, 600000},
	"NV": {0, 500000}, "NW": {100000, 500000}, "NX": {200000, 500000}, "NY": {300000, 500000}, "NZ": {400000, 500000}, "OA": {500000, 900000}, "OB": {600000, 900000}, "OC": {700000, 900000}, "OF": {500000, 800000}, "OG": {600000, 800000}, "OH": {700000, 800000}, "OL": {500000, 700000}, "OM": {600000, 700000}, "ON": {700000, 700000}, "OQ": {500000, 600000}, "OR": {600000, 600000}, "OS": {700000, 600000}, "OV": {500000, 500000}, "OW": {600000, 500000}, "OX": {700000, 500000},
	"SA": {0, 400000}, "SB": {100000, 400000}, "SC": {200000, 400000}, "SD": {300000, 400000}, "SE": {400000, 400000},
	"SF": {0, 300000}, "SG": {100000, 300000}, "SH": {200000, 300000}, "SJ": {300000, 300000}, "SK": {400000, 300000},
	"SL": {0, 200000}, "SM": {100000, 200000}, "SN": {200000, 200000}, "SO": {300000, 200000}, "SP": {400000, 200000},
	"SQ": {0, 100000}, "SR": {100000, 100000}, "SS": {200000, 100000}, "ST": {300000, 100000}, "SU": {400000, 100000},
	"SV": {0, 0}, "SW": {100000, 0}, "SX": {200000, 0}, "SY": {300000, 0}, "SZ": {400000, 0}, "TA": {500000, 400000}, "TB": {600000, 400000}, "TC": {700000, 400000}, "TF": {500000, 300000}, "TG": {600000, 300000}, "TH": {700000, 300000}, "TL": {500000, 200000}, "TM": {600000, 200000}, "TN": {700000, 200000}, "TQ": {500000, 100000}, "TR": {600000, 100000}, "TS": {700000, 100000}, "TV": {500000, 0}, "TW": {600000, 0}, "TX": {700000, 0},
	"XA": {0, -100000}, "XB": {100000, -100000}, "XC": {200000, -100000}, "XD": {300000, -100000}, "XE": {400000, -100000}, "YA": {500000, -100000}, "YB": {600000, -100000}, "YC": {700000, -100000},
}

// Myriads for ToGridRef. Channel Islands to North Sea. Not within the
// Ordnance Survey's proscribed range but certainly within the range
// of actual use.
//
// Also used to determine myriad offset. Compared to the actual grid it appears
// upside down.
var myriadTable = [][]string{
	{"XA", "SV", "SQ", "SL", "SF", "SA", "NV", "NQ", "NL", "NF", "NA", "HV", "HQ", "HL", "HF"},
	{"XB", "SW", "SR", "SM", "SG", "SB", "NW", "NR", "NM", "NG", "NB", "HW", "HR", "HM", "HG"},
	{"XC", "SX", "SS", "SN", "SH", "SC", "NX", "NS", "NN", "NH", "NC", "HX", "HS", "HN", "HH"},
	{"XD", "SY", "ST", "SO", "SJ", "SD", "NY", "NT", "NO", "NJ", "ND", "HY", "HT", "HO", "HJ"},
	{"XE", "SZ", "SU", "SP", "SK", "SE", "NZ", "NU", "NP", "NK", "NE", "HZ", "HU", "HP", "HK"},
	{"YA", "TV", "TQ", "TL", "TF", "TA", "OV", "OQ", "OL", "OF", "OA", "JV", "JQ", "JL", "JF"},
	{"YB", "TW", "TR", "TM", "TG", "TB", "OW", "OR", "OM", "OG", "OB", "JW", "JR", "JM", "JG"},
	{"YC", "TX", "TS", "TN", "TH", "TC", "OX", "OS", "ON", "OH", "OC", "JX", "JS", "JN", "JH"},
}
