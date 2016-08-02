package main

import (
	"fmt"
	"io"
	"launchpad.net/xmlpath"
	"log"
	"net/http"
	"strconv"
	"time"
)

const baseURL = "http://www.yr.no/sted/%s/varsel.xml"

// Yrno parses and merges updates from yr.no weather forecast website
type Yrno struct {
}

// DayForecast contains weather forecast by date with history of temperatures
type DayForecast struct {
	Date      time.Time        // same as key of in memory map
	Forecasts map[int]forecast // contains day periods 0 - 3 by six hours
}

type forecast struct {
	History map[time.Time]int // contains forecasted temperature in time
}

// Update from yr.no after parsing XML data
type Update struct {
	Updated     time.Time
	Date        time.Time
	Period      int
	Temperature int
}

// Fetch latest weather forecast for given location
func (weather Yrno) Fetch(location string) ([]Update, error) {
	xmlURL := fmt.Sprintf(baseURL, location)
	log.Printf("Fetching XML data from %s...\n", xmlURL)

	resp, err := http.Get(xmlURL)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	defer resp.Body.Close()

	// parse current forecast
	updates := parseXML(resp.Body)

	return updates, nil
}

// Merge updated forecast into an existing forecasts
func (weather Yrno) Merge(existing map[time.Time]DayForecast, updates []Update) {
	for _, update := range updates {
		period := update.Period
		updated := update.Updated

		// apply updates to in memory struct
		if current, ok := existing[update.Date]; ok {
			// store update forecast with updated temperattures
			current.Forecasts[period].History[updated] = update.Temperature
		} else {
			// add new forecast dta
			current := DayForecast{
				update.Date,
				make(map[int]forecast),
			}
			current.Forecasts[update.Period] = forecast{
				make(map[time.Time]int),
			}
			current.Forecasts[period].History[updated] = update.Temperature
			existing[update.Date] = current
		}
	}
}

// Extract current forecasts from XML
func parseXML(r io.Reader) []Update {
	log.Print("Parsing XML file...")

	root, err := xmlpath.Parse(r)
	if err != nil {
		log.Fatal(err)
	}
	log.Print("XML parsed...")

	// define XPath queries
	updatedPath := xmlpath.MustCompile("/weatherdata/meta/lastupdate")
	timePath := xmlpath.MustCompile("/weatherdata/forecast/tabular/time")
	fromPath := xmlpath.MustCompile("./@from")
	periodPath := xmlpath.MustCompile("./@period")
	tempPath := xmlpath.MustCompile("./temperature/@value")

	// date of forecast update
	updated := parseXMLDate(applyXPath(updatedPath, root))

	var results []Update
	iter := timePath.Iter(root)
	for iter.Next() {
		// forecasted day
		update := Update{
			updated,
			parseXMLDate(applyXPath(fromPath, iter.Node())).Local(),
			parseInt(applyXPath(periodPath, iter.Node())),
			parseInt(applyXPath(tempPath, iter.Node())),
		}
		results = append(results, update)
	}

	return results
}

func applyXPath(path *xmlpath.Path, node *xmlpath.Node) string {
	var result string
	if value, ok := path.String(node); ok {
		result = value
	}
	return result
}

func parseXMLDate(input string) time.Time {
	t, err := time.Parse("2006-01-02T15:04:05", input)
	if err != nil {
		log.Print(err)
	}
	return t
}

func parseInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		log.Print(err)
	}
	return n
}
