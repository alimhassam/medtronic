package medtronic

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// Force timezone to match test data.
func init() {
	os.Setenv("TZ", "America/New_York")
}

func TestTimeOfDay(t *testing.T) {
	cases := []struct {
		s   string
		t   TimeOfDay
		err error
	}{
		{"00:00", 0, nil},
		{"12:00", durationToTimeOfDay(12 * time.Hour), nil},
		{"23:59", durationToTimeOfDay(24*time.Hour - 1*time.Minute), nil},
		{"01:02:03", 0, fmt.Errorf("")},
		{"24:00", 0, fmt.Errorf("")},
		{"01:60", 0, fmt.Errorf("")},
	}
	for _, c := range cases {
		t.Run(c.s, func(t *testing.T) {
			if c.err == nil {
				s := c.t.String()
				if s != c.s {
					t.Errorf("%v.String() == %v, want %v", c.t, s, c.s)
				}
			}
			td, err := parseTimeOfDay(c.s)
			if err == nil {
				if c.err == nil {
					if td == c.t {
						return
					} else {
						t.Errorf("parseTimeOfDay(%s) == %v, want %v", c.s, td, c.t)
					}
				} else {
					t.Errorf("parseTimeOfDay(%s) == %v, want error", c.s, td)
				}
			} else {
				if c.err != nil {
					return
				} else {
					t.Errorf("parseTimeOfDay(%s) == %v, want %v", c.s, err, c.t)
				}
			}
		})
	}

}

func TestHalfHours(t *testing.T) {
	cases := []struct {
		t uint8
		d time.Duration
	}{
		{0, 0},
		{1, 30 * time.Minute},
		{3, 90 * time.Minute},
		{4, 2 * time.Hour},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%02d", c.t), func(t *testing.T) {
			d := halfHoursToDuration(c.t)
			if d != Duration(c.d) {
				t.Errorf("halfHoursToDuration(%d) == %v, want %v", c.t, d, c.d)
			}
		})
	}
}

var layouts = []string{
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04",
}

func parseTime(s string) time.Time {
	var t time.Time
	var err error
	for _, layout := range layouts {
		t, err = time.ParseInLocation(layout, s, time.Local)
		if err == nil {
			return t
		}
	}
	panic(err)
}

func TestSinceMidnight(t *testing.T) {
	cases := []struct {
		t time.Time
		d TimeOfDay
	}{
		{parseTime("2015-01-01T09:00"), durationToTimeOfDay(9 * time.Hour)},
		{parseTime("2016-03-15T10:00:00.5"), durationToTimeOfDay(10*time.Hour + 500*time.Millisecond)},
		{parseTime("2016-06-15T20:30"), durationToTimeOfDay(20*time.Hour + 30*time.Minute)},
		{parseTime("2010-11-30T23:59:59.999"), durationToTimeOfDay(24*time.Hour - time.Millisecond)},
		// DST changes
		{parseTime("2016-03-13T01:00"), durationToTimeOfDay(1 * time.Hour)},
		{parseTime("2016-03-13T03:00"), durationToTimeOfDay(3 * time.Hour)},
		{parseTime("2016-03-13T12:00"), durationToTimeOfDay(12 * time.Hour)},
		{parseTime("2016-11-06T01:00"), durationToTimeOfDay(1 * time.Hour)},
		{parseTime("2016-11-06T02:00"), durationToTimeOfDay(2 * time.Hour)},
		{parseTime("2016-11-06T03:00"), durationToTimeOfDay(3 * time.Hour)},
		{parseTime("2016-11-06T23:00"), durationToTimeOfDay(23 * time.Hour)},
		{parseTime("2016-11-06T23:30"), durationToTimeOfDay(23*time.Hour + 30*time.Minute)},
	}
	for _, c := range cases {
		t.Run(c.t.Format(time.Kitchen), func(t *testing.T) {
			d := sinceMidnight(c.t)
			if d != c.d {
				// Print TimeOfDay as underlying time.Duration.
				t.Errorf("sinceMidnight(%v) == %v, want %v", c.t, time.Duration(d), time.Duration(c.d))
			}
		})
	}
}

func TestDecodeTime(t *testing.T) {
	cases := []struct {
		b []byte
		t time.Time
	}{
		{[]byte{0x1F, 0x40, 0x00, 0x01, 0x05}, parseTime("2005-01-01T00:00:31")},
		{[]byte{0x75, 0xB7, 0x13, 0x04, 0x10}, parseTime("2016-06-04T19:55:53")},
		{[]byte{0x5D, 0xB3, 0x0F, 0x06, 0x10}, parseTime("2016-06-06T15:51:29")},
		{[]byte{0x40, 0x94, 0x12, 0x0F, 0x10}, parseTime("2016-06-15T18:20:00")},
	}
	for _, c := range cases {
		t.Run(c.t.Format(time.Kitchen), func(t *testing.T) {
			ts := time.Time(decodeTime(c.b))
			if !ts.Equal(c.t) {
				t.Errorf("decodeTime(% X) == %v, want %v", c.b, ts, c.t)
			}
		})
	}
}

func TestDecodeDate(t *testing.T) {
	cases := []struct {
		b []byte
		t time.Time
	}{
		{[]byte{0xBF, 0x0F}, parseTime("2015-10-31T00:00")},
		{[]byte{0x78, 0x10}, parseTime("2016-06-24T00:00")},
	}
	for _, c := range cases {
		t.Run(c.t.Format("2006-01-02"), func(t *testing.T) {
			ts := time.Time(decodeDate(c.b))
			if !ts.Equal(c.t) {
				t.Errorf("decodeDate(% X) == %v, want %v", c.b, ts, c.t)
			}
		})
	}
}

func TestDecodeCGMTime(t *testing.T) {
	cases := []struct {
		b []byte
		t time.Time
	}{
		{[]byte{0x8D, 0x9B, 0x1D, 0x0C}, parseTime("2012-10-29T13:27")},
		{[]byte{0x0B, 0xAE, 0x0A, 0x0E}, parseTime("2014-02-10T11:46")},
		{[]byte{0x4F, 0x5B, 0x13, 0x8F}, parseTime("2015-05-19T15:27")},
		{[]byte{0x14, 0xB6, 0x28, 0x10}, parseTime("2016-02-08T20:54")},
	}
	for _, c := range cases {
		t.Run(c.t.Format(time.Kitchen), func(t *testing.T) {
			ts := time.Time(decodeCGMTime(c.b))
			if !ts.Equal(c.t) {
				t.Errorf("decodeCGMTime(% X) == %v, want %v", c.b, ts, c.t)
			}
		})
	}
}
