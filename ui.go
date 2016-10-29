// Author: Liam Stanley <me@liamstanleyio>
// Docs: https://marill.liamsh/
// Repo: https://githubcom/Liamraystanley/marill

package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"strings"

	"github.com/Liamraystanley/marill/domfinder"
	"github.com/Liamraystanley/marill/scraper"
	"github.com/Liamraystanley/marill/utils"
	"github.com/jroimartin/gocui"
)

// mMenu holds X/Y coords of the menu for calculation from other views
type mMenu struct {
	maxX, maxY                                     int
	scan                                           *Scan
	domains, summary, details, failures, successes *gocui.View
}

var menu mMenu

// centerText takes a string of text and a length and pads the beginning
// of the string with spaces to center that text in the available space
func centerText(text string, maxX int) string {
	numSpaces := maxX/2 - len(text)/2
	for i := 1; i < numSpaces; i++ {
		text = " " + text
	}

	return text
}

// padText takes a string of text and pads the end of it with spaces to
// fill the available space in a cell
func padText(text string, maxX int) string {
	numSpaces := maxX - len(text)
	for i := 0; i < numSpaces; i++ {
		text += " "
	}

	return text
}

// readSel reads the currently selected line and returns a string
// containing its contents, without trailing spaces
func readSel(view *gocui.View) string {
	_, posY := view.Cursor()
	selection, _ := view.Line(posY)
	selection = strings.TrimSpace(selection)

	return selection
}

// drawTitle adds the title to the top of the menu
func drawTitle(gooey *gocui.Gui) error {
	// place the title view at the top of the menu and extend it down two lines
	if title, err := gooey.SetView("title", 0, 0, menu.maxX-1, 2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		fmt.Fprintln(title, centerText("Marill: Automated site testing utility", menu.maxX))

	}

	return nil
}

// drawSidebar draws the sidebar on the left side of the gui
func drawSidebar(gooey *gocui.Gui) error {
	// find minY, which will be the bottom of the header view
	_, _, _, minY, err := gooey.ViewPosition("title")
	if err != nil {
		log.Fatal(err)
	}

	maxX := menu.maxX / 6

	// create a view to hold the sidebar header and print it
	if sideHead, err := gooey.SetView("sideHead", 0, minY, maxX, minY+2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		fmt.Fprintln(sideHead, centerText("Abilities", maxX))
	}

	// create a view for the sidebar itself
	if sidebar, err := gooey.SetView("sidebar", 0, minY+2, maxX, menu.maxY-3); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		// print options and ensure highlights are enabled
		fmt.Fprintln(sidebar, padText("Domains", maxX))
		fmt.Fprintln(sidebar, padText("Summary", maxX))
		fmt.Fprintln(sidebar, padText("Details", maxX))
		fmt.Fprintln(sidebar, padText("Failures", maxX))
		fmt.Fprintln(sidebar, padText("Successes", maxX))
		sidebar.Highlight = true

	}

	return nil
}

// drawDomains draws the domains view
func drawDomains(gooey *gocui.Gui) error {
	// find minY, which will be the bottom of the title view
	_, _, _, minY, err := gooey.ViewPosition("title")
	if err != nil {
		log.Fatal(err)
	}

	// find minX, which will be the right edge of the sidebar view
	_, _, minX, _, err := gooey.ViewPosition("sidebar")
	if err != nil {
		log.Fatal(err)
	}

	// create a view to hold the domains header and print it
	if domHead, err := gooey.SetView("domHead", minX, minY, menu.maxX-1, minY+2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		fmt.Fprintln(domHead, centerText("Scan Domains", menu.maxX-minX))
	}

	// create the domains view
	if domains, err := gooey.SetView("domains", minX, minY+2, menu.maxX-1, menu.maxY-3); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		// ensure the view is editable and store a pointer to the view for quick access
		domains.Editable = true
		menu.domains = domains

	}

	return nil
}

