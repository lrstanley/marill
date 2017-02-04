// Author: Liam Stanley <me@liamstanley.io>
// Docs: https://marill.liam.sh/
// Repo: https://github.com/lrstanley/marill
//
//       O
//    o 0  o        [ Marill -- Automated site testing utility ]
//       O      ___      ___       __        _______    __    ___      ___
//     o       |"  \    /"  |     /""\      /"      \  |" \  |"  |    |"  |
//    [  ]      \   \  //   |    /    \    |:        | ||  | ||  |    ||  |
//    / O\      /\\  \/.    |   /' /\  \   |_____/   ) |:  | |:  |    |:  |
//   / o  \    |: \.        |  //  __'  \   //      /  |.  |  \  |___  \  |___
//  / O  o \   |.  \    /:  | /   /  \\  \ |:  __   \  /\  |\( \_|:  \( \_|:  \
// [________]  |___|\__/|___|(___/    \___)|__|  \___)(__\_|_)\_______)\_______)
//

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/lrstanley/marill/domfinder"
	"github.com/lrstanley/marill/scraper"
	"github.com/lrstanley/marill/utils"
	"github.com/urfave/cli"
)

// These should be defined during the make process, not always however.
var version, commithash, compiledate = "", "", ""

const updateURI = "https://api.github.com/repos/lrstanley/marill/releases/latest"
const docURI = "https://marill.liam.sh/"

const motd = `
{magenta}      {lightgreen}O{magenta}     {yellow}     [ Marill -- Automated site testing utility ]
{magenta}   {lightgreen}o{magenta} {lightgreen}0{magenta}  {lightgreen}o{magenta}   {lightyellow}             %4s, rev %s
{magenta}                    Documentation: %s
{magenta}      {lightgreen}O{magenta}     {lightblue} ___      ___       __        _______    __    ___      ___
{magenta}    {lightgreen}o{magenta}       {lightblue}|"  \    /"  |     /""\      /"      \  |" \  |"  |    |"  |
{magenta}   [  ]     {lightblue} \   \  //   |    /    \    |:        | ||  | ||  |    ||  |
{magenta}   / {lightmagenta}O{magenta}\     {lightblue} /\\  \/.    |   /' /\  \   |_____/   ) |:  | |:  |    |:  |
{magenta}  / {lightmagenta}o{magenta}  \    {lightblue}|: \.        |  //  __'  \   //      /  |.  |  \  |___  \  |___
{magenta} / {lightmagenta}O{magenta}  {lightmagenta}o{magenta} \   {lightblue}|.  \    /:  | /   /  \\  \ |:  __   \  /\  |\( \_|:  \( \_|:  \
{magenta}[________]  {lightblue}|___|\__/|___|(___/    \___)|__|  \___)(__\_|_)\_______)\_______)
`

var textTeplate = `
{{- if .Result.Error }}{red}{bold}[FAILURE]{c}
{{- else }}
	{{- if OutputConfig.ShowWarnings }}
		{{- if ne .FailedTests "" }}{yellow}{bold}[WARNING]{c}{{- else }}{green}{bold}[SUCCESS]{c}{{- end }}
	{{- else }}{green}{bold}[SUCCESS]{c}{{- end }}
{{- end }}

{{- /* add colors for the score */}} [score:
{{- if gt .Score ScanConfig.MinScore }}{green}
{{- else }}
	{{- if (or (le .Score ScanConfig.MinScore) (gt .Score 5.0)) }}{yellow}{{- end }}
{{- end }}
{{- if lt .Score 5.0 }}{red}{{- end }}

{{- .Score | printf "%5.1f/10" }}{c}]

{{- /* status code output */}}
{{- if .Result.Resource }} [code:{yellow}{{ if .Result.Response.Code }}{{ .Result.Response.Code }}{{ else }}---{{ end }}{c}]
{{- else }} [code:{red}---{c}]{{- end }}

{{- /* IP address */}}
{{- if .Result.Request.IP }} [{lightmagenta}{{ printf "%s" .Result.Request.IP }}{c}]{{- end }}

{{- /* number of assets */}}
{{- if .Result.Assets }} [{cyan}{{ printf "%d" (len .Result.Assets) }} assets{c}]{{- end }}

{{- /* response time for main resource */}}
{{- if not .Result.Error }} [{green}{{ .Result.Time.Milli }}ms{c}]{{- end }}

{{- " "}}{{- .Result.URL }}
{{- if .Result.Error }} ({red}errors: {{ .Result.Error }}{c})
{{- else }}
	{{- if OutputConfig.ShowWarnings }}
		{{- if ne .FailedTests "" }} ({yellow}warning: {{ .FailedTests }}{c}){{ end }}
	{{- end }}
{{- end }}`

