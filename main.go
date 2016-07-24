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

var (
	// CLI flags
	httpPort   = flag.String("port", ":8080", "Listen port")
	pollPeriod = flag.Duration("poll", 10*60*time.Second, "Poll period")
	location   = flag.String("location", "Germany/Berlin/Berlin", "Location")

	// stores in memory as map with forecast day as a key
	forecasts map[time.Time]DayForecast = make(map[time.Time]DayForecast)
)

func main() {
	flag.Parse()

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
	iris.Get("/", func(ctx *iris.Context) {
		// gzip compression
		ctx.ServeFile("docroot/index.html", false)
	})

	iris.Listen(*httpPort)
}

// Filter forecast data for particular day by offset
func filterForecast(offset float64) []Forecast {
	var result []Forecast
	return result
}

// Parse QS for forecast date or use current date
func getQueryDate(qs string) time.Time {
	log.Println(qs)
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

func maybeLogError(e error) {
	if e != nil {
		log.Print(e)
	}
}
