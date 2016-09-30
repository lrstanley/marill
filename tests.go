// Author: Liam Stanley <me@liamstanley.io>
// Docs: https://marill.liam.sh/
// Repo: https://github.com/Liamraystanley/marill

package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Liamraystanley/marill/scraper"
	"github.com/Liamraystanley/marill/utils"
)

// TODO:
//   Thinking about how test scores are going to be kept. Say, it starts off with a
//   score of 10. each test will mark it up, or down.
//
//   Also:
//      - if 25% of resources fail to load, or over 30 (for sites with 200+ assets), auto fail the result?

const (
	defaultScore = 10.0
)

var defaultTestTypes = [...]string{
	"url",           // resource url
	"host",          // resource host
	"body",          // resource html-stripped body
	"body_html",     // resource body
	"code",          // resource status code
	"headers",       // resource headers in string form
	"asset_url",     // asset (js/css/img/png) url
	"asset_code",    // asset status code
	"asset_headers", // asset headers in string form
}

// Test represents a type of check, comparing is the resource matches specific inputs
type Test struct {
	Name       string   `json:"name"`        // the name of the test
	Type       string   `json:"type"`        // type of test (see above)
	Weight     float64  `json:"weight"`      // how much does this test decrease or increase the score
	Bad        bool     `json:"bad"`         // decrease, or increase score if match?
	Match      []string `json:"match"`       // list of glob based matches
	MatchRegex []string `json:"match_regex"` // list of regex based matches

	Origin string // where the test originated from
}

// parseTests parses a json object or array from a byte array (file, url, etc)
func parseTests(raw []byte, originType, origin string) (tests []*Test, err error) {
	tmp := []*Test{}

	// check to see if it's an array of json tests
	err = json.Unmarshal(raw, &tmp)
	if err != nil {
		t := &Test{}

		// or just a single json test
		err2 := json.Unmarshal(raw, &t)
		if err2 != nil {
			return nil, fmt.Errorf("unable to load asset from %s %s: %s", originType, origin, err)
		}

		tmp = append(tmp, t)
	}

	for i := range tmp {
		tmp[i].Origin = fmt.Sprintf("%s:%s", originType, origin)
		tests = append(tests, tmp[i])
	}

	return tests, nil
}

// genTests compiles a list of tests from various locations
func genTests() (tests []*Test) {
	tmp := []*Test{}

	genTestsFromStd(&tmp)
	genTestsFromPath(&tmp)
	genTestsFromURL(&tmp)

	blacklist := strings.Split(conf.scan.ignoreTest, "|")
	whitelist := strings.Split(conf.scan.matchTest, "|")

	// loop through each test and ensure that they match our criteria, and are safe
	// to start testing against
	for _, test := range tmp {
		var matches bool

		// check to see if it matches our blacklist. if so, ignore it
		if len(conf.scan.ignoreTest) > 0 {
			for _, match := range blacklist {
				if utils.Glob(test.Name, match) {
					matches = true
					break
				}
			}

			if matches {
				continue // skip
			}
		}

		matches = false

		// check to see if it matches our whitelist. if not, ignore it.
		if len(conf.scan.matchTest) > 0 {
			for _, match := range whitelist {
				if !utils.Glob(test.Name, match) {
					matches = true
					break
				}
			}

			if matches {
				continue // skip
			}
		}

		// check to see if the type matches the builtin list of types
		var isin bool
		for i := 0; i < len(defaultTestTypes); i++ {
			if test.Type == defaultTestTypes[i] {
				isin = true
				break
			}
		}
		if !isin {
			out.Fatalf("test '%s' (%s) has invalid type", test.Name, test.Origin)
		}

		// loop through the regexp and ensure it's valid
		for i := 0; i < len(test.MatchRegex); i++ {
			_, err := regexp.Compile(test.MatchRegex[i])
			if err != nil {
				out.Fatalf("test '%s' (%s) has invalid regex (%s): %s", test.Name, test.Origin, test.MatchRegex[i], err)
			}
		}

		tests = append(tests, test)
	}

	// ensure there are no duplicate tests
	names := []string{}
	for i := 0; i < len(tests); i++ {
		for n := 0; n < len(names); n++ {
			if names[n] == tests[i].Name {
				out.Fatalf("duplicate tests found for %s (origin: %s)", tests[i].Name, tests[i].Origin)
			}
		}
		names = append(names, tests[i].Name)
	}

	logger.Printf("found %d total tests", len(tests))

	return tests
}

// genTestsFromStd reads from builtin tests (e.g. bindata)
func genTestsFromStd(tests *[]*Test) {
	if conf.scan.ignoreStdTests {
		logger.Print("ignoring all standard (built-in) tests per request")
	} else {
		fns := AssetNames()
		logger.Printf("found %d test files", len(fns))

		for i := 0; i < len(fns); i++ {
			file, err := Asset(fns[i])
			if err != nil {
				out.Fatalf("unable to load asset from file %s: %s", fns[i], err)
			}

			parsedTests, err := parseTests(file, "file-builtin", fns[i])
			if err != nil {
				out.Fatal(err)
			}

			*tests = append(*tests, parsedTests...)
		}
	}
}

// genTestsFromPath reads tests from a user-specified path
func genTestsFromPath(tests *[]*Test) {
	if len(conf.scan.testsFromPath) == 0 {
		return
	}

	var matches []string

	var testPathCheck = func(path string, info os.FileInfo, err error) error {
		if err != nil {
			out.Fatalf("unable to open file '%s' for reading: %s", path, err)
		}

		if info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".json") {
			return nil
		}

		matches = append(matches, path)

		return nil
	}

	err := filepath.Walk(conf.scan.testsFromPath, testPathCheck)
	if err != nil {
		out.Fatalf("unable to scan path '%s' for tests: %s", conf.scan.testsFromPath, err)
	}

	for i := 0; i < len(matches); i++ {
		file, err := ioutil.ReadFile(matches[i])
		if err != nil {
			out.Fatalf("unable to open file '%s' for reading: %s", matches[i], err)
		}

		parsedTests, err := parseTests(file, "file-path", matches[i])
		if err != nil {
			out.Fatalf("unable to parse JSON from file '%s': %s", matches[i], err)
		}

		*tests = append(*tests, parsedTests...)
	}
}

