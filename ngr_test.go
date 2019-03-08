package ngr

import (
	"fmt"
	"github.com/pkg/errors"
	"math"
	"testing"
)

func TestSimpleConversions(t *testing.T) {
	table := []struct {
		ngr   GridRef
		point GridCoord
	}{
		{GridRef{"TQ", "30695", "80671"}, GridCoord{530695, 180671}}, // London
		{GridRef{"SH", "123", "124"}, GridCoord{212300, 312400}},
		{GridRef{"HU", "473", "414"}, GridCoord{447300, 1141400}},
		{GridRef{"SP", "08", "86"}, GridCoord{408000, 286000}},
		{GridRef{"XA", "0", "0"}, GridCoord{0, -100000}},
		{GridRef{"XD", "980", "240"}, GridCoord{398000, -76000}},
		{GridRef{"XD", "923", "208"}, GridCoord{392300, -79200}},
		{GridRef{"XD", "92356", "20839"}, GridCoord{392356, -79161}}, // St Helier
		{GridRef{"TQ", "", ""}, GridCoord{500000, 100000}},
		{GridRef{"NN", "166", "712"}, GridCoord{216600, 771200}},      // Ben Nevis
		{GridRef{"HU", "39668", "75316"}, GridCoord{439668, 1175316}}, // Sullom Voe oil terminal
	}

	// NGR to GridCoord
	for _, s := range table {
		ngr2, err := NewGridRefFromString(s.ngr.String())
		if err != nil {
			t.Fatalf("%v", errors.Wrap(err, "could not create new GridRef"))
		}
		if *ngr2 != s.ngr {
			t.Fatalf("Failed round trip on NGR->string->NGR: %s", s.ngr.String())
		}
		point, err := s.ngr.ToLatLon()
		if err != nil {
			t.Fatalf("%v", errors.Wrap(err, "could not convert NGR to LatLon"))
		}
		if point.Easting != s.point.Easting || point.Northing != s.point.Northing {
			t.Fatalf("Result does not match %s (%d, %d): (%d, %d)",
				s.ngr.String(),
				s.point.Easting, s.point.Northing,
				point.Easting, point.Northing)
		}
	}

	// GridCoord to NGR
	for _, s := range table {
		ngr, err := s.point.ToGridRef(s.ngr.DigitResolution())
		if err != nil {
			t.Fatalf("%v", errors.Wrap(err, "could not convert point GridCoord to GridRef"))
		}
		if *ngr != s.ngr {
			t.Fatalf("Result does not match (%s): (%s)", s.ngr, ngr)
		}
	}
}

// TestRounding is a misnomer. Any reduction in precision gives the south-west
// corner of the relevant tile in which the coordinate exists at that precision.
func TestRounding(t *testing.T) {
	table := []struct {
		inbound  []string
		point    GridCoord
		outbound string
	}{
		{[]string{"NT", "26002", "73723"}, GridCoord{326002, 673723}, "NT2673"},    // Edinburgh
		{[]string{"NW", "46298", "29866"}, GridCoord{146298, 529866}, "NW462298"},  // Belfast
		{[]string{"XD", "923", "208"}, GridCoord{392300, -79200}, "XD9230020800"},  // St Helier
		{[]string{"HU", "39668", "75316"}, GridCoord{439668, 1175316}, "HU396753"}, // Sullom Voe
	}

	for _, s := range table {
		ngr1 := GridRef{s.inbound[0], s.inbound[1], s.inbound[2]}
		if ngr1.String() != fmt.Sprintf("%s %s %s", s.inbound[0], s.inbound[1], s.inbound[2]) {
			t.Fatalf("Bad GridRef String() %s", ngr1.String())
		}
		ngr2, err := NewGridRefFromString(ngr1.String())
		if err != nil {
			t.Fatalf("%v", errors.Wrap(err, "could not create new GridRef"))
		}
		expected := fmt.Sprintf("%s %s %s", s.inbound[0], s.inbound[1], s.inbound[2])
		if ngr1.String() != expected {
			t.Fatalf("Something went wrong with the GridRef %s", expected)
		}
		if ngr1 != *ngr2 {
			t.Fatalf("Failure of NewGridRefFromString() \"%s\" vs \"%s\"", ngr1.String(), ngr2.String())
		}

		ngr3, err := s.point.ToGridRef(ngr2.DigitResolution())
		if err != nil {
			t.Fatalf("%v", err)
		}
		if *ngr2 != *ngr3 {
			t.Fatalf("Results do not match \"%s\" \"%s\"", ngr2, ngr3)
		}

		ngr5, err := NewGridRefFromString(s.outbound)
		if err != nil {
			t.Fatalf("%v", err)
		}
		ngr6, err := s.point.ToGridRef(ngr5.DigitResolution())
		if err != nil {
			t.Fatalf("%v", err)
		}
		if *ngr5 != *ngr6 {
			t.Fatalf("Results do not match \"%s\" \"%s\"", ngr5.String(), ngr6.String())
		}
	}
}

func TestGoodNgrString(t *testing.T) {
	goodValues := []string{
		"TQ",
		"TQ12",
		"TQ1234",
		"TQ123456",
		"TQ12345678",
		"TQ1234567890", // 1 metre resolution
	}
	for _, value := range goodValues {
		ngr, err := NewGridRefFromString(value)
		if err != nil {
			t.Fatalf("%v", err)
		}

		if _, err := ngr.ToLatLon(); err != nil {
			t.Fatalf("%v", err)
		}
	}
}

