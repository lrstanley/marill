package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"github.com/Liamraystanley/marill/scraper"
	"github.com/Liamraystanley/marill/utils"
)

var example = `
{
        "name": "Example test x2",
        "type": "url",
        "weight": 0.5,
        "bad": true,
        "match": ["*liquid*"],
        "match_regex": []
}
`

// TODO:
//   Thinking about how test scores are going to be kept. Say, it starts off with a
//   score of 10. each test will mark it up, or down.
//
//   Also:
//      - if 25% of resources fail to load, or over 30 (for sites with 200+ assets), auto fail the result?

type Test struct {
	Name       string   `json:"name"`        // the name of the test
	Type       string   `json:"type"`        // type of test. e.g: url, host, resource, body, statuscode, ip
	Weight     float32  `json:"weight"`      // how much does this test decrease or increase the score
	Bad        bool     `json:"bad"`         // decrease, or increase score if match?
	Match      []string `json:"match"`       // list of glob based matches
	MatchRegex []string `json:"match_regex"` // list of regex based matches
}

func generateTests() (tests []*Test) {
	// for now, just return dummy test
	t := &Test{}

	err := json.Unmarshal([]byte(example), &t)
	if err != nil {
		panic(err)
	}

	for i := 0; i < len(t.MatchRegex); i++ {
		// TODO: manual error returns are more useful
		_ = regexp.MustCompile(t.MatchRegex[i]) // make sure it is valid or panic
	}

	tests = append(tests, t)

	return
}

func checkTests(results []*scraper.Results) {
	tests := generateTests()
	timer := utils.NewTimer()
	logger.Print("starting test checks")

	for _, dom := range results {
		fmt.Printf("%#v\n", checkDomain(dom, tests))
	}

	timer.End()
	logger.Printf("finished tests, elapsed time: %ds\n", timer.Result.Seconds)
}

type TestResult struct {
	Domain *scraper.Results
	Score  float32
}

// applyScore applies the score from test to the result, assuming test matched
func (res *TestResult) applyScore(test *Test) {
	// TODO: add a list of score changes, useful for debugging?
	// e.g. test "something here" had -3, test "other" had +3
	// TODO: add a relation to the above. E.g. what did it match?
	if test.Bad {
		res.Score = res.Score - test.Weight
	} else {
		res.Score = res.Score + test.Weight
	}

	visual := "-"
	if !test.Bad {
		visual = "+"
	}
	fmt.Printf("applied score for %s to: %s%.2f (now %.2f)\n", res.Domain.Resource.Response.URL.String(), visual, test.Weight, res.Score)
}

func checkDomain(dom *scraper.Results, tests []*Test) *TestResult {
	res := &TestResult{Domain: dom, Score: 10.0}

	if dom.Error != nil {
		res.Score = 0
		return res
	}

	// TODO: Complete IP address lookups for use with "ip" type

	for _, t := range tests {
		switch t.Type {
		case "url":
			if testMatch(t, dom.Response.URL.String()) {
				res.applyScore(t)
			}
		case "host":
			if testMatch(t, dom.Response.URL.Host) {
				res.applyScore(t)
			}
		case "resource":
			for i := 0; i < len(dom.Resources); i++ {
				if testMatch(t, dom.Resources[i].Response.URL.String()) {
					res.applyScore(t)
				}
			}
		case "body":
			if testMatch(t, dom.Response.Body) {
				res.applyScore(t)
			}
		case "statuscode":
			if testMatch(t, strconv.Itoa(dom.Response.Code)) {
				res.applyScore(t)
			}
		}
	}

	return res
}

func testMatch(test *Test, data string) bool {
	// loop through test.Match as GLOB
	for i := 0; i < len(test.Match); i++ {
		if utils.Glob(data, test.Match[i]) {
			return true
		}
	}

	// ...and test.MatchRegex
	for i := 0; i < len(test.MatchRegex); i++ {
		re := regexp.MustCompile(test.MatchRegex[i])
		if re.MatchString(data) {
			return true
		}
	}

	return false
}