// OutputConfig handles what the user sees (stdout, debugging, logs, etc).
type OutputConfig struct {
	NoColors     bool   // Don't print colors to stdout.
	NoBanner     bool   // Don't print the app banner.
	PrintDebug   bool   // Print debugging information.
	IgnoreStd    bool   // Ignore regular stdout (human-formatted).
	Log          string // Optional log file to dump regular logs.
	DebugLog     string // Optional log file to dump debugging info.
	ResultFile   string // Filename/path of file which to dump results to.
	ShowWarnings bool   // If warnings should be triggered when score > MinScore but not 10/10.
}

// ScanConfig handles how and what is scanned/crawled.
type ScanConfig struct {
	Threads       int           // Number of threads to run the scanner in.
	ManualList    string        // List of manually supplied domains.
	Assets        bool          // Pull all assets for the page.
	IgnoreSuccess bool          // Ignore urls/domains that were successfully fetched.
	AllowInsecure bool          // If SSL errors should be ignored.
	Delay         time.Duration // Delay for the stasrt of each resource crawl.
	HTTPTimeout   time.Duration // Timeout before http request becomes stale.

	// Domain filter related.
	IgnoreHTTP   bool   // Ignore http://.
	IgnoreHTTPS  bool   // Ignore https://.
	IgnoreRemote bool   // Ignore resources where the domain is using remote ip.
	IgnoreMatch  string // Glob match of domains to blacklist.
	MatchOnly    string // Glob match of domains to whitelist.

	// Test related.
	MinScore       float64 // Minimum score before a resource is considered "failed".
	IgnoreTest     string  // Glob match of tests to blacklist.
	MatchTest      string  // Glob match of tests to whitelist.
	TestsFromURL   string  // Load tests from a remote url.
	TestsFromPath  string  // Load tests from a specified path.
	IgnoreStdTests bool    // Don't execute standard builtin tests.

	// User input tests.
	TestPassText string // Glob match against body, will give it a weight of 10.
	TestFailText string // Glob match against body, will take away a weight of 10.

	// Output related.
	OutTmpl    string // The output text/template template for use with printing results.
	HTMLFile   string // The file path to dump results in html.
	JsonFile   string // The file path to dump results in json.
	JsonPretty bool   // Prettifies the json output.
}

// appConfig handles what the app does (scans/crawls, printing data, some
// other task, etc).
type appConfig struct {
	exitOnFail    bool // Exit with a status code of 1 if any of the domains failed.
	noUpdateCheck bool // Don't check to see if an update exists at Github.
}

// config is a wrapper for all the other configs to put them in one place.
type config struct {
	app  appConfig
	scan ScanConfig
	out  OutputConfig
}

var conf config

func getVersion() string {
	if version != "" && commithash != "" && compiledate != "" {
		return fmt.Sprintf("%s, git revision %s (compiled %s)", version, commithash, compiledate)
	} else if version != "" && commithash != "" {
		return fmt.Sprintf("%s, git revision %s", version, commithash)
	} else if version != "" {
		return version
	} else if commithash != "" {
		return "git revision " + commithash
	}

	return "unknown"
}

// statsLoop prints out memory/load/runtime statistics to debug output.
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
				"allocated mem: %dM, sys: %dM, routines: %d, cores: %d load5: %.2f load10: %.2f load15: %.2f",
				mem.Alloc/1024/1024, mem.Sys/1024/1024, numRoutines, numCPU, load5, load10, load15)

			time.Sleep(2 * time.Second)
		}
	}
}

