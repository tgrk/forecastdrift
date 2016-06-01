package main

import (
	"fmt"
	"net/http"
	"os"
	//	"net/http/httptest"
	//	"strings"
	"reflect"
	"testing"
	//	"time"
)

// statusHandler is an http.Handler that writes an empty response using itself
// as the response status code.
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

	/*
		if reflect.TypeOf(results).Kind() != reflect.slice {
			log.Print("Not a slice")
		}
	*/
	fmt.Println(reflect.TypeOf(results))
	fmt.Println(reflect.TypeOf(results).Kind())
	fmt.Println(reflect.TypeOf(results[0]))
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
