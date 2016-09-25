package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

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

// supported tests:
//  - url: resource url
//  - host: resource host
//  - body: resource html-stripped body
//  - body_html: resource body
//  - code: resource status code
//  - headers: resource headers in string form
//  - asset_url: asset (js/css/img/png) url
//  - asset_code: asset status code
//  - asset_headers: asset headers in string form

// Test represents a type of check, comparing is the resource matches specific inputs
type Test struct {
	Name       string   `json:"name"`        // the name of the test
	Type       string   `json:"type"`        // type of test (see above)
	Weight     float64  `json:"weight"`      // how much does this test decrease or increase the score
	Bad        bool     `json:"bad"`         // decrease, or increase score if match?
	Match      []string `json:"match"`       // list of glob based matches
	MatchRegex []string `json:"match_regex"` // list of regex based matches

	OriginFile string // where the test originated from
}

// generateTests compiles a list of tests from bindata or a specified directory
func generateTests() (tests []*Test) {
	fns := AssetNames()
	logger.Printf("found %d test files", len(fns))

	for i := 0; i < len(fns); i++ {
		file, err := Asset(fns[i])
		if err != nil {
			out.Fatalf("unable to load asset %s: %s", fns[i], err)
		}

		testsFromFile := []*Test{}

		// check to see if it's an array of json tests
		err = json.Unmarshal(file, &testsFromFile)
		if err != nil {
			t := &Test{}

			// or just a single json test
			err2 := json.Unmarshal(file, &t)
			if err2 != nil {
				out.Fatalf("unable to load asset %s: %s", fns[i], err)
			}

			testsFromFile = append(testsFromFile, t)
		}

		blacklist := strings.Split(conf.scan.ignoreTest, "|")
		whitelist := strings.Split(conf.scan.matchTest, "|")

		for _, test := range testsFromFile {
			var matches bool
			test.OriginFile = fns[i]

			// check to see if it matches our blacklist. if so, ignore it
			if len(conf.scan.ignoreTest) > 0 {
				for _, match := range blacklist {
					if utils.Glob(test.Name, match) {
						matches = true
						break
					}
				}

				if matches {
					continue
				}
			}

			matches = false

			// check to see if it matches our whitelist. if not, ignore it.
			if len(conf.scan.matchTest) > 0 {
				for _, match := range whitelist {
					if utils.Glob(test.Name, match) {
						matches = true
						break
					}
				}

				if !matches {
					continue
				}
			}

			// loop through the regexp and ensure it's valid
			for re_i := 0; re_i < len(test.MatchRegex); re_i++ {
				_, err := regexp.Compile(test.MatchRegex[re_i])
				if err != nil {
					out.Fatalf("test '%s' (%s) has invalid regex (%s): %s", test.Name, test.OriginFile, test.MatchRegex[re_i], err)
				}
			}

			tests = append(tests, test)
		}
	}

	// ensure there are no duplicate tests
	names := []string{}
	for i := 0; i < len(tests); i++ {
		for n := 0; n < len(names); n++ {
			if names[n] == tests[i].Name {
				out.Fatalf("duplicate tests found for %s", tests[i].Name)
			}
		}
		names = append(names, tests[i].Name)
	}

	logger.Printf("found %d total tests", len(tests))

	return tests
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
	logger.Printf("applied test [%s::%s] score against %s to: %s%.2f (now %.2f)\n", test.Name, test.OriginFile, res.Domain.Resource.Response.URL.String(), visual, test.Weight, res.Score)
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

	body_nohtml := reHTMLTag.ReplaceAllString(dom.Response.Body, "")

	for _, t := range tests {
		logger.Printf("running test [%s::%s] against %s", t.Name, t.OriginFile, dom.Response.URL.String())
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
			res.testMatch(t, body_nohtml)
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
