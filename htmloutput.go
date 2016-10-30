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
	Version  string
	MinScore float64
	Out      []*TestResult
	Success  bool
}

func genHTMLOutput(results []*TestResult) ([]byte, error) {
	out, err := json.Marshal(&JSONOutput{
		Version:  getVersion(),
		MinScore: 8.0,
		Out:      results,
		Success:  true,
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
	tmpl := template.Must(template.New("html").Parse(string(htmlTmpl)))

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
