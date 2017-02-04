// Author: Liam Stanley <me@liamstanley.io>
// Docs: https://marill.liam.sh/
// Repo: https://github.com/lrstanley/marill

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/lrstanley/marill/utils"
)

// just setup a global logger, and change output during runtime...

var logf *os.File
var logger *log.Logger

func initLoggerWriter(w io.Writer) {
	logger = log.New(w, "", log.Lshortfile|log.LstdFlags)
	logger.Println("initializing logger")
}

func initLogger() {
	var err error
	if conf.out.DebugLog != "" && conf.out.PrintDebug {
		logf, err = os.OpenFile(conf.out.DebugLog, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Printf("error opening log file: %s, %v", conf.out.DebugLog, err)
			os.Exit(1)
		}

		initLoggerWriter(io.MultiWriter(logf, os.Stdout))
		return
	}

	if conf.out.DebugLog != "" {
		logf, err = os.OpenFile(conf.out.DebugLog, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Printf("error opening log file: %s, %v", conf.out.DebugLog, err)
			os.Exit(1)
		}

		initLoggerWriter(logf)
		return
	}

	if conf.out.PrintDebug {
		initLoggerWriter(os.Stdout)
		return
	}

	initLoggerWriter(ioutil.Discard)
}

func closeLogger() {
	if logf != nil {
		logf.Close()
	}
}

// Color represents an ASCII color sequence for use with prettified output
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

// StripColor strips all color {patterns} from input
func StripColor(format *string) {
	for _, clr := range colors {
		*format = strings.Replace(*format, "{"+clr.Name+"}", "", -1)
	}
}

var reNonASCII = regexp.MustCompile(`\x1b.*?m`)

// StripColorBytes strips all color {patterns} from input (however, in bytes)
func StripColorBytes(format *[]byte) {
	*format = reNonASCII.ReplaceAll(*format, []byte("")) //re-apply back to the original format
}

// FmtColor adds (or removes) color output depending on user input
func FmtColor(format *string, shouldStrip bool) {
	if shouldStrip {
		StripColor(format)

		return
	}

	for _, clr := range colors {
		*format = strings.Replace(*format, "{"+clr.Name+"}", "\x1b["+strconv.Itoa(clr.ID)+"m", -1)
	}

	*format = *format + "\x1b[0;m"
}

// Output is the bare out struct for which stdout messages are passed to
type Output struct {
	log    *log.Logger
	buffer []string
	logf   *os.File
}

var out = Output{}

func initOutWriter(w ...io.Writer) {
	out.log = log.New(io.MultiWriter(w...), "", 0)
}

func initOut(w io.Writer) {
	var err error
	if conf.out.Log != "" && !conf.out.IgnoreStd {
		out.logf, err = os.OpenFile(conf.out.Log, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Printf("error opening log file: %s, %v", conf.out.Log, err)
			os.Exit(1)
		}

		initOutWriter(w, utils.NewFuncWriter(StripColorBytes, out.logf))
		return
	} else if conf.out.Log != "" {
		out.logf, err = os.OpenFile(conf.out.Log, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Printf("error opening log file: %s, %v", conf.out.Log, err)
			os.Exit(1)
		}

		initOutWriter(utils.NewFuncWriter(StripColorBytes, out.logf))
		return
	}

	if !conf.out.IgnoreStd {
		initOutWriter(w)
		return
	}

	initOutWriter(ioutil.Discard)
}

func closeOut() {
	if out.logf != nil {
		out.logf.Close()
	}
}

func (o Output) Write(b []byte) (int, error) {
	str := fmt.Sprintf("%s", b)
	o.AddLog(str)

	FmtColor(&str, conf.out.NoColors)
	o.log.Print(str)

	return len(b), nil
}

// AddLog adds log line to log stack
func (o *Output) AddLog(line string) {
	o.buffer = append(o.buffer, line)
}

// Printf interprets []*Color{} escape codes and prints them to stdout
func (o *Output) Printf(format string, a ...interface{}) {
	if conf.out.IgnoreStd {
		return
	}

	FmtColor(&format, conf.out.NoColors)

	out.log.Printf(format, a...)
	o.AddLog(fmt.Sprintf(format, a...))
}

// Println interprets []*Color{} escape codes and prints them to stdout
func (o *Output) Println(a ...interface{}) {
	if conf.out.IgnoreStd {
		return
	}

	str := fmt.Sprint(a...)
	FmtColor(&str, conf.out.NoColors)

	out.log.Print(str)
	o.AddLog(str)
}

// Fatalf interprets []*Color{} escape codes and prints them to stdout/logger, and exits
func (o *Output) Fatalf(format string, a ...interface{}) {
	// print to regular stdout
	if !conf.out.IgnoreStd {
		str := fmt.Sprintf(fmt.Sprintf("{bold}{red}error:{c} %s", format), a...)
		FmtColor(&str, conf.out.NoColors)
		out.log.Print(str)
		o.AddLog(str)
	}

	// strip color from format
	StripColor(&format)
	logger.Fatalf("error: "+format, a...)
}

// Fatal interprets []*Color{} escape codes and prints them to stdout, and exits
func (o *Output) Fatal(a ...interface{}) {
	// print to regular stdout
	if !conf.out.IgnoreStd {
		str := fmt.Sprintf("{bold}{red}error:{c} %s", fmt.Sprintln(a...))
		FmtColor(&str, conf.out.NoColors)
		out.log.Print(str)
		o.AddLog(str)
	}

	str := fmt.Sprintln(a...)

	logger.Fatal("error: " + str)
}