// numThreads calculates the number of threads to use.
func numThreads() {
	// Use runtime.NumCPU for judgement call.
	if conf.scan.Threads < 1 {
		if runtime.NumCPU() >= 2 {
			conf.scan.Threads = runtime.NumCPU() / 2
		} else {
			conf.scan.Threads = 1
		}
	} else if conf.scan.Threads > runtime.NumCPU() {
		logger.Printf("warning: %d threads specified, which is more than the amount of cores", conf.scan.Threads)
		out.Printf("{yellow}warning: %d threads specified, which is more than the amount of cores on the server!{c}", conf.scan.Threads)

		// Set it to the amount of cores on the server. go will do this
		// regardless, so.
		conf.scan.Threads = runtime.NumCPU()
		logger.Printf("limiting number of threads to %d", conf.scan.Threads)
		out.Printf("limiting number of threads to %d", conf.scan.Threads)
	}

	logger.Printf("using %d cores (max %d)", conf.scan.Threads, runtime.NumCPU())

	return
}

// reManualDomain can match the following:
// (DOMAIN|URL):IP:PORT
// (DOMAIN|URL):IP
// (DOMAIN|URL):PORT
// (DOMAIN|URL)
var reManualDomain = regexp.MustCompile(`^(?P<domain>(?:[A-Za-z0-9_.-]{2,350}\.[A-Za-z0-9]{2,63})|https?://[A-Za-z0-9_.-]{2,350}\.[A-Za-z0-9]{2,63}[!-9;-~]+?)(?::(?P<ip>\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}))?(?::(?P<port>\d{2,5}))?$`)
var reSpaces = regexp.MustCompile(`[\t\n\v\f\r ]+`)

/// parseManualList parses the list of domains specified from --domains.
func parseManualList() (domlist []*scraper.Domain, err error) {
	input := strings.Split(reSpaces.ReplaceAllString(conf.scan.ManualList, " "), " ")

	for _, item := range input {
		item = strings.TrimSuffix(strings.TrimPrefix(item, " "), " ")
		if item == "" {
			continue
		}

		results := reManualDomain.FindStringSubmatch(item)
		if len(results) != 4 {
			return nil, NewErr{Code: ErrBadDomains, value: item}
		}

		domain, ip, port := results[1], results[2], results[3]

		if domain == "" {
			return nil, NewErr{Code: ErrBadDomains, value: item}
		}

		uri, err := utils.IsDomainURL(domain, port)
		if err != nil {
			return nil, NewErr{Code: ErrBadDomains, deepErr: err}
		}

		domlist = append(domlist, &scraper.Domain{
			URL: uri,
			IP:  ip,
		})
	}

	return domlist, nil
}

// printUrls prints the urls that /would/ be scanned, if we were to start
// crawling.
func printUrls(c *cli.Context) error {
	printBanner()

	if conf.scan.ManualList != "" {
		domains, err := parseManualList()
		if err != nil {
			out.Fatal(NewErr{Code: ErrDomains, deepErr: err})
		}

		for _, domain := range domains {
			out.Printf("{blue}%-40s{c} {green}%s{c}", domain.URL, domain.IP)
		}
	} else {
		finder := &domfinder.Finder{Log: logger}
		if err := finder.GetWebservers(); err != nil {
			out.Fatal(NewErr{Code: ErrProcList, deepErr: err})
		}

		if err := finder.GetDomains(); err != nil {
			out.Fatal(NewErr{Code: ErrGetDomains, deepErr: err})
		}

		finder.Filter(domfinder.DomainFilter{
			IgnoreHTTP:  conf.scan.IgnoreHTTP,
			IgnoreHTTPS: conf.scan.IgnoreHTTPS,
			IgnoreMatch: conf.scan.IgnoreMatch,
			MatchOnly:   conf.scan.MatchOnly,
		})

		if len(finder.Domains) == 0 {
			out.Fatal(NewErr{Code: ErrNoDomainsFound})
		}

		for _, domain := range finder.Domains {
			out.Printf("{blue}%-40s{c} {green}%s{c}", domain.URL, domain.IP)
		}
	}

	return nil
}

// listTests lists all loaded tests, based on supplied args to Marill.
func listTests(c *cli.Context) error {
	printBanner()

	tests := genTests()

	out.Printf("{lightgreen}%d{c} total tests found:", len(tests))

	for _, test := range tests {
		out.Printf("{lightblue}name:{c} %-30s {lightblue}weight:{c} %-6.2f {lightblue}origin:{c} %s", test.Name, test.Weight, test.Origin)

		if c.Bool("extended") {
			if len(test.Match) > 0 {
				out.Println("    - {cyan}Match ANY{c}:")
				for i := 0; i < len(test.Match); i++ {
					out.Println("        -", test.Match[i])
				}
			}

			if len(test.MatchAll) > 0 {
				out.Println("    - {cyan}Match ALL{c}:")
				for i := 0; i < len(test.MatchAll); i++ {
					out.Println("        -", test.MatchAll[i])
				}
			}

			out.Println("")
		}
	}

	return nil
}