func TestBadNgrString(t *testing.T) {
	badValues := []string{
		"",
		"1",
		"T",
		"TQ1",
		"TQ123",
		"TQ12345",
		"TQ1234567",
		"TQ12345678901",
		"TQ1234567890133",
		"tq",
		"tq12",
	}
	for _, value := range badValues {
		_, err := NewGridRefFromString(value)
		if err == nil {
			t.Fatalf("Expected bad NGR value for NewGridRefFromString(\"%s\")", value)
		}
	}
}

func TestInvalidNGRMyriad(t *testing.T) {
	badMyriads := []string{
		"AA",
		"ZZ",
	}
	for _, myriad := range badMyriads {
		if _, err := NewGridRefFromString(myriad); err == nil {
			t.Fatalf("Expected bad NGR myriad for NewGridRefFromString(\"%s\")", myriad)
		}

		ngr := GridRef{myriad, "", ""}
		if _, err := ngr.ToLatLon(); err == nil {
			t.Fatalf("Expected bad NGR myriad for ToLatLon(\"%s\")", myriad)
		}
	}
}

func TestToNGR1(t *testing.T) {
	o := GridCoord{530695, 180671}
	// Check that these digits Pass
	goodDigitCounts := []int{0, 1, 2, 3, 4, 5}

	for _, nDigits := range goodDigitCounts {
		if _, err := o.ToGridRef(nDigits); err != nil {
			t.Fatalf("%v", err)
		}
	}

	// Check that these digits Fail
	badDigitCounts := []int{-1, 6, 7, 9, 11, 12, 13}
	for _, nDigits := range badDigitCounts {
		if _, err := o.ToGridRef(nDigits); err == nil {
			t.Fatalf("Expected bad digit count for GridCoord.ToGridRef(%d)", nDigits)
		}
	}
}

// TestToNGR2() tests just outside the southwest limits.
func TestToNGR2(t *testing.T) {
	ngr, err := NewGridRefFromString("XA")
	if err != nil {
		t.Fatal("Unexpected error")
	}
	o1, _ := ngr.ToLatLon()
	if err != nil {
		t.Fatal("Unexpected error")
	}
	// Check for each valid number of digits.
	for n := 1; n < 6; n++ {
		for _, o2 := range []GridCoord{
			{o1.Easting - 1, o1.Northing},
			{o1.Easting, o1.Northing - 1},
			{o1.Easting - 1, o1.Northing - 1},
		} {
			_, err := o2.ToGridRef(n * 2)
			if err == nil {
				t.Fatalf("Expected error GridCoord%v (nDigits=%d)", o2, n*2)
			}
		}
	}
}

// TestToNGR3() tests just outside the northeast limits.
func TestToNGR3(t *testing.T) {
	ngr, err := NewGridRefFromString("JH")
	if err != nil {
		t.Fatal("Unexpected error")
	}
	o1, _ := ngr.ToLatLon()
	if err != nil {
		t.Fatal("Unexpected error")
	}
	// Check for each valid number of digits.
	for n := 1; n < 6; n++ {
		for _, o2 := range []GridCoord{
			{o1.Easting + 100000, o1.Northing},
			{o1.Easting, o1.Northing + 100000},
			{o1.Easting + 100000, o1.Northing + 100000},
		} {
			_, err := o2.ToGridRef(n * 2)
			if err == nil {
				t.Fatalf("Expected error GridCoord%v (nDigits=%d)", o2, n*2)
			}
		}
	}
}

// TestCre checks that:
// - ngrCre matches all the myriads in the myriadOffsets
// - all the myriads in myriadOffsets are matched by ngrCre
func TestCre(t *testing.T) {
	p := make([]byte, 26) // Create A-Z
	for i := range p {
		p[i] = 'A' + byte(i)
	}

	for i := range p {
		for j := range p {
			myriad := []byte{p[i], p[j]}

			_, ok := myriadOffsets[string(myriad)]
			if ok {
				if !ngrCre.Match(myriad) {
					t.Fatalf("ngrCre not matching all myriads in myriadOffsets: %s", string(myriad))
				}
			} else if ngrCre.Match(myriad) {
				t.Fatalf("ngrCre matching myriad not in myriadOffsets: %s", string(myriad))
			}
		}
	}
}

// Check that the myriadOffsets and myriadTable contain the same myriads.
func TestLookupTables(t *testing.T) {
	myriadTableMyriads := make(map[string]bool)
	// fill myriadTableMyriads and check myriadTable for duplicates
	for i := range myriadTable {
		for j := range myriadTable[i] {
			myriad := myriadTable[i][j]
			if _, ok := myriadTableMyriads[myriad]; ok {
				t.Fatalf("Duplicated myriad \"%s\" in myriadTable", myriad)
			} else {
				myriadTableMyriads[myriad] = true
			}
		}
	}

	if len(myriadTableMyriads) != len(myriadOffsets) {
		t.Fatal("myriadTable and myriadLookup should contain the same number of myriads")
	}

	// Sets would be useful here...
	// check myriadTable and myriadOffsets contain the same myriads
	for myriad := range myriadTableMyriads {
		if _, ok := myriadOffsets[myriad]; !ok {
			t.Fatalf("Could not find myriad \"%s\" in myriadOffsets", myriad)
		}
	}
}

func TestGeodesy(t *testing.T) {
	//gridRef, err := NewGridRefFromString("SP3311178363")
	gridRef, err := NewGridRefFromString("TG 51409 13177")
	if err != nil {
		t.Fatalf("%v", err)
	}

	latlon := gridRef.ToWGS84()

	// The values were taken from the reference JavaScript implementation.
	if math.Abs(latlon.Lon-1.716038) > 0.000001 {
		t.Fatalf("Longitude out by %f", math.Abs(latlon.Lat-52.657968))
	}

	if math.Abs(latlon.Lat-52.657977) > 0.000001 {
		t.Fatalf("Latitude out by %f", math.Abs(latlon.Lat-52.657968))
	}
}
