/*
--------------------------------------------------------------------------------
Web server that fetches data for specific location using yr.no XML format and
visualize changes in forecasted temperatures in time.
--------------------------------------------------------------------------------
TODO: - how to visualize (elm? react? vanilla?) using D3.js
      - ability to filter/query?
      - periodic updates (or handle update based on data in XML - nextupdate elem?
--------------------------------------------------------------------------------
*/
package main

import (
	"encoding/gob"
	"expvar"
	"flag"
	"fmt"
	"html/template"
	"io"
	"launchpad.net/xmlpath"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// Command-line flags.
var (
	httpAddr   = flag.String("http", ":8080", "Listen address")
	pollPeriod = flag.Duration("poll", 10*60*time.Second, "Poll period")
	location   = flag.String("location", "Germany/Berlin/Berlin", "Location")
)

const baseURL = "http://www.yr.no/sted/%s/varsel.xml"

func main() {
	flag.Parse()
	xmlURL := fmt.Sprintf(baseURL, *location)
	http.Handle("/", NewServer(*location, xmlURL, *pollPeriod))
	log.Fatal(http.ListenAndServe(*httpAddr, nil))

	// setup periodic ticker
	ticker := time.NewTicker(time.Millisecond * 500)
	go func() {
		for t := range ticker.C {
			log.Print("Tick at", t)
		}
	}()
}

// Exported variables for monitoring the server.
// These are exported via HTTP as a JSON object at /debug/vars.
var (
	hitCount       = expvar.NewInt("hitCount")
	pollCount      = expvar.NewInt("pollCount")
	pollError      = expvar.NewString("pollError")
	pollErrorCount = expvar.NewInt("pollErrorCount")
)

// It serves the user interface (it's an http.Handler)
// and polls the remote website for changes.
type Server struct {
	mu       sync.RWMutex // guards the fields below
	location string
	url      string
	period   time.Duration
}

// stores in memory as map with forecast day as a key
var forecasts map[time.Time]DayForecast = make(map[time.Time]DayForecast)

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

// NewServer returns an initialized server.
func NewServer(location, url string, period time.Duration) *Server {
	s := &Server{location: location, url: url, period: period}
	go s.fetch()
	return s
}

// Hooks that may be overridden for integration tests.
var (
	pollSleep = time.Sleep
	pollDone  = func() {}
)

func (s *Server) fetch() {
	log.Print("Fetching XML data...")
	resp, err := http.Get(s.url)
	if err != nil {
		// handle error
		log.Print(err)
		pollError.Set(err.Error())
		pollErrorCount.Add(1)
		pollSleep(s.period)
	}
	defer resp.Body.Close()

	// restore history from disk and load it to mermory
	loadHistory(&forecasts)

	// parse current forecast
	updates := parseXml(resp.Body)

	// check if we already have forecast for this day otherwise
	// create new one
	forecasts = applyUpdates(updates)

	// persist history to disk
	//storeHistory(forecasts)

	pollDone()
}

func applyUpdates(updates []Update) map[time.Time]DayForecast {
	var results map[time.Time]DayForecast = make(map[time.Time]DayForecast)

	for _, update := range updates {
		period := update.Period
		updated := update.Updated
		temp := update.Temperature

		// apply updates to in memory struct
		if current, ok := forecasts[update.Date]; ok {
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

func parseInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		log.Print(err)
	}
	return n
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

func parseXmlDate(input string) time.Time {
	t, err := time.Parse("2006-01-02T15:04:05", input)
	if err != nil {
		log.Print(err)
	}
	return t
}

func applyXPath(path *xmlpath.Path, node *xmlpath.Node) string {
	var result string = ""
	if value, ok := path.String(node); ok {
		result = value
	}
	return result
}

// Filter forecast data for particular day by offset
func filterForecast(offset float64) []Forecast {
	var result []Forecast
	return result
}

const storagePath = "./data/history.gob"

// Encode historical data via Gob to file
func storeHistory(object interface{}) error {
	log.Print("Storing historic data....")
	file, err := os.Create(storagePath)
	if err == nil {
		encoder := gob.NewEncoder(file)
		encoder.Encode(object)
	}
	file.Close()
	return err
}

// Decode historical data via Gob from file
func loadHistory(object interface{}) error {
	log.Print("Loading historic data....")
	file, err := os.Open(storagePath)
	if err == nil {
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(object)
	}
	file.Close()
	return err
}

// Parse QS for forecast date or use current date
func getQueryDate(qs string) time.Time {
	if len(qs) > 0 {
		query, err1 := time.Parse("01/02/2006", qs)
		if err1 != nil {
			log.Print(err1)
		}
		return query
	} else {
		return time.Now()
	}
}

// ServeHTTP implements the HTTP user interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	hitCount.Add(1)

	// by default user current time
	query := getQueryDate(r.URL.Query().Get("date"))
	duration := time.Now().Sub(query)
	offset := math.Floor(duration.Hours() / 24)
	log.Printf("Querying for date %s - offset=%f", query, offset)

	// handle qs and render filtered data
	s.mu.RLock()
	data := struct {
		Location     string
		Date         time.Time
		ForecastList []Forecast
	}{
		s.location,
		query,
		filterForecast(offset),
	}
	s.mu.RUnlock()
	err2 := tmpl.Execute(w, data)
	if err2 != nil {
		log.Print(err2)
	}
}

func maybeLogError(e error) {
	if e != nil {
		log.Print(e)
	}
}

//TODO: maybe load template from assets directory
// tmpl is the HTML template that drives the user interface.
var tmpl = template.Must(template.New("tmpl").Parse(`
<!DOCTYPE html>
<html>
<title>Weather for {{.Location}}?</title>
<body>
<h1>Weather for {{.Location}}</h1>
<p>Day {{.Date.Format "2006 Jan 02"}}</p>
</body>
</html>
`))