func printBanner() {
	if len(version) != 0 && len(commithash) != 0 {
		logger.Printf("marill: version:%s revision:%s", version, commithash)
		if conf.out.NoBanner {
			out.Printf("{bold}{blue}marill version: %s (rev %s)", version, commithash)
			out.Printf("{bold}{magenta}documentation: %s", docURI)
		} else {
			out.Printf(motd, version, commithash, docURI)
		}
	} else {
		out.Println("{bold}{blue}Running marill (unknown version){c}")
	}

	out.Println("{black}{bold}\x1b[43m!! [WARNING] !!{c} {bold}{yellow}THIS IS AN ALPHA VERSION OF MARILL. NOTIFY IF ANYTHING IS BROKEN.{c}")
}

func updateCheck() {
	if len(version) == 0 {
		logger.Println("version not set, ignoring update check")
		return
	}

	client := &http.Client{
		Timeout: time.Duration(3) * time.Second,
	}

	req, err := http.NewRequest("GET", updateURI, nil)
	if err != nil {
		out.Println(NewErr{Code: ErrUpdateGeneric})
		logger.Println(NewErr{Code: ErrUpdate, value: "during check", deepErr: err})
		return
	}

	// Set the necessary headers per Github's request.
	req.Header.Set("User-Agent", "repo: lrstanley/marill (internal update check utility)")

	resp, err := client.Do(req)
	if err != nil {
		out.Println(NewErr{Code: ErrUpdateGeneric})
		logger.Println(NewErr{Code: ErrUpdateUnknownResp, deepErr: err})
		return
	}

	if resp.Body == nil {
		out.Println(NewErr{Code: ErrUpdateGeneric})
		logger.Println(NewErr{Code: ErrUpdateUnknownResp, value: "(no body?)"})
		return
	}
	defer resp.Body.Close() // Ensure the body is closed.

	bbytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		out.Println(NewErr{Code: ErrUpdateGeneric})
		logger.Println(NewErr{Code: ErrUpdate, value: "unable to convert body to bytes", deepErr: err})
		return
	}

	rawRemaining := resp.Header.Get("X-RateLimit-Remaining")
	if rawRemaining == "" {
		logger.Println(NewErr{Code: ErrUpdateUnknownResp, value: fmt.Sprintf("%s", bbytes)})
	}

	remain, err := strconv.Atoi(rawRemaining)
	if err != nil {
		logger.Println(NewErr{Code: ErrUpdate, value: "unable to convert api limit remaining to int", deepErr: err})
	}

	if remain < 10 {
		logger.Println("update check: warning, remaining api queries less than 10 (of 60!)")
	}

	if resp.StatusCode == 404 {
		out.Println("{green}update check: no updates found{c}")
		logger.Println("update check: no releases found.")
		return
	}

	if resp.StatusCode == 403 {
		// Fail silently.
		logger.Println(NewErr{Code: ErrUpdateUnknownResp, value: "status code 403 (too many update checks?)"})
		return
	}

	var data = struct {
		URL  string `json:"html_url"`
		Name string `json:"name"`
		Tag  string `json:"tag_name"`
	}{}

	err = json.Unmarshal(bbytes, &data)
	if err != nil {
		out.Println(NewErr{Code: ErrUpdateGeneric})
		logger.Println(NewErr{Code: ErrUpdate, value: "unable to unmarshal json body", deepErr: err})
		return
	}

	logger.Printf("update check: update info: %#v", data)

	if data.Tag == "" {
		out.Println(NewErr{Code: ErrUpdateGeneric})
		logger.Println(NewErr{Code: ErrUpdate, value: "unable to unmarshal json body", deepErr: err})
		return
	}

	if data.Tag == version {
		out.Println("{green}your version of marill is up to date{c}")
		logger.Printf("update check: up to date. current: %s, latest: %s (%s)", version, data.Tag, data.Name)
		return
	}

	out.Printf("{bold}{yellow}there is an update available for marill. current: %s new: %s (%s){c}", version, data.Tag, data.Name)
	out.Printf("release link: %s", data.URL)
	logger.Printf("update check found update %s available (doesn't match current: %s): %s", data.Tag, version, data.URL)

	return
}