// drawSummary draws the summary view
func drawSummary(gooey *gocui.Gui) error {
	// find minY, which will be the bottom of the header view
	_, _, _, minY, err := gooey.ViewPosition("title")
	if err != nil {
		log.Fatal(err)
	}

	// find minX, which will be the right edge of the sidebar view
	_, _, minX, _, err := gooey.ViewPosition("sidebar")
	if err != nil {
		log.Fatal(err)
	}

	// create a view to hold the summary header and print it
	if sumHead, err := gooey.SetView("sumHead", minX, minY, menu.maxX-1, minY+2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		fmt.Fprintln(sumHead, centerText("Results Summary", menu.maxX-minX))
	}

	// create the summary view
	if summary, err := gooey.SetView("summary", minX, minY+2, menu.maxX-1, menu.maxY-3); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		// store a pointer to the view for quick access and initialize the view as the default log location
		menu.summary = summary
		initOutWriter(summary)
		fmt.Fprintln(summary, "Scan summary will be available once the scan started.")
	}

	return nil
}

// drawDetails draws the details view
func drawDetails(gooey *gocui.Gui) error {
	// find minY, which will be the bottom of the header view
	_, _, _, minY, err := gooey.ViewPosition("title")
	if err != nil {
		log.Fatal(err)
	}

	// find minX, which will be the right edge of the sidebar view
	_, _, minX, _, err := gooey.ViewPosition("sidebar")
	if err != nil {
		log.Fatal(err)
	}

	// create a view to hold the results header and print it
	if detHead, err := gooey.SetView("detHead", minX, minY, menu.maxX-1, minY+2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		fmt.Fprintln(detHead, centerText("Detailed Results", menu.maxX-minX))
	}

	// create the details view
	if details, err := gooey.SetView("details", minX, minY+2, menu.maxX-1, menu.maxY-3); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		// store a pointer to the view for quick access
		menu.details = details
		fmt.Fprintln(details, "Detailed results will be available once a scan has completed.")
	}

	return nil
}

// drawFailures draws the failures view
func drawFailures(gooey *gocui.Gui) error {
	// find minY, which will be the bottom of the header view
	_, _, _, minY, err := gooey.ViewPosition("title")
	if err != nil {
		log.Fatal(err)
	}

	// find minX, which will be the right edge of the sidebar view
	_, _, minX, _, err := gooey.ViewPosition("sidebar")
	if err != nil {
		log.Fatal(err)
	}

	// create a view to hold the failures header and print it
	if failHead, err := gooey.SetView("failHead", minX, minY, menu.maxX-1, minY+2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		fmt.Fprintln(failHead, centerText("Failures", menu.maxX-minX))
	}

	// create the failures view
	if failures, err := gooey.SetView("failures", minX, minY+2, menu.maxX-1, menu.maxY-3); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		// store a pointer to the view for quick access
		menu.failures = failures
		fmt.Fprintln(failures, "Failures will be available once a scan has completed.")
	}

	return nil
}

// drawSuccesses draws the successes view
func drawSuccesses(gooey *gocui.Gui) error {
	// find minY, which will be the bottom of the header view
	_, _, _, minY, err := gooey.ViewPosition("title")
	if err != nil {
		log.Fatal(err)
	}

	// find minX, which will be the right edge of the sidebar view
	_, _, minX, _, err := gooey.ViewPosition("sidebar")
	if err != nil {
		log.Fatal(err)
	}

	// create a view to hold the successes header and print it
	if sucHead, err := gooey.SetView("sucHead", minX, minY, menu.maxX-1, minY+2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		fmt.Fprintln(sucHead, centerText("Successes", menu.maxX-minX))
	}

	// create the successes view
	if successes, err := gooey.SetView("successes", minX, minY+2, menu.maxX-1, menu.maxY-3); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		// store a pointer to the view for quick access
		menu.successes = successes
		fmt.Fprintln(successes, "Successes will be available once a scan has completed.")
	}

	return nil
}

// drawLegend draws the legend view
func drawLegend(gooey *gocui.Gui) error {
	// create the legend view at the bottom of the screen
	if legend, err := gooey.SetView("legend", 0, menu.maxY-3, menu.maxX, menu.maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		fmt.Fprintln(legend, centerText("← ↑ → ↓: Move | ^F: Find Domains | ^S: Scan Domains | ^A: Scan All | ^C: Exit", menu.maxX))
	}

	return nil
}

