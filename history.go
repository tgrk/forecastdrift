package main

import (
	"encoding/gob"
	"log"
	"os"
)

const storagePath = "./data/history.gob"

// ForecastHistory is responsible for GOB persistance
type ForecastHistory struct {
}

// Store historical data via Gob to file
func (history ForecastHistory) Store(object interface{}) error {
	log.Print("Storing historic data....")
	file, err := os.Create(storagePath)
	defer file.Close()
	if err == nil {
		encoder := gob.NewEncoder(file)
		encoder.Encode(object)
		return nil
	}
	return err
}

// Load historical data via Gob from file
func (history ForecastHistory) Load(object interface{}) error {
	log.Print("Loading historic data....")
	file, err := os.Open(storagePath)
	defer file.Close()
	if err == nil {
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(object)
	}
	return err
}

// Path to GOB persistance file
func (history ForecastHistory) Path() string {
	return storagePath
}