func run(c *cli.Context) error {
	if len(conf.scan.ManualList) == 0 {
		conf.scan.ManualList = strings.Join(c.Args(), " ")
	}

	printBanner()

	if !conf.app.noUpdateCheck {
		updateCheck()
	}

	var text string
	if len(conf.scan.OutTmpl) > 0 {
		text = conf.scan.OutTmpl
	} else {
		text = textTeplate
	}

	var resultFn *os.File
	var err error
	if len(conf.out.ResultFile) > 0 {
		// Open up the requested resulting file. Make sure only to do this
		// in write-only and creation mode.
		resultFn, err = os.OpenFile(conf.out.ResultFile, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			out.Fatal(err)
		}
	}

	textFmt := text

	// Ensure we strip color from regular text (to output to a log file).
	StripColor(&text)

	// Ensure we check to see if they want color with regular output.
	FmtColor(&textFmt, conf.out.NoColors)

	tmplFuncMap := map[string]interface{}{
		"ScanConfig":   func() ScanConfig { return conf.scan },
		"OutputConfig": func() OutputConfig { return conf.out },
	}

	tmpl := template.Must(template.New("success").Funcs(tmplFuncMap).Parse(text + "\n"))
	tmplFormatted := template.Must(template.New("success").Funcs(tmplFuncMap).Parse(textFmt + "\n"))

	scan, err := crawl()
	if err != nil {
		out.Fatal(err)
	}

	for _, res := range scan.results {
		// Ignore successful, per request.
		if conf.scan.IgnoreSuccess && res.Result.Error == nil {
			continue
		}

		err = tmplFormatted.Execute(os.Stdout, res)
		if err != nil {
			out.Println("")
			out.Fatal("executing template:", err)
		}

		if len(conf.out.ResultFile) > 0 {
			// Pipe it to the result file as necessary.
			err = tmpl.Execute(resultFn, res)
			if err != nil {
				out.Println("")
				out.Fatal("executing template:", err)
			}
		}
	}

	var jsonOut *JSONOutput

	if conf.scan.JsonFile != "" || conf.scan.HTMLFile != "" {
		if jsonOut, err = genJSONOutput(scan); err != nil {
			out.Fatal(err)
		}
	}

	if conf.scan.JsonFile != "" {
		if conf.scan.JsonPretty {
			if err := ioutil.WriteFile(conf.scan.JsonFile, []byte(jsonOut.StringPretty()), 0666); err != nil {
				out.Fatal(err)
			}
		} else {
			if err := ioutil.WriteFile(conf.scan.JsonFile, jsonOut.Bytes(), 0666); err != nil {
				out.Fatal(err)
			}
		}
	}

	if conf.scan.HTMLFile != "" {
		htmlOut, err := genHTMLOutput(jsonOut)
		if err != nil {
			out.Fatal(err)
		}

		if err := ioutil.WriteFile(conf.scan.HTMLFile, htmlOut, 0666); err != nil {
			out.Fatal(err)
		}
	}

	// This should be the last thing we do.
	if conf.app.exitOnFail && scan.failed > 0 {
		out.Fatalf("exit-on-error enabled, %d errors. giving status code 1.", scan.failed)
	}

	return nil
}

