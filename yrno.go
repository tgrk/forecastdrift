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

type Yrno struct {
}

// Store weather information by date with history of forecasted temperatures
type DayForecast struct {
	Date      time.Time        // same as key of in memory map
	Forecasts map[int]Forecast // contains day periods 0 - 3 by six hours
}
type Forecast struct {
	History map[time.Time]int // contains forecasted temperature in time
}
type Update struct {
	Updated     time.Time
	Date        time.Time
	Period      int
	Temperature int
}

// Fetches latest weather forecast for given location
func (weather Yrno) Fetch(location string) ([]Update, error) {
	var xmlURL string = fmt.Sprintf(baseURL, location)
	log.Printf("Fetching XML data from %s...\n", xmlURL)

	resp, err := http.Get(xmlURL)
	if err != nil {
		// handle error
		log.Print(err)
		return nil, err
	}
	defer resp.Body.Close()

	// parse current forecast
	updates := parseXml(resp.Body)

	return updates, nil
}

// Merges updated forecast into an existing forecasts
func (weather Yrno) MergeUpdates(existing map[time.Time]DayForecast, updates []Update) map[time.Time]DayForecast {
	var results map[time.Time]DayForecast = make(map[time.Time]DayForecast)

	for _, update := range updates {
		period := update.Period
		updated := update.Updated
		temp := update.Temperature

		// apply updates to in memory struct
		if current, ok := existing[update.Date]; ok {
			// store update forecast
			current.Forecasts[period].History[updated] = temp
			results[update.Date] = current
		} else {
			// inititialize struct
			current := DayForecast{
				update.Date,
				make(map[int]Forecast),
			}
			current.Forecasts[update.Period] = Forecast{
				make(map[time.Time]int),
			}
			// store new forecasts
			current.Forecasts[period].History[updated] = temp
			results[update.Date] = current
		}
	}

	return results
}

// Extract current forecasts from XML
func parseXml(r io.Reader) []Update {
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
	var updated time.Time = parseXmlDate(applyXPath(updatedPath, root))

	var results []Update
	iter := timePath.Iter(root)
	for iter.Next() {
		// forecasted day
		update := Update{
			updated,
			parseXmlDate(applyXPath(fromPath, iter.Node())).Local(),
			parseInt(applyXPath(periodPath, iter.Node())),
			parseInt(applyXPath(tempPath, iter.Node())),
		}
		results = append(results, update)
	}

	return results
}

func applyXPath(path *xmlpath.Path, node *xmlpath.Node) string {
	var result string = ""
	if value, ok := path.String(node); ok {
		result = value
	}
	return result
}

func parseXmlDate(input string) time.Time {
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
