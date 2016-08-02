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
	"flag"
	"fmt"
	"github.com/kataras/iris"
	"log"
	"math"
	"time"
)

import _ "net/http/pprof"

var (
	// CLI flags
	httpPort = flag.String("port", ":8080", "Listen port")
	location = flag.String("location", "Germany/Berlin/Berlin", "Location")
	gzipMode = flag.Bool("gzip", true, "GZip Compression")

	// yr.no polite polling policy
	pollPeriod = flag.Duration("poll", 10*60*time.Second, "Poll period")

	// stores in memory as map with forecast day as a key
	forecasts = make(map[time.Time]DayForecast)

	weather = new(Yrno)
)

func main() {
	flag.Parse()

	// first load current forecast
	pollUpdates(*location)

	// Ops API
	iris.Get("/ping", func(ctx *iris.Context) {
		ctx.Text(iris.StatusOK, "pong")
	})

	// REST API
	iris.Get("/api/weather", func(ctx *iris.Context) {
		fmt.Println(ctx)
		date := ctx.URLParam("date")

		// parser user supplied date
		query := getQueryDate(date)
		duration := time.Now().Sub(query)
		offset := math.Floor(duration.Hours() / 24)
		log.Printf("Querying for date %s - offset=%f", query, offset)

		// serve dummy JSON response
		ctx.JSON(iris.StatusOK, iris.Map{
			"date": query.Format("01/02/2006"),
		})
	})

	// Static page & assets
	iris.StaticServe("docroot/assets/")
	iris.Get("docroot/style.css", func(ctx *iris.Context) {
		ctx.ServeFile("docroot/style.css", *gzipMode)
	})
	iris.Get("/", func(ctx *iris.Context) {
		ctx.ServeFile("docroot/index.html", *gzipMode)
	})

	//TODO: implement exponential backoff in ase API is down
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

	iris.Listen(*httpPort)
}

// Filter forecast data for particular day by offset
/*
func filterForecast(offset float64) []Forecast {
	var result []Forecast
	return result
}
*/

// Parse QS for forecast date or use current date
func getQueryDate(qs string) time.Time {
	log.Println(qs)
	if len(qs) > 0 {
		query, err1 := time.Parse("01/02/2006", qs)
		if err1 != nil {
			log.Print(err1)
		}
		return query
	}
	return time.Now()
}

func pollUpdates(location string) {
	// first fetch updates
	updates, err := weather.Fetch(location)
	if err != nil {
		log.Fatalf("Unable to fetch forecast for %s", location)
	}

	// merge fetched updates
	weather.Merge(forecasts, updates)
}
