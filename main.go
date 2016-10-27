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
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
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
type WeatherForecasts struct {
	Date      time.Time         `json:"date"`
	Location  string            `json:"location"`
	Forecasts []WeatherForecast `json:"forecasts"`
}

// Weather Forecast for a particular day
type WeatherForecast struct {
	Date    time.Time `json:"date"`
	Periods []Period  `json:"periods"`
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

	// init helper modules for weather parsing and persistance
	weather = new(Yrno)
	history = new(ForecastHistory)

	// location of static assets including index page
	docroot = "./docroot/"
)

func init() {
	gob.Register(forecasts)

	// check if persistance file exists and create empty one if not and
	// try persist empty struct for the first time
	if _, err := os.Stat(history.Path()); os.IsNotExist(err) {
		err := history.Store(&forecasts)
		if err != nil {
			log.Fatalf("Unable to create persitance file on init - %s", err)
		}
	}
}

func main() {
	// parse CLI arguments
	flag.Parse()

	// load persisted forecasts to memory
	err := history.Load(&forecasts)
	if err != nil {
		log.Fatalf("Unable to load persisted forecasts to memory  - %s", err)
	}

	// first load current forecast
	pollUpdates(*location)

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

	// register router based on defined routing schema
	router := NewRouter()

	// register static assets handler
	router.PathPrefix("/").Handler(http.FileServer(http.Dir(docroot)))

	log.Printf("Starting HTTP server - http://localhost:%s/", *httpPort)

	log.Fatal(http.ListenAndServe(*httpPort, router))
}

func APIWeather(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	date := vars["date"]

	// parser user supplied date
	queryDate := getQueryDate(date)
	log.Printf("Querying for date %s", queryDate)

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
func filterForecast(queryTime time.Time) WeatherForecasts {
	var measurements []Measurement
	var periods []Period

	// create top level struct
	var result = WeatherForecasts{
		Date:      queryTime.Truncate(24 * time.Hour),
		Forecasts: []WeatherForecast{},
	}

	for date, day := range forecasts {
		// display forecast for specified date and future
		if date.Truncate(24 * time.Hour).After(queryTime.Truncate(24 * time.Hour)) {
			periods = periods[:0]

			// transform all measurements
			for period, forecast := range day.Forecasts {
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
				periods = append(periods, period)
			}

			// finally append whole forecast for a day
			result.Forecasts = append(result.Forecasts, WeatherForecast{
				Date:    date.Truncate(24 * time.Hour),
				Periods: periods,
			})
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

	log.Println("Ticker: polling latest forecasts...")

	// merge fetched updates
	weather.Merge(forecasts, updates)

	// persist fetched data periodically too
	err = history.Store(&forecasts)
	if err != nil {
		log.Fatalf("Unable to persist forecasts after merging updates  - %s", err)
	}
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
