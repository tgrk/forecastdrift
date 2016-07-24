package main

import (
	"encoding/gob"
	"log"
	"os"
)

const storagePath = "./data/history.gob"

type ForecastHistory struct {
}

// Encode historical data via Gob to file
func (history ForecastHistory) Store(object interface{}) error {
	log.Print("Storing historic data....")
	file, err := os.Create(storagePath)
	if err == nil {
		encoder := gob.NewEncoder(file)
		encoder.Encode(object)
		return nil
	}
	file.Close()
	return err
}

// Decode historical data via Gob from file
func (history ForecastHistory) Load(object interface{}) error {
	log.Print("Loading historic data....")
	file, err := os.Open(storagePath)
	if err == nil {
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(object)
	}
	file.Close()
	return err
}

func (history ForecastHistory) Path() string {
	return storagePath
}
