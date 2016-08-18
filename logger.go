package main

import (
	"fmt"
	"io"
	"log"
	"os"
)

var logf *os.File
var logger *log.Logger

func initLogger(w io.Writer) {
	logger = log.New(w, "", log.Lshortfile|log.LstdFlags)
}

func initLoggerToFile(fn string) {
	logf, err := os.OpenFile(fn, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("error opening log file: %s, %v", fn, err)
		os.Exit(1)
	}
	initLogger(logf)
}

func closeLogger() {
	logf.Close()
}