// genTestsFromURL reads tests from a user-specified remote http-url
func genTestsFromURL(tests *[]*Test) {
	if len(conf.scan.testsFromURL) == 0 {
		return
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: transport,
	}

	req, err := http.NewRequest("GET", conf.scan.testsFromURL, nil)
	if err != nil {
		out.Fatalf("unable to load tests from %s: %s", conf.scan.testsFromURL, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		out.Fatalf("in fetch of tests from %s: %s", conf.scan.testsFromURL, err)
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		out.Fatalf("unable to parse JSON from %s: %s", conf.scan.testsFromURL, err)
	}

	parsedTests, err := parseTests(bodyBytes, "url", conf.scan.testsFromURL)
	if err != nil {
		out.Fatal(err)
	}

	*tests = append(*tests, parsedTests...)
}

// checkTests iterates over all domains and runs checks across all domains
func checkTests(results []*scraper.Results, tests []*Test) (completedTests []*TestResult) {
	timer := utils.NewTimer()
	logger.Print("starting test checks")

	for _, dom := range results {
		completedTests = append(completedTests, checkDomain(dom, tests))
	}

	timer.End()
	logger.Printf("finished tests, elapsed time: %ds\n", timer.Result.Seconds)

	for i := 0; i < len(completedTests); i++ {
		if completedTests[i].Domain.Error != nil {
			continue
		}

		if completedTests[i].Score < conf.scan.minScore {
			failedTests := []string{}
			for k := range completedTests[i].MatchedTests {
				failedTests = append(failedTests, k)
			}

			completedTests[i].Domain.Error = errors.New("failed tests: " + strings.Join(failedTests, ", "))
		}
	}

	return completedTests
}

// TestResult represents the result of testing a single resource
type TestResult struct {
	Domain       *scraper.Results   // Origin domain/resource data
	Score        float64            // resulting score, skewed off defaultScore
	MatchedTests map[string]float64 // map of negative affecting tests that were applied
}

// applyScore applies the score from test to the result, assuming test matched
func (res *TestResult) applyScore(test *Test) {
	// TODO: what did it match?

	if test.Bad {
		res.Score = res.Score - test.Weight

		if _, ok := res.MatchedTests[test.Name]; !ok {
			res.MatchedTests[test.Name] = 0.0
		}
		res.MatchedTests[test.Name] = res.MatchedTests[test.Name] - test.Weight
	} else {
		res.Score = res.Score + test.Weight
	}

	visual := "-"
	if !test.Bad {
		visual = "+"
	}
	logger.Printf("applied test [%s::%s] score against %s to: %s%.2f (now %.2f)\n", test.Name, test.Origin, res.Domain.Resource.Response.URL.String(), visual, test.Weight, res.Score)
}

// testMatch compares the input test match parameters with the input strings
func (res *TestResult) testMatch(test *Test, data string) {
	// loop through test.Match as GLOB
	for i := 0; i < len(test.Match); i++ {
		if utils.Glob(data, test.Match[i]) {
			res.applyScore(test)
			return
		}
	}

	// ...and test.MatchRegex
	for i := 0; i < len(test.MatchRegex); i++ {
		re := regexp.MustCompile(test.MatchRegex[i])
		if re.MatchString(data) {
			res.applyScore(test)
			return
		}
	}

	return
}

var reHTMLTag = regexp.MustCompile(`<[^>]+>`)

// checkDomain loops through all tests and guages what test score the domain gets
func checkDomain(dom *scraper.Results, tests []*Test) *TestResult {
	res := &TestResult{Domain: dom, Score: defaultScore, MatchedTests: make(map[string]float64)}

	if dom.Error != nil {
		res.Score = 0
		return res
	}

	bodyNoHTML := reHTMLTag.ReplaceAllString(dom.Response.Body, "")

	for _, t := range tests {
		logger.Printf("running test [%s::%s] against %s", t.Name, t.Origin, dom.Response.URL.String())
		switch t.Type {
		case "url":
			res.testMatch(t, dom.Response.URL.String())
		case "host":
			res.testMatch(t, dom.Response.URL.Host)
		case "asset_url":
			for i := 0; i < len(dom.Resources); i++ {
				res.testMatch(t, dom.Resources[i].Response.URL.String())
			}
		case "body":
			res.testMatch(t, bodyNoHTML)
		case "body_html":
			res.testMatch(t, dom.Response.Body)
		case "code":
			res.testMatch(t, strconv.Itoa(dom.Response.Code))
		case "asset_code":
			for i := 0; i < len(dom.Resources); i++ {
				res.testMatch(t, strconv.Itoa(dom.Resources[i].Response.Code))
			}
		case "headers":
			for name, values := range dom.Response.Headers {
				hv := fmt.Sprintf("%s: %s", name, strings.Join(values, " "))
				fmt.Printf("%#v\n", hv)

				res.testMatch(t, hv)
			}
		case "asset_headers":
			for i := 0; i < len(dom.Resources); i++ {
				for name, values := range dom.Resources[i].Response.Headers {
					hv := fmt.Sprintf("%s: %s", name, strings.Join(values, " "))
					fmt.Printf("%#v\n", hv)

					res.testMatch(t, hv)
				}
			}
		}
	}

	return res
}
