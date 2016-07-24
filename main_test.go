package main

import (
	//	"net/http/httptest"
	//	"strings"
	//"reflect"
	"os"
	"testing"
	//	"time"
)

func TestHistory(t *testing.T) {
	var testPayload string = "Hello World!"

	var history = new(ForecastHistory)
	res := history.Store("Hello Brave New World!")
	if res != nil {
		t.Fatal("Unable to store payload!")
	}

	err := history.Load(&testPayload)
	if err != nil {
		t.Fatal("Unable to load stored payload!")
	}

	if testPayload != "Hello Brave New World!" {
		t.Error("Unexpected value after restoring hisotry!")
	}
	if _, err := os.Stat(history.Path()); os.IsNotExist(err) {
		t.Fatal("Unable to find storage file!", err)
	}
}

func TestYrno(t *testing.T) {
	var weather = new(Yrno)
	//var updates []Update
	var location string = "Germany/Berlin/Berlin"

	updates, err := weather.Fetch(location)
	if err != nil {
		t.Fatal("Unable to fetch forecast for %s", location)
	}
	if updates == nil || len(updates) == 0 {
		t.Fatal("No forecast data for %s", location)
	}
	t.Log(updates)
}

// statusHandler is an http.Handler that writes an empty response using itself
// as the response status code.
/*
type statusHandler int

func (h *statusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(int(*h))
}

func TestParseXml(t *testing.T) {
	r, err := os.Open("./data/test/data.xml")
	if err != nil {
		t.Fatal(err)
	}
	results := parseXml(r)

		if reflect.TypeOf(results).Kind() != reflect.slice {
			log.Print("Not a slice")
		}
	fmt.Println(reflect.TypeOf(results))
	fmt.Println(reflect.TypeOf(results).Kind())
	fmt.Println(reflect.TypeOf(results[0]))
}

func TestIntegration(t *testing.T) {
	status := statusHandler(http.StatusNotFound)
	ts := httptest.NewServer(&status)
	defer ts.Close()

	// Replace the pollSleep with a closure that we can block and unblock.
	sleep := make(chan bool)
	pollSleep = func(time.Duration) {
		sleep <- true
		sleep <- true
	}

	// Replace pollDone with a closure that will tell us when the poller is
	// exiting.
	done := make(chan bool)
	pollDone = func() { done <- true }

	// Put things as they were when the test finishes.
	defer func() {
		pollSleep = time.Sleep
		pollDone = func() {}
	}()

	s := NewServer("1.x", ts.URL, 1*time.Millisecond)

	<-sleep // Wait for poll loop to start sleeping.

	// Make first request to the server.
	r, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if b := w.Body.String(); !strings.Contains(b, "No.") {
		t.Fatalf("body = %s, want no", b)
	}

	status = http.StatusOK

	<-sleep // Permit poll loop to stop sleeping.
	<-done  // Wait for poller to see the "OK" status and exit.

	// Make second request to the server.
	w = httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if b := w.Body.String(); !strings.Contains(b, "YES!") {
		t.Fatalf("body = %q, want yes", b)
	}
}
*/