// uiLayout (re-)draws all of the UI's views and sets the sidebar's selected view on top
func uiLayout(gooey *gocui.Gui) error {
	// bind and set gui dimensions
	menu.maxX, menu.maxY = gooey.Size()

	// draw the views in the menu
	if err := drawTitle(gooey); err != nil {
		return err
	}
	if err := drawSidebar(gooey); err != nil {
		return err
	}
	if err := drawDomains(gooey); err != nil {
		return err
	}
	if err := drawSummary(gooey); err != nil {
		return err
	}
	if err := drawDetails(gooey); err != nil {
		return err
	}
	if err := drawFailures(gooey); err != nil {
		return err
	}
	if err := drawSuccesses(gooey); err != nil {
		return err
	}
	if err := drawLegend(gooey); err != nil {
		return err
	}

	// if no views are selected, set sidebar as active
	if view := gooey.CurrentView(); view == nil {
		if _, err := gooey.SetCurrentView("sidebar"); err != nil {
			return err
		}
	}

	// read the selected menu item and put the corresponding views on top
	if sidebar, err := gooey.View("sidebar"); err == nil {
		selection := readSel(sidebar)
		switch selection {

		case "Domains":
			gooey.Cursor = true
			if _, err = gooey.SetViewOnTop("domHead"); err != nil {
				return err
			}
			if _, err = gooey.SetViewOnTop("domains"); err != nil {
				return err
			}

		case "Summary":
			gooey.Cursor = false
			if _, err = gooey.SetViewOnTop("sumHead"); err != nil {
				return err
			}
			if _, err = gooey.SetViewOnTop("summary"); err != nil {
				return err
			}

		case "Details":
			gooey.Cursor = false
			if _, err = gooey.SetViewOnTop("detHead"); err != nil {
				return err
			}
			if _, err = gooey.SetViewOnTop("details"); err != nil {
				return err
			}

		case "Failures":
			gooey.Cursor = false
			if _, err = gooey.SetViewOnTop("failHead"); err != nil {
				return err
			}
			if _, err = gooey.SetViewOnTop("failures"); err != nil {
				return err
			}

		case "Successes":
			gooey.Cursor = false
			if _, err = gooey.SetViewOnTop("sucHead"); err != nil {
				return err
			}
			if _, err = gooey.SetViewOnTop("successes"); err != nil {
				return err
			}
		}
	}

	return nil
}

const detailsTemp = `
{{- "["}}{{.Score | printf "%4.1f/10.0"}}]
{{- " "}}{{if .Result.Response.Code}}[{{.Result.Response.Code}}{{else}}---{{end}}]
{{- " "}}{{if .Result.Request.IP }}[{{printf "%s" .Result.Request.IP }}]{{end}}
{{- " "}}{{if not .Result.Error }}[{{.Result.Time.Milli}} ms]{{end}}
{{- " "}}{{ .Result.Request.URL }}
{{- if .Result.Assets}}{{" ["}}{{printf "%d" (len .Result.Assets)}} assets]{{end}}`

const successTemp = `
{{- if gt .Score 5.0 }}
{{- "["}}{{.Score | printf "%4.1f/10.0"}}]
{{- " "}}{{if .Result.Response.Code}}[{{.Result.Response.Code}}{{else}}---{{end}}]
{{- " "}}{{if .Result.Request.IP }}[{{printf "%s" .Result.Request.IP }}]{{end}}
{{- " "}}{{ .Result.Request.URL }}
{{- if .Result.Assets}}{{" ["}}{{printf "%d" (len .Result.Assets)}} assets]{{end}}
{{- end}}`

const failTemp = `
{{- if lt .Score 5.0 }}
{{- "["}}{{.Score | printf "%4.1f/10.0"}}]
{{- " "}}{{if .Result.Response.Code}}[{{.Result.Response.Code}}{{else}}---{{end}}]
{{- " "}}{{if .Result.Request.IP }}[{{printf "%s" .Result.Request.IP }}]{{end}}
{{- " "}}{{ .Result.Request.URL }}
{{- if .Result.Error }} errors: {{ .Result.Error }}{{end}}
{{- end}}`

