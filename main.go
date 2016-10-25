/*
--------------------------------------------------------------------------------
Web server that fetches data for specific location using yr.no XML format and
visualize changes in forecasted temperatures in time.
--------------------------------------------------------------------------------
TODO: - create simple SPA website using D3.js for data visualization
      - periodic updates (or handle update based on data in XML - nextupdate elem?
--------------------------------------------------------------------------------
*/
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/gorilla/mux"
)

// HTTP API routes
type APIRoute struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}
type APIRoutes []APIRoute

// Weather Forecast to be served as JSON
type WeatherForecast struct {
	Date     time.Time `json:"date"`
	Location string    `json:"location"`
	Periods  []Period  `json:"periods"`
}

// Day period with history of temperature measurements
type Period struct {
	Period       int           `json:"period"`
	Measurements []Measurement `json:"measurements"`
}

// Temperature measurement in time
type Measurement struct {
	Updated     time.Time `json:"updated"`
	Temperature int       `json:"temperature"`
}

var (
	// CLI flags
	httpPort = flag.String("port", ":8080", "Listen port")
	location = flag.String("location", "Germany/Berlin/Berlin", "Location")

	// yr.no polite polling policy
	pollPeriod = flag.Duration("poll", 10*60*time.Second, "Poll period")

	// stores in memory as map with forecast day as a key
	forecasts = make(map[time.Time]DayForecast)

	// access YR.NO weather forecast information
	weather = new(Yrno)

	// location of static assets including index page
	docroot = "./docroot/"
)

func main() {
	// parse CLI arguments
	flag.Parse()

	// first load current forecast
	pollUpdates(*location)

	// register router based on defined routing schema
	router := NewRouter()

	// register static assets handler
	router.PathPrefix("/").Handler(http.FileServer(http.Dir(docroot)))

	log.Fatal(http.ListenAndServe(*httpPort, router))

	// periodically fetch weather forecast updates
	log.Printf("Ticker: polling every %s", *pollPeriod)
	ticker := time.NewTicker(*pollPeriod)
	quit := make(chan bool, 1)
	go func() {
		for {
			select {
			case <-ticker.C:
				pollUpdates(*location)
			case <-quit:
				log.Println("Ticker: stop")
				ticker.Stop()
				return
			}
		}
	}()
}

func APIWeather(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	date := vars["date"]

	// parser user supplied date
	queryDate := getQueryDate(date)
	duration := time.Now().Sub(queryDate)
	offset := math.Floor(duration.Hours() / 24)
	log.Printf("Querying for date %s - offset=%f", queryDate, offset)

	// server JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// prepare data based on user input
	var filtered = filterForecast(queryDate)

	//TODO: get location from QS too
	filtered.Location = *location

	// serve JSON response
	if err := json.NewEncoder(w).Encode(filtered); err != nil {
		fmt.Fprintf(w, "Unable to serve weather data - %s", err)
	}
}

// Filter forecast data for particular day by offset
func filterForecast(queryTime time.Time) WeatherForecast {
	var periods []Period
	var result = WeatherForecast{
		Date:    queryTime.Truncate(24 * time.Hour),
		Periods: periods,
	}

	var measurements []Measurement
	for t, weather := range forecasts {
		// create response structure only with data that we need based on query date
		if queryTime.Truncate(24 * time.Hour).Equal(t.Truncate(24 * time.Hour)) {
			for period, forecast := range weather.Forecasts {
				// transform all measurements
				measurements = measurements[:0]
				period := Period{
					Period:       period,
					Measurements: measurements[:0],
				}
				for updated, temperature := range forecast.History {
					measurement := Measurement{
						Updated:     updated,
						Temperature: temperature,
					}
					period.Measurements = append(period.Measurements, measurement)
				}
				result.Periods = append(result.Periods, period)
			}
		}
	}
	return result
}

// periodically pull updated forecast and merge it with existing one to track changes
func pollUpdates(location string) {
	// first fetch updates
	updates, err := weather.Fetch(location)
	if err != nil {
		log.Fatalf("Unable to fetch forecast for %s - %s", location, err)
	}

	log.Println("Ticker: pulling latest forecasts...")

	// merge fetched updates
	weather.Merge(forecasts, updates)
}

// Parse QS for forecast date or use current date
func getQueryDate(qs string) time.Time {
	if len(qs) > 0 {
		query, err1 := time.Parse("01/02/2006", qs)
		if err1 != nil {
			log.Print(err1)
		}
		return query
	}
	return time.Now()
}

func NewRouter() *mux.Router {
	// define URI routes
	var routes = APIRoutes{
		APIRoute{
			"API",
			"GET",
			"/api/weather",
			APIWeather,
		},
	}

	// bind router and access logger
	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {
		var handler http.Handler
		handler = route.HandlerFunc
		handler = Logger(handler, route.Name)

		router.Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}
	return router
}

func Logger(inner http.Handler, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		inner.ServeHTTP(w, r)

		log.Printf(
			"%s\t%s\t%s\t%s",
			r.Method,
			r.RequestURI,
			name,
			time.Since(start),
		)
	})
}
