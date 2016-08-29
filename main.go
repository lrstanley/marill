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

type outputConfig struct {
	noColors   bool
	printDebug bool
	logFile    string
}

type scanConfig struct{}

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

func run() {
	runtime.GOMAXPROCS(runtime.NumCPU() * 2)
	logger.Printf("limiting max threads to %d", runtime.NumCPU()*2)

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

func main() {
	defer closeLogger() // ensure we're cleaning up the logger if there is one

	app := cli.NewApp()

	app.Name = "marill"
	app.Version = "0.1.0"
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
		cli.StringFlag{
			Name:        "log-file",
			Usage:       "File to log debugging information",
			Destination: &conf.out.logFile,
		},
	}

	app.Action = func(c *cli.Context) error {
		// initialize the logger. ensure this only occurs after the cli args are
		// pulled.
		initLogger()

		if conf.app.printUrls {
			if err := printUrls(); err != nil {
				return err
			}

			os.Exit(0)
		}

		run()

		return nil
	}

	app.Run(os.Args)
}
