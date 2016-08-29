package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

// may just setup a global logger, and change output during runtime...
// http://codereview.stackexchange.com/a/59733

var logf *os.File
var logger *log.Logger

func initLoggerWriter(w io.Writer) {
	logger = log.New(w, "", log.Lshortfile|log.LstdFlags)
	logger.Println("initializing logger")
}

func initLogger() {
	if conf.out.logFile != "" && conf.out.printDebug {
		logf, err := os.OpenFile(conf.out.logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Printf("error opening log file: %s, %v", conf.out.logFile, err)
			os.Exit(1)
		}

		initLoggerWriter(io.MultiWriter(logf, os.Stdout))
		return
	}

	if conf.out.logFile != "" {
		logf, err := os.OpenFile(conf.out.logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Printf("error opening log file: %s, %v", conf.out.logFile, err)
			os.Exit(1)
		}

		initLoggerWriter(logf)
		return
	}

	if conf.out.printDebug {
		initLoggerWriter(os.Stdout)
		return
	}

	initLoggerWriter(ioutil.Discard)
}

func closeLogger() {
	if logf == nil {
		return
	}

	logf.Close()
}

type Color struct {
	Name string
	ID   int
}

var colors = []*Color{
	{"c", 0}, {"bold", 1}, {"black", 30}, {"red", 31}, {"green", 32}, {"yellow", 33},
	{"blue", 34}, {"magenta", 35}, {"cyan", 36}, {"white", 37}, {"gray", 90},
	{"lightred", 91}, {"lightgreen", 92}, {"lightyellow", 93}, {"lightblue", 94},
	{"lightmagenta", 95}, {"lightcyan", 96}, {"lightgray", 97},
}

func FmtColor(format *string, shouldStrip bool) {
	for _, clr := range colors {
		if shouldStrip {
			*format = strings.Replace(*format, "{"+clr.Name+"}", "", -1)
		} else {
			*format = strings.Replace(*format, "{"+clr.Name+"}", "\x1b["+strconv.Itoa(clr.ID)+"m", -1)
		}
	}

	if !shouldStrip {
		*format = *format + "\x1b[0;m"
	}

	return
}

type Output struct{}

// Printf interprets []*Color{} escape codes and prints them to stdout
func (o Output) Printf(format string, a ...interface{}) (n int, err error) {
	FmtColor(&format, conf.out.noColors)

	return fmt.Printf(format, a...)
}

// Println interprets []*Color{} escape codes and prints them to stdout
func (o Output) Println(a ...interface{}) (n int, err error) {
	return fmt.Println(a...)
}
