package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/Liamraystanley/marill/domfinder"
	"github.com/Liamraystanley/marill/scraper"
	"github.com/Liamraystanley/marill/utils"
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
	cores        int
	manualList   string
	ignoreHttp   bool
	ignoreHttps  bool
	ignoreRemote bool
	ignoreMatch  string
	matchOnly    string
	recursive    bool
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
	var load5, load10, load15 float32

	for {
		select {
		case <-done:
			return
		default:
			runtime.ReadMemStats(mem)
			numRoutines = runtime.NumGoroutine()
			numCPU = runtime.NumCPU()

			if contents, err := ioutil.ReadFile("/proc/loadavg"); err == nil {
				fmt.Sscanf(string(contents), "%f %f %f %*s %*d", &load5, &load10, &load15)
			}

			logger.Printf(
				"allocated mem: %dM, sys: %dM, threads: %d, cores: %d load5: %.2f load10: %.2f load15: %.2f",
				mem.Alloc/1024/1024, mem.Sys/1024/1024, numRoutines, numCPU, load5, load10, load15)

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

// reManualDomain can match the following:
// (DOMAIN|URL):IP:PORT
// (DOMAIN|URL):IP
// (DOMAIN|URL):PORT
// (DOMAIN|URL)
var reManualDomain = regexp.MustCompile(`^(?P<domain>(?:[A-Za-z0-9_.-]{2,350}\.[A-Za-z0-9]{2,63})|https?://[A-Za-z0-9_.-]{2,350}\.[A-Za-z0-9]{2,63}[!-~]+?)(?::(?P<ip>\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}))?(?::(?P<port>\d{2,5}))?$`)
var reSpaces = regexp.MustCompile(`[\t\n\v\f\r ]+`)

/// parseManualList parses the list of domains specified from --domains
func parseManualList() (domlist []*scraper.Domain, err error) {
	input := strings.Split(reSpaces.ReplaceAllString(conf.scan.manualList, " "), " ")

	for _, item := range input {
		item = strings.TrimSuffix(strings.TrimPrefix(item, " "), " ")
		if item == "" {
			continue
		}

		results := reManualDomain.FindStringSubmatch(item)
		if len(results) != 4 {
			return nil, fmt.Errorf("invalid domain manually provided: %s", item)
		}

		domain, ip, port := results[1], results[2], results[3]

		if domain == "" {
			return nil, fmt.Errorf("invalid domain manually provided: %s", item)
		}

		uri, err := utils.IsDomainURL(domain, port)
		if err != nil {
			return nil, fmt.Errorf("invalid domain manually provided: %s", err)
		}

		domlist = append(domlist, &scraper.Domain{
			URL: uri,
			IP:  ip,
		})
	}

	return domlist, nil
}

func printUrls() error {
	if conf.scan.manualList != "" {
		domains, err := parseManualList()
		if err != nil {
			return fmt.Errorf("unable to parse domain list: %s", err)
		}

		for _, domain := range domains {
			out.Printf("{blue}%-40s{c} {green}%s{c}\n", domain.URL, domain.IP)
		}
	} else {
		finder := &domfinder.Finder{Log: logger}
		if err := finder.GetWebservers(); err != nil {
			return fmt.Errorf("unable to get process list: %s", err)
		}

		if err := finder.GetDomains(); err != nil {
			return fmt.Errorf("unable to auto-fetch domain list: %s", err)
		}

		finder.Filter(domfinder.DomainFilter{
			IgnoreHTTP:  conf.scan.ignoreHttp,
			IgnoreHTTPS: conf.scan.ignoreHttps,
			IgnoreMatch: conf.scan.ignoreMatch,
			MatchOnly:   conf.scan.matchOnly,
		})

		for _, domain := range finder.Domains {
			out.Printf("{blue}%-40s{c} {green}%s{c}\n", domain.URL, domain.IP)
		}
	}

	return nil
}

func run() {
	if len(version) != 0 && len(commithash) != 0 {
		out.Printf("{bold}{blue}Running marill version %s (git revision %s){c}\n", version, commithash)
		logger.Printf("marill: version:%s revision:%s\n", version, commithash)
	} else {
		out.Println("{bold}{blue}Running marill (unknown version){c}")
	}

	crawler := &scraper.Crawler{Log: logger}
	var err error

	if conf.scan.manualList != "" {
		logger.Println("manually supplied url list")
		crawler.Cnf.Domains, err = parseManualList()
		if err != nil {
			out.Println("{red}Unable to parse --domains:{c}", err)
			logger.Fatal("unable to parse domain list:", err)
		}
	} else {
		logger.Println("checking for running webservers")

		finder := &domfinder.Finder{Log: logger}
		if err := finder.GetWebservers(); err != nil {
			logger.Fatalf("unable to get process list: %s", err)
		}

		if outlist := ""; len(finder.Procs) > 0 {
			for _, proc := range finder.Procs {
				outlist += fmt.Sprintf("[%s:%s] ", proc.Name, proc.PID)
			}
			logger.Printf("found %d procs matching a webserver: %s", len(finder.Procs), outlist)
			out.Printf("found %d procs matching a webserver\n", len(finder.Procs))
		}

		// start crawling for domains
		if err := finder.GetDomains(); err != nil {
			logger.Fatalf("unable to auto-fetch domain list: %s", err)
		}

		finder.Filter(domfinder.DomainFilter{
			IgnoreHTTP:  conf.scan.ignoreHttp,
			IgnoreHTTPS: conf.scan.ignoreHttps,
			IgnoreMatch: conf.scan.ignoreMatch,
			MatchOnly:   conf.scan.matchOnly,
		})

		logger.Printf("found %d domains on webserver %s (exe: %s, pid: %s)", len(finder.Domains), finder.MainProc.Name, finder.MainProc.Exe, finder.MainProc.PID)

		for _, domain := range finder.Domains {
			crawler.Cnf.Domains = append(crawler.Cnf.Domains, &scraper.Domain{URL: domain.URL, IP: domain.IP})
		}
	}

	crawler.Cnf.Recursive = conf.scan.recursive
	crawler.Cnf.NoRemote = conf.scan.ignoreRemote

	logger.Printf("starting crawler...")
	out.Printf("Starting scan on %d domains...\n", len(crawler.Cnf.Domains))
	crawler.Crawl()
	out.Println("Scan complete.")

	for _, dom := range crawler.Results {
		if dom.Error != nil {
			out.Printf("{red}[FAILURE]{c} [code: ---] [%15s] [{cyan}  0 resources{c}] [{green}     0ms{c}] %s ({red}%s{c})\n", dom.Request.IP, dom.Request.URL, dom.Error)
		} else {
			out.Printf("{green}[SUCCESS]{c} [code: {yellow}%d{c}] [%15s] [{cyan}%3d resources{c}] [{green}%6dms{c}] %s\n", dom.Resource.Response.Code, dom.Request.IP, len(dom.Resources), dom.Resource.Time.Milli, dom.Resource.Response.URL)
		}
	}

	checkTests(crawler.Results)
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
			Name:        "debug, d",
			Usage:       "Print debugging information to stdout",
			Destination: &conf.out.printDebug,
		},
		cli.BoolFlag{
			Name:        "quiet, q",
			Usage:       "Do not print regular stdout messages",
			Destination: &conf.out.ignoreStd,
		},
		cli.BoolFlag{
			Name:        "no-color",
			Usage:       "Do not print with color",
			Destination: &conf.out.noColors,
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
		cli.StringFlag{
			Name:        "domains",
			Usage:       "Manually specify list of domains to scan in form: `DOMAIN:IP ...`, or DOMAIN:IP:PORT",
			Destination: &conf.scan.manualList,
		},
		cli.BoolFlag{
			Name:        "print-urls",
			Usage:       "Print the list of urls as if they were going to be scanned",
			Destination: &conf.app.printUrls,
		},
		cli.BoolFlag{
			Name:        "ignore-http",
			Usage:       "Ignore http-based URLs during domain search",
			Destination: &conf.scan.ignoreHttp,
		},
		cli.BoolFlag{
			Name:        "ignore-https",
			Usage:       "Ignore https-based URLs during domain search",
			Destination: &conf.scan.ignoreHttps,
		},
		cli.BoolFlag{
			Name:        "ignore-remote",
			Usage:       "Ignore all resources that resolve to a remote IP (use with --recursive)",
			Destination: &conf.scan.ignoreRemote,
		},
		cli.StringFlag{
			Name:        "domain-ignore",
			Usage:       "Ignore URLS during domain search that match `GLOB`",
			Destination: &conf.scan.ignoreMatch,
		},
		cli.StringFlag{
			Name:        "domain-match",
			Usage:       "Allow URLS during domain search that match `GLOB`",
			Destination: &conf.scan.matchOnly,
		},
		cli.BoolFlag{
			Name:        "recursive, r",
			Usage:       "Check all assets (css/js/images) for each page, recursively",
			Destination: &conf.scan.recursive,
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