func main() {
	defer closeLogger() // Ensure we're cleaning up the logger if there is one.

	cli.VersionPrinter = func(c *cli.Context) {
		if version != "" && commithash != "" && compiledate != "" {
			fmt.Printf("version %s, revision %s (compiled %s)\n", version, commithash, compiledate)
		} else if commithash != "" && compiledate != "" {
			fmt.Printf("revision %s (compiled %s)\n", commithash, compiledate)
		} else if version != "" && compiledate != "" {
			fmt.Printf("version %s (compiled %s)\n", version, compiledate)
		} else if version != "" {
			fmt.Printf("version %s\n", version)
		} else {
			fmt.Println("version unknown")
		}
	}

	app := cli.NewApp()

	app.Name = "marill"
	app.Version = getVersion()

	// Needed for stats look.
	done := make(chan struct{}, 1)

	app.Before = func(c *cli.Context) error {
		// Initialize the standard output.
		initOut(os.Stdout)

		// Initialize the logger.
		initLogger()

		// Initialize the max amount of threads to use.
		numThreads()

		// Initialize the stats data.
		go statsLoop(done)

		return nil
	}

	app.After = func(c *cli.Context) error {
		// Close the stats data goroutine when we're complete.
		done <- struct{}{}

		// Close Out log files or anything it has open.
		defer closeOut()

		// As with the debug log.
		defer closeLogger()

		return nil
	}

	app.Commands = []cli.Command{
		{
			Name:   "scan",
			Usage:  "[DEFAULT] Start scan for all domains on server",
			Action: run,
		},
		{
			Name:    "urls",
			Aliases: []string{"domains"},
			Usage:   "Print the list of urls as if they were going to be scanned",
			Action:  printUrls,
		},
		{
			Name:   "tests",
			Usage:  "Print the list of tests that are loaded and would be used",
			Action: listTests,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "extended",
					Usage: "Show exta test information",
				},
			},
		},
		{
			Name:   "ui",
			Usage:  "Display a GUI/TUI mouse-enabled UI that allows a more visual approach to Marill",
			Hidden: true,
			Action: func(c *cli.Context) error {
				if err := uiInit(); err != nil {
					closeOut()         // Close any open files that Out has open.
					initOut(os.Stdout) // Re-initialize with stdout so we can give them the error.
					out.Fatal(err)
				}
				os.Exit(0)

				return nil
			},
		},
	}

	app.Flags = []cli.Flag{
		// Output style flags.
		cli.BoolFlag{
			Name:        "d, debug",
			Usage:       "Print debugging information to stdout",
			Destination: &conf.out.PrintDebug,
		},
		cli.BoolFlag{
			Name:        "q, quiet",
			Usage:       "Do not print regular stdout messages",
			Destination: &conf.out.IgnoreStd,
		},
		cli.BoolFlag{
			Name:        "no-color",
			Usage:       "Do not print with color",
			Destination: &conf.out.NoColors,
		},
		cli.BoolFlag{
			Name:        "no-banner",
			Usage:       "Do not print the colorful banner",
			Destination: &conf.out.NoBanner,
		},
		cli.BoolFlag{
			Name:        "show-warnings",
			Usage:       "Show warnings if any tests failed, even it isn't a failure",
			Destination: &conf.out.ShowWarnings,
		},
		cli.BoolFlag{
			Name:        "exit-on-fail",
			Usage:       "Send exit code 1 if any domains fail tests",
			Destination: &conf.app.exitOnFail,
		},
		cli.StringFlag{
			Name:        "log",
			Usage:       "Log information to `FILE`",
			Destination: &conf.out.Log,
		},
		cli.StringFlag{
			Name:        "debug-log",
			Usage:       "Log debugging information to `FILE`",
			Destination: &conf.out.DebugLog,
		},
		cli.StringFlag{
			Name:        "result-file",
			Usage:       "Dump result template into `FILE` (will overwrite!)",
			Destination: &conf.out.ResultFile,
		},
		cli.BoolFlag{
			Name:        "no-updates",
			Usage:       "Don't check to see if there are updates",
			Destination: &conf.app.noUpdateCheck,
		},

		// Scan configuration.
		cli.IntFlag{
			Name:        "threads",
			Usage:       "Use `n` threads to fetch data (0 defaults to server cores/2)",
			Destination: &conf.scan.Threads,
		},
		cli.DurationFlag{
			Name:        "delay",
			Usage:       "Delay `DURATION` before each resource is crawled (e.g. 5s, 1m, 100ms)",
			Destination: &conf.scan.Delay,
		},
		cli.DurationFlag{
			Name:        "http-timeout",
			Usage:       "`DURATION` before an http request is timed out (e.g. 5s, 10s, 1m)",
			Destination: &conf.scan.HTTPTimeout,
			Value:       10 * time.Second,
		},
		cli.StringFlag{
			Name:        "domains",
			Usage:       "Manually specify list of domains to scan in form: `DOMAIN:IP ...`, or DOMAIN:IP:PORT",
			Destination: &conf.scan.ManualList,
		},
		cli.Float64Flag{
			Name:        "min-score",
			Usage:       "Minimum score for domain",
			Value:       8.0,
			Destination: &conf.scan.MinScore,
		},
		cli.BoolFlag{
			Name:        "a, assets",
			Usage:       "Crawl assets (css/js/images) for each page",
			Destination: &conf.scan.Assets,
		},
		cli.BoolFlag{
			Name:        "ignore-success",
			Usage:       "Only print results if they are considered failed",
			Destination: &conf.scan.IgnoreSuccess,
		},
		cli.BoolFlag{
			Name:        "allow-insecure",
			Usage:       "Don't check to see if an SSL certificate is valid",
			Destination: &conf.scan.AllowInsecure,
		},
		cli.StringFlag{
			Name:        "tmpl",
			Usage:       "Golang text/template string template for use with formatting scan output",
			Destination: &conf.scan.OutTmpl,
		},
		cli.StringFlag{
			Name:        "html",
			Usage:       "Optional `PATH` to output html results to",
			Hidden:      true,
			Destination: &conf.scan.HTMLFile,
		},
		cli.StringFlag{
			Name:        "json",
			Usage:       "Optional `PATH` to output json results to",
			Destination: &conf.scan.JsonFile,
		},
		cli.BoolFlag{
			Name:        "json-pretty",
			Usage:       "Used with [--json], pretty-prints the output json",
			Destination: &conf.scan.JsonPretty,
		},

		// Domain filtering.
		cli.BoolFlag{
			Name:        "ignore-http",
			Usage:       "Ignore http-based URLs during domain search",
			Destination: &conf.scan.IgnoreHTTP,
		},
		cli.BoolFlag{
			Name:        "ignore-https",
			Usage:       "Ignore https-based URLs during domain search",
			Destination: &conf.scan.IgnoreHTTPS,
		},
		cli.BoolFlag{
			Name:        "ignore-remote",
			Usage:       "Ignore all resources that resolve to a remote IP (use with --assets)",
			Destination: &conf.scan.IgnoreRemote,
		},
		cli.StringFlag{
			Name:        "domain-ignore",
			Usage:       "Ignore URLS during domain search that match `GLOB`, pipe separated list",
			Destination: &conf.scan.IgnoreMatch,
		},
		cli.StringFlag{
			Name:        "domain-match",
			Usage:       "Allow URLS during domain search that match `GLOB`, pipe separated list",
			Destination: &conf.scan.MatchOnly,
		},

		// Test filtering.
		cli.StringFlag{
			Name:        "test-ignore",
			Usage:       "Ignore tests that match `GLOB`, pipe separated list",
			Destination: &conf.scan.IgnoreTest,
		},
		cli.StringFlag{
			Name:        "test-match",
			Usage:       "Allow tests that match `GLOB`, pipe separated list",
			Destination: &conf.scan.MatchTest,
		},
		cli.StringFlag{
			Name:        "tests-url",
			Usage:       "Import tests from a specified `URL`",
			Destination: &conf.scan.TestsFromURL,
		},
		cli.StringFlag{
			Name:        "tests-path",
			Usage:       "Import tests from a specified file-system `PATH`",
			Destination: &conf.scan.TestsFromPath,
		},
		cli.BoolFlag{
			Name:        "ignore-std-tests",
			Usage:       "Ignores all built-in tests (useful with --tests-url)",
			Destination: &conf.scan.IgnoreStdTests,
		},
		cli.StringFlag{
			Name:        "pass-text",
			Usage:       "Give sites a +10 score if body matches `GLOB`",
			Destination: &conf.scan.TestPassText,
		},
		cli.StringFlag{
			Name:        "fail-text",
			Usage:       "Give sites a -10 score if body matches `GLOB`",
			Destination: &conf.scan.TestFailText,
		},
	}

	app.Authors = []cli.Author{
		{
			Name:  "Liam Stanley",
			Email: "me@liamstanley.io",
		},
	}
	app.Copyright = fmt.Sprintf("(c) %d Liam Stanley", time.Now().Year())
	app.Usage = "Automated website testing utility"
	app.Action = run

	if err := app.Run(os.Args); err != nil {
		log.Fatal(NewErr{Code: ErrInstantiateApp, deepErr: err})
	}
}
