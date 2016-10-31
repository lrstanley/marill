// Author: Liam Stanley <me@liamstanley.io>
// Docs: https://marill.liam.sh/
// Repo: https://github.com/Liamraystanley/marill

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
)

// JSONOutput is the generated json that will be embedded in Angular
type JSONOutput struct {
	Version    string
	MinScore   float64
	Out        []*HTMLDomResult
	Successful int
	Failed     int
	Success    bool
	HostFile   string
}

// HTMLDomResult is a wrapper around the test results, providing string representations
// of some errors and other items that during JSON conversion get converted to structs.
type HTMLDomResult struct {
	*TestResult
	ErrorString string // string representation of any errors
}

func genHTMLOutput(scan *Scan) ([]byte, error) {
	htmlConvertedResults := make([]*HTMLDomResult, len(scan.results))
	var hosts string

	for i := 0; i < len(scan.results); i++ {
		htmlConvertedResults[i] = &HTMLDomResult{TestResult: scan.results[i]}
		if htmlConvertedResults[i].Result.Error != nil {
			htmlConvertedResults[i].ErrorString = htmlConvertedResults[i].Result.Error.Error()
		}

		if htmlConvertedResults[i].Result.Request.IP != "" {
			hosts += fmt.Sprintf("%s %s\n", htmlConvertedResults[i].Result.Request.IP, htmlConvertedResults[i].Result.Request.URL.Host)
		}
	}

	out, err := json.Marshal(&JSONOutput{
		Version:    getVersion(),
		MinScore:   8.0,
		Out:        htmlConvertedResults,
		Successful: scan.successful,
		Failed:     scan.failed,
		HostFile:   hosts,
		Success:    true,
	})
	if err != nil {
		return nil, err
	}

	htmlTmpl, err := Asset("data/html/index.html")
	if err != nil {
		return nil, err
	}
	jsTmpl, err := Asset("data/html/main.js")
	if err != nil {
		return nil, err
	}
	cssTmpl, err := Asset("data/html/main.css")
	if err != nil {
		return nil, err
	}

	jsonStr := fmt.Sprintf("%s", out)
	tmpl := template.New("html")
	tmpl.Delims("{[", "]}")
	tmpl = template.Must(tmpl.Parse(string(htmlTmpl)))

	var buf bytes.Buffer
	tmpl.Execute(&buf, struct {
		JSON string
		JS   string
		CSS  string
	}{
		JSON: jsonStr,
		JS:   string(jsTmpl),
		CSS:  string(cssTmpl),
	})

	return buf.Bytes(), nil
}
