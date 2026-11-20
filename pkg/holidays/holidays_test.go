package holidays

import (
	"encoding/xml"
	"io/fs"
	"strings"
	"testing"
	"time"

	geneosxml "github.com/itrs-group/cordial/pkg/geneos"
)

func TestHolidayIncludeOverrides(t *testing.T) {
	calendars := make(map[string]map[string]bool)
	regions := make(map[string]string)
	groupNames := make(map[string]bool)
	regionNames := []string{"as", "au", "eu", "in", "na", "sa"}
	files, err := fs.Glob(Files, "files/holidays.layer0.*.xml")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != len(regionNames) {
		t.Fatalf("expected %d populated regional includes, got %v", len(regionNames), files)
	}
	for index, region := range regionNames {
		if want := "files/holidays.layer0." + region + ".xml"; files[index] != want {
			t.Fatalf("regional include %q, want %q", files[index], want)
		}
		data, err := Files.ReadFile("files/holidays.layer0." + region + ".xml")
		if err != nil {
			t.Fatal(err)
		}
		var setup geneosxml.Gateway
		if err := xml.Unmarshal(data, &setup); err != nil {
			t.Fatal(err)
		}
		if setup.ActiveTimes == nil || setup.ActiveTimes.GatewayControlledProbeActiveTimes != nil {
			t.Fatalf("%s: expected reusable active times without a Gateway-wide scheduling setting", region)
		}
		groups := setup.ActiveTimes.Groups
		if len(setup.ActiveTimes.ActiveTimes) != 0 || len(groups) != 1 || groups[0].Name != "Layer 0 - Holidays" ||
			len(groups[0].ActiveTimes) != 0 || len(groups[0].Groups) == 0 {
			t.Fatalf("%s: expected one Layer 0 - Holidays parent holding country groups", region)
		}
		var lastCountry string
		for _, group := range groups[0].Groups {
			prefix := "Layer 0 - " + strings.ToUpper(region) + " - "
			country := strings.TrimPrefix(group.Name, prefix)
			if !strings.HasPrefix(group.Name, prefix) || country == "" ||
				len(group.ActiveTimes) == 0 || len(group.Groups) != 0 || strings.TrimSpace(group.Comment) == "" {
				t.Fatalf("%s: expected a documented country group containing calendars, got %q", region, group.Name)
			}
			if country <= lastCountry {
				t.Errorf("%s: country %q follows %q out of alphabetical order", region, country, lastCountry)
			}
			lastCountry = country
			if groupNames[group.Name] {
				t.Fatalf("duplicate group name %q across regional includes", group.Name)
			}
			groupNames[group.Name] = true
			lastKind, lastCalendar := -1, ""
			for _, calendar := range group.ActiveTimes {
				var kind int
				switch {
				case strings.HasSuffix(calendar.Name, " - Public"):
					kind = 0
				case strings.HasSuffix(calendar.Name, " - Bank"):
					kind = 1
				case strings.HasSuffix(calendar.Name, " - Collective Leave"):
					kind = 2
				case strings.HasSuffix(calendar.Name, " - Exchange"):
					kind = 3
				default:
					t.Fatalf("%s: unknown calendar kind %q", group.Name, calendar.Name)
				}
				if kind < lastKind || (kind == lastKind && calendar.Name <= lastCalendar) {
					t.Errorf("%s: calendar %q follows %q out of country/kind order", group.Name, calendar.Name, lastCalendar)
				}
				lastKind, lastCalendar = kind, calendar.Name
				if _, exists := calendars[calendar.Name]; exists {
					t.Fatalf("duplicate calendar name %q", calendar.Name)
				}
				dates := make(map[string]bool)
				calendars[calendar.Name] = dates
				regions[calendar.Name] = region
				var lastAnnual, lastDated time.Time
				for _, child := range calendar.Children {
					if child.XMLName.Local != "scheduledPeriod" {
						continue
					}
					var period struct {
						Date      string `xml:"date"`
						StartTime string `xml:"startTime"`
						EndTime   string `xml:"endTime"`
						Override  bool   `xml:"override"`
						Invert    bool   `xml:"invert"`
					}
					if err := xml.Unmarshal([]byte("<scheduledPeriod>"+child.Content+"</scheduledPeriod>"), &period); err != nil {
						t.Fatal(err)
					}
					if !period.Override || period.Invert || period.StartTime != "" || period.EndTime != "" {
						t.Errorf("%s: %s is not a full-day closure override", calendar.Name, period.Date)
					}
					annual := strings.Count(period.Date, "/") == 1
					date, previous := period.Date, &lastDated
					if annual {
						date, previous = period.Date+"/2000", &lastAnnual
						if !lastDated.IsZero() {
							t.Errorf("%s: recurring date %s follows a year-specific date", calendar.Name, period.Date)
						}
					}
					if parsed, err := time.Parse("02/01/2006", date); err != nil {
						t.Errorf("%s: %v", calendar.Name, err)
					} else {
						if parsed.Before(*previous) {
							t.Errorf("%s: date %s is out of chronological order", calendar.Name, period.Date)
						}
						*previous = parsed
					}
					if dates[period.Date] {
						t.Errorf("%s: duplicate date %s", calendar.Name, period.Date)
					}
					dates[period.Date] = true
				}
			}
		}
	}
	for _, check := range []struct {
		calendar, region, date string
		want                   bool
	}{
		{"Holidays - FR Metropolitan - Public", "eu", "25/12", true},
		{"Holidays - FR Metropolitan - Public", "eu", "25/12/2027", false},
		{"Holidays - GB England and Wales - Public", "eu", "27/12/2027", true},
		{"Holidays - GB England and Wales - Public", "eu", "25/12", false},
		{"Holidays - AU NSW - Public", "au", "26/04/2027", true},
		{"Holidays - AU ASX Cash - Exchange", "au", "26/04/2027", false},
		{"Holidays - TW TWSE - Exchange", "as", "02/02/2027", true},
		{"Holidays - PH - Public", "as", "20/03/2026", true},
		{"Holidays - PH - Public", "as", "25/02/2026", false},
		{"Holidays - MY Kuala Lumpur - Public", "as", "01/02", true},
		{"Holidays - MY Kuala Lumpur - Public", "as", "03/02/2026", true},
		{"Holidays - ID - Public", "as", "10/03/2027", true},
		{"Holidays - ID - Public", "as", "24/03/2026", false},
		{"Holidays - ID - Collective Leave", "as", "24/03/2026", true},
		{"Holidays - US Cboe Options - Exchange", "na", "03/07/2026", false},
		{"Holidays - US Cboe Options - Exchange", "na", "03/04/2026", true},
		{"Holidays - MX BMV - Exchange", "na", "02/11/2026", true},
		{"Holidays - BR B3 Listed Equities - Exchange", "sa", "20/11/2026", true},
		{"Holidays - IN Chennai - Bank", "in", "23/04/2026", true},
		{"Holidays - IN Mumbai - Bank", "in", "23/04/2026", false},
		{"Holidays - IN Mumbai - Bank", "in", "03/03/2026", true},
		{"Holidays - IN New Delhi - Bank", "in", "03/03/2026", false},
		{"Holidays - IN New Delhi - Bank", "in", "04/03/2026", true},
		{"Holidays - IN Chennai - Bank", "in", "01/04/2026", true},
		{"Holidays - IN Chennai - Bank", "in", "26/01", false},
		{"Holidays - IN Chennai - Bank", "in", "26/01/2027", false},
		{"Holidays - IN NSE Cash - Exchange", "in", "01/04/2026", false},
		{"Holidays - IN NSE Cash - Exchange", "in", "08/11/2026", false},
		{"Holidays - IN NSE Cash - Exchange", "in", "10/11/2026", true},
	} {
		if got := regions[check.calendar]; got != check.region {
			t.Errorf("%s: region %q, want %q", check.calendar, got, check.region)
		}
		dates := calendars[check.calendar]
		if dates[check.date] != check.want {
			t.Errorf("%s: date %s present = %t, want %t", check.calendar, check.date, dates[check.date], check.want)
		}
	}
}