// uiPrintResults parses scan results in to a human-readable format
func uiPrintResults(gooey *gocui.Gui) {
	// clear the domains view and print/re-print the scanned domains
	menu.domains.Clear()
	for _, result := range menu.scan.results {
		fmt.Fprint(menu.domains, result.Result.Request.IP, " ", result.Result.Request.URL, "\n")
	}

	// clear the details view and print detailed scan results to it
	menu.details.Clear()
	detTmpl := template.Must(template.New("details").Parse(detailsTemp + "\n"))
	for _, result := range menu.scan.results {
		detTmpl.Execute(menu.details, result)
	}

	// clear the failures view and print just the failed scans, plus errors
	menu.failures.Clear()
	failTmpl := template.Must(template.New("failures").Parse(failTemp + "\n"))
	for _, result := range menu.scan.results {
		failTmpl.Execute(menu.failures, result)
	}

	// clear the successes view and print just the successful scans
	menu.successes.Clear()
	successTmpl := template.Must(template.New("successes").Parse(successTemp + "\n"))
	for _, result := range menu.scan.results {
		successTmpl.Execute(menu.successes, result)
	}

	// manually trigger the layout function to refresh view contents
	gooey.Execute(uiLayout)
}

// uiFindDomains scans the server for all currently configured domains and prints them to the domains view
func uiFindDomains(gooey *gocui.Gui, view *gocui.View) error {
	// clear the summary and domains views, as we'll be printing to them shortly
	menu.summary.Clear()
	menu.domains.Clear()

	// check the currently running web server
	if err := menu.scan.finder.GetWebservers(); err != nil {
		return err
	}

	// check number of web server processes and print to summary
	if outlist := ""; len(menu.scan.finder.Procs) > 0 {
		for _, proc := range menu.scan.finder.Procs {
			outlist += fmt.Sprintf("[%s:%s] ", proc.Name, proc.PID)
		}
		out.Printf("Found %d procs matching a webserver", len(menu.scan.finder.Procs))
		gooey.Execute(uiLayout)
	}

	// crawl the server for all configured domains
	if err := menu.scan.finder.GetDomains(); err != nil {
		return err
	}

	// print the number of domains to the summary
	out.Printf("Found %d domains on webserver %s (exe: %s, pid: %s)", len(menu.scan.finder.Domains), menu.scan.finder.MainProc.Name, menu.scan.finder.MainProc.Exe, menu.scan.finder.MainProc.PID)

	// print the collected IP's and domains to the domains view
	for _, domain := range menu.scan.finder.Domains {
		fmt.Fprintln(menu.domains, domain.IP, domain.URL)
	}

	gooey.Execute(uiLayout)

	return nil
}

// uiReadDomains scrapes IP's/domains (hosts file format) from the domains view and loads them for the scraper
func uiReadDomains(gooey *gocui.Gui, view *gocui.View) error {
	var doms []*scraper.Domain

	// Read the domains view to a string.  Split each line of that string into a string slice.
	hosts := menu.domains.Buffer()
	for _, host := range strings.Split(hosts, "\n") {

		// trim any preceding or trailing whitespace and split each field in to separate entries
		host = strings.TrimSpace(host)
		entries := strings.Fields(host)

		// if there are less than two entries, skip it
		if len(entries) >= 2 {

			// TODO: regex to verify that's an IP
			// first entry *should* be the IP
			ip := entries[0]

			// run through the rest of the entries and tie them to the IP
			for i, entry := range entries {
				if i != 0 {

					// TODO: regex to verify it's a real domain
					// TODO: ports?
					uri, err := utils.IsDomainURL(entry, "")
					if err != nil {
						return err
					}

					// add each ip/domain to the scaper
					dom := scraper.Domain{IP: ip, URL: uri}
					doms = append(doms, &dom)
				}
			}
		}
	}

	// set scraper domains in the scanner
	menu.scan.crawler.Cnf.Domains = doms
	return nil
}

