package main

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestHistory(t *testing.T) {
	var testPayload = "Hello World!"

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
	var location = "Germany/Berlin/Berlin"

	updates, err := weather.Fetch(location)
	if err != nil {
		t.Fatal("Unable to fetch forecast for %s", location)
	}
	if updates == nil || len(updates) == 0 {
		t.Fatal("No forecast data for %s", location)
	}

	// merge fetched updates
	existing := make(map[time.Time]DayForecast)
	weather.Merge(existing, updates)
	if existing == nil || len(existing) == 0 {
		t.Fatal("Unable to merge changes!", err)
	}

	// merge one new update
	var newUpdates = []Update{
		Update{
			time.Now(),
			time.Now(),
			0,
			30,
		},
	}

	fmt.Println(newUpdates)
	//weather.Merge(existing, newUpdates)

	// TODO: merge an existing update

	fmt.Println(existing)
}

/*
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
