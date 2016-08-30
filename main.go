package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/Liamraystanley/marill/domfinder"
	"github.com/Liamraystanley/marill/scraper"
	"github.com/urfave/cli"
)

// these /SHOULD/ be defined during the make process. not always however.
var version, commithash, compiledate = "", "", ""

type outputConfig struct {
	noColors   bool
	printDebug bool
	ignoreStd  bool
	logFile    string
}

type scanConfig struct {
	cores int
}

type appConfig struct {
	printUrls bool
}

type config struct {
	app  appConfig
	scan scanConfig
	out  outputConfig
}

var conf config
var out = Output{}

func statsLoop(done <-chan struct{}) {
	mem := &runtime.MemStats{}
	var numRoutines, numCPU int

	for {
		select {
		case <-done:
			return
		default:
			runtime.ReadMemStats(mem)
			numRoutines = runtime.NumGoroutine()
			numCPU = runtime.NumCPU()

			logger.Printf("allocated mem: %dM, sys: %dM, threads: %d, cores: %d", mem.Alloc/1024/1024, mem.Sys/1024/1024, numRoutines, numCPU)

			time.Sleep(2 * time.Second)
		}
	}
}

func numCores() {
	if conf.scan.cores == 0 {
		if runtime.NumCPU() == 1 {
			conf.scan.cores = 1
		} else {
			conf.scan.cores = runtime.NumCPU() / 2
		}
	} else if conf.scan.cores > runtime.NumCPU() {
		logger.Printf("warning: using %d cores, which is more than the amount of cores", conf.scan.cores)
		out.Printf("{yellow}warning: using %d cores, which is more than the amount of cores on the server!{c}\n", conf.scan.cores)

		// set it to the amount of cores on the server. go will do this regardless, so.
		conf.scan.cores = runtime.NumCPU()
		logger.Printf("limiting number of cores to %d", conf.scan.cores)
		out.Printf("limiting number of cores to %d\n", conf.scan.cores)
	}

	runtime.GOMAXPROCS(conf.scan.cores)
	logger.Printf("using %d cores (max %d)", conf.scan.cores, runtime.NumCPU())

	return
}

func printUrls() error {
	finder := &domfinder.Finder{Log: logger}
	if err := finder.GetWebservers(); err != nil {
		return fmt.Errorf("unable to get process list: %s", err)
	}

	if err := finder.GetDomains(); err != nil {
		return fmt.Errorf("unable to auto-fetch domain list: %s", err)
	}

	for _, domain := range finder.Domains {
		out.Printf("{blue}%-40s{c} {green}%s{c}\n", domain.URL, domain.IP)
	}

	return nil
}

func run() {
	logger.Println("checking for running webservers...")

	finder := &domfinder.Finder{Log: logger}
	if err := finder.GetWebservers(); err != nil {
		logger.Fatalf("unable to get process list: %s", err)
	}

	if out := ""; len(finder.Procs) > 0 {
		for _, proc := range finder.Procs {
			out += fmt.Sprintf("[%s:%s] ", proc.Name, proc.PID)
		}
		logger.Printf("found %d procs matching a webserver: %s", len(finder.Procs), out)
	}

	// start crawling for domains
	if err := finder.GetDomains(); err != nil {
		logger.Fatalf("unable to auto-fetch domain list: %s", err)
	}

	logger.Printf("found %d domains on webserver %s (exe: %s, pid: %s)", len(finder.Domains), finder.MainProc.Name, finder.MainProc.Exe, finder.MainProc.PID)

	tmplist := []*scraper.Domain{}
	for _, domain := range finder.Domains {
		tmplist = append(tmplist, &scraper.Domain{URL: domain.URL, IP: domain.IP})
	}
	crawler := &scraper.Crawler{Log: logger, Domains: tmplist}
	crawler.Crawl()
}

func main() {
	defer closeLogger() // ensure we're cleaning up the logger if there is one

	cli.VersionPrinter = func(c *cli.Context) {
		if version != "" && commithash != "" && compiledate != "" {
			fmt.Printf("version %s, revision %s (%s)\n", version, commithash, compiledate)
		} else if commithash != "" && compiledate != "" {
			fmt.Printf("revision %s (%s)\n", commithash, compiledate)
		} else if version != "" {
			fmt.Printf("version %s\n", version)
		} else {
			fmt.Println("version unknown")
		}
	}

	app := cli.NewApp()

	app.Name = "marill"

	if version != "" && commithash != "" {
		app.Version = fmt.Sprintf("%s, git revision %s", version, commithash)
	} else if version != "" {
		app.Version = version
	} else if commithash != "" {
		app.Version = "git revision " + commithash
	}

	app.Authors = []cli.Author{
		cli.Author{
			Name:  "Liam Stanley",
			Email: "me@liamstanley.io",
		},
	}
	app.Compiled = time.Now()
	app.Usage = "Automated website testing utility"

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:        "printurls",
			Usage:       "Print the list of urls as if they were going to be scanned",
			Destination: &conf.app.printUrls,
		},
		cli.BoolFlag{
			Name:        "debug, d",
			Usage:       "Print debugging information to stdout",
			Destination: &conf.out.printDebug,
		},
		cli.BoolFlag{
			Name:        "quiet, q",
			Usage:       "Dont't print regular stdout messages",
			Destination: &conf.out.ignoreStd,
		},
		cli.StringFlag{
			Name:        "log-file",
			Usage:       "Log debugging information to `logfile`",
			Destination: &conf.out.logFile,
		},
		cli.IntFlag{
			Name:        "cores",
			Usage:       "Use `n` cores to fetch data (0 being server cores/2)",
			Destination: &conf.scan.cores,
		},
	}

	app.Action = func(c *cli.Context) error {
		// initialize the logger. ensure this only occurs after the cli args are
		// pulled.
		initLogger()

		// initialize some form of max go procs
		numCores()

		// initialize the stats data
		done := make(chan struct{}, 1)
		go statsLoop(done)

		// close the stats data goroutine when we're complete.
		defer func() {
			done <- struct{}{}
		}()

		if conf.app.printUrls {
			if err := printUrls(); err != nil {
				fmt.Printf("err: %s", err)
				return err
			}

			os.Exit(0)
		}

		run()

		return nil
	}

	app.Run(os.Args)
}