// uiCrawl pulls the list of domains from the domains view, crawls them, tests them, and outputs test results
func uiCrawl(gooey *gocui.Gui, view *gocui.View) error {
	if err := uiReadDomains(gooey, view); err != nil {
		return err
	}

	// ensure configurations are loaded
	menu.scan.finder.Filter(domfinder.DomainFilter{
		IgnoreHTTP:  conf.scan.ignoreHTTP,
		IgnoreHTTPS: conf.scan.ignoreHTTPS,
		IgnoreMatch: conf.scan.ignoreMatch,
		MatchOnly:   conf.scan.matchOnly,
	})

	// Start the crawl in a goroutine so it doesn't cause the UI to hang.  This also allows
	// sending periodic updates, for those long-running tests.
	go func() {

		// start the crawl
		out.Printf("Starting scan on %d domains...", len(menu.scan.crawler.Cnf.Domains))
		gooey.Execute(uiLayout)
		menu.scan.crawler.Crawl()

		// "periodic updates"
		out.Println("Scan complete.")
		out.Println("Starting tests...")
		gooey.Execute(uiLayout)

		// run the tests
		menu.scan.results = checkTests(menu.scan.crawler.Results, menu.scan.tests)

		// increment the failed or success count
		for _, result := range menu.scan.results {
			if result.Result.Error != nil {
				menu.scan.failed++
			} else {
				menu.scan.successful++
			}
		}

		// print the full results/summary
		out.Printf("%d successful, %d failed", menu.scan.successful, menu.scan.failed)
		uiPrintResults(gooey)

		// reset results and success/fail counters
		menu.scan.crawler.Results = nil
		menu.scan.successful = 0
		menu.scan.failed = 0

	}()

	return nil
}

// scanAll just uiFindDomains and uiCrawl to scan a whole server
func scanAll(gooey *gocui.Gui, view *gocui.View) error {
	if err := uiFindDomains(gooey, view); err != nil {
		return err
	}
	if err := uiCrawl(gooey, view); err != nil {
		return err
	}

	return nil
}

// quit closes out the main UI event loop
func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

// selUp moves the cursor/selection up one line
func selUp(gooey *gocui.Gui, view *gocui.View) error {
	if view != nil {
		view.MoveCursor(0, -1, false)
	}

	return nil
}

// selDown moves the selected menu item down one line, without moving past the last line
func selDown(gooey *gocui.Gui, view *gocui.View) error {
	if view != nil {
		view.MoveCursor(0, 1, false)

		// if the cursor moves to an empty line, move it back
		if readSel(view) == "" {
			view.MoveCursor(0, -1, false)
		}
	}

	return nil
}

// setKeyBinds is a necessary evil
func setKeyBinds(gooey *gocui.Gui) error {
	// press ctrl+c at any time to bail
	if err := gooey.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}

	// press ctrl+a at any time to run a full server scan
	if err := gooey.SetKeybinding("", gocui.KeyCtrlA, gocui.ModNone, scanAll); err != nil {
		return err
	}

	// press ctrl+f at any time to find all domains on the server and populate the domains view
	if err := gooey.SetKeybinding("", gocui.KeyCtrlF, gocui.ModNone, uiFindDomains); err != nil {
		return err
	}

	// press ctrl+s at any time to start/restart a scan
	if err := gooey.SetKeybinding("", gocui.KeyCtrlS, gocui.ModNone, uiCrawl); err != nil {
		return err
	}

	// if the sidebar is active and ↑ is pressed, move the selection up one
	if err := gooey.SetKeybinding("sidebar", gocui.KeyArrowUp, gocui.ModNone, selUp); err != nil {
		return err
	}

	// If the sidebar is active and ↓ is pressed, move the selection down one
	if err := gooey.SetKeybinding("sidebar", gocui.KeyArrowDown, gocui.ModNone, selDown); err != nil {
		return err
	}

	return nil
}

func uiInit() error {
	gooey := gocui.NewGui()
	if err := gooey.Init(); err != nil {
		return err
	}
	defer gooey.Close()

	menu.scan = &Scan{}
	menu.scan.tests = genTests()
	menu.scan.finder = &domfinder.Finder{Log: logger}
	menu.scan.crawler = &scraper.Crawler{Log: logger}

	gooey.SetLayout(uiLayout)
	if err := setKeyBinds(gooey); err != nil {
		return err
	}

	gooey.SelBgColor = gocui.ColorGreen
	gooey.SelFgColor = gocui.ColorBlack
	gooey.Mouse = true

	initOut(ioutil.Discard)
	conf.out.noColors = true

	if err := gooey.MainLoop(); err != nil && err != gocui.ErrQuit {
		return err
	}

	return nil
}
