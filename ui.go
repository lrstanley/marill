// Author: Liam Stanley <me@liamstanley.io>
// Docs: https://marill.liam.sh/
// Repo: https://github.com/Liamraystanley/marill

package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"strings"

	"github.com/jroimartin/gocui"
)

// mMenu holds some
type mMenu struct {
	maxX, maxY int
}

var menu mMenu

// centerText takes a string of text and a length and pads the beginning
// of the string with spaces to center that text in the available space.
func centerText(text string, maxX int) string {

	numSpaces := maxX/2 - len(text)/2
	for i := 1; i < numSpaces; i++ {
		text = " " + text
	}

	return text
}

// padText takes a string of text and pads the end of it with spaces to
// fill the available space in a cell.
func padText(text string, maxX int) string {

	numSpaces := maxX - len(text)
	for i := 0; i < numSpaces; i++ {
		text += " "
	}

	return text
}

// readSel reads the currently selected line and returns a string
// containing its contents, without trailing spaces.
func readSel(view *gocui.View) string {

	_, posY := view.Cursor()
	selection, _ := view.Line(posY)
	selection = strings.TrimSpace(selection)

	return selection
}

// drawTitle adds the title to the top of the menu.
func drawTitle(gooey *gocui.Gui) error {

	// Place the title view at the top of the menu and extend it down two lines.
	if title, err := gooey.SetView("title", 0, 0, menu.maxX-1, 2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		// Center the title text and print it to the view.
		fmt.Fprintln(title, centerText("Marill: Automated site testing utility", menu.maxX))

	}

	return nil
}

// drawSidebar draws the sidebar on the left side of the gui.
func drawSidebar(gooey *gocui.Gui) error {

	// Find minY, which will be the bottom of the header view.
	_, _, _, minY, err := gooey.ViewPosition("title")
	if err != nil {
		log.Fatal(err)
	}

	// Set maxX, which will be fill one sixth of the menu.
	maxX := menu.maxX / 6

	// Create a view to hold the sidebar header.
	if sideHead, err := gooey.SetView("sideHead", 0, minY, maxX, minY+2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		// Print the title to the sidebar header.
		fmt.Fprintln(sideHead, centerText("Abilities", maxX))

	}

	// Create a view for the sidebar itself.
	if sidebar, err := gooey.SetView("sidebar", 0, minY+2, maxX, menu.maxY-3); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		// Print options and ensure highlights are enabled.
		fmt.Fprintln(sidebar, padText("Domains", maxX))
		fmt.Fprintln(sidebar, padText("Summary", maxX))
		fmt.Fprintln(sidebar, padText("Details", maxX))
		fmt.Fprintln(sidebar, padText("Failures", maxX))
		fmt.Fprintln(sidebar, padText("Successes", maxX))
		sidebar.Highlight = true

	}

	return nil
}

// drawDomains draws the domains view.
func drawDomains(gooey *gocui.Gui) error {

	// Find minY, which will be the bottom of the header view.
	_, _, _, minY, err := gooey.ViewPosition("title")
	if err != nil {
		log.Fatal(err)
	}

	// Find minX, which will be the right edge of the sidebar view.
	_, _, minX, _, err := gooey.ViewPosition("sidebar")
	if err != nil {
		log.Fatal(err)
	}

	// Create a view to hold the domains header.
	if domHead, err := gooey.SetView("domHead", minX, minY, menu.maxX-1, minY+2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		// Print the title to the domains header.
		fmt.Fprintln(domHead, centerText("Scan Domains", menu.maxX-minX))

	}

	// Create the domains view.
	if domains, err := gooey.SetView("domains", minX, minY+2, menu.maxX-1, menu.maxY-3); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		fmt.Fprintln(domains, "Marill's UI currently only supports full server scans.")
		fmt.Fprintln(domains, "To start a full scan, press crl+a.")
	}

	return nil
}

// drawSummary draws the summary view.
func drawSummary(gooey *gocui.Gui) error {

	// Find minY, which will be the bottom of the header view.
	_, _, _, minY, err := gooey.ViewPosition("title")
	if err != nil {
		log.Fatal(err)
	}

	// Find minX, which will be the right edge of the sidebar view.
	_, _, minX, _, err := gooey.ViewPosition("sidebar")
	if err != nil {
		log.Fatal(err)
	}

	// Create a view to hold the summary header.
	if sumHead, err := gooey.SetView("sumHead", minX, minY, menu.maxX-1, minY+2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		// Print the title to the summary header.
		fmt.Fprintln(sumHead, centerText("Results Summary", menu.maxX-minX))
	}

	// Create the summary view.
	if summary, err := gooey.SetView("summary", minX, minY+2, menu.maxX-1, menu.maxY-3); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		fmt.Fprintln(summary, "Scan summary will be available once the scan started.")
	}

	return nil
}

// drawDetails draws the details view.
func drawDetails(gooey *gocui.Gui) error {

	// Find minY, which will be the bottom of the header view.
	_, _, _, minY, err := gooey.ViewPosition("title")
	if err != nil {
		log.Fatal(err)
	}

	// Find minX, which will be the right edge of the sidebar view.
	_, _, minX, _, err := gooey.ViewPosition("sidebar")
	if err != nil {
		log.Fatal(err)
	}

	// Create a view to hold the results header.
	if detHead, err := gooey.SetView("detHead", minX, minY, menu.maxX-1, minY+2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		// Print the title to the details header.
		fmt.Fprintln(detHead, centerText("Detailed Results", menu.maxX-minX))
	}

	// Create the details view.
	if details, err := gooey.SetView("details", minX, minY+2, menu.maxX-1, menu.maxY-3); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		fmt.Fprintln(details, "Detailed results will be available once the scan has completed.")
	}

	return nil
}

// drawFailures draws the failures view.
func drawFailures(gooey *gocui.Gui) error {

	// Find minY, which will be the bottom of the header view.
	_, _, _, minY, err := gooey.ViewPosition("title")
	if err != nil {
		log.Fatal(err)
	}

	// Find minX, which will be the right edge of the sidebar view.
	_, _, minX, _, err := gooey.ViewPosition("sidebar")
	if err != nil {
		log.Fatal(err)
	}

	// Create a view to hold the failures header.
	if failHead, err := gooey.SetView("failHead", minX, minY, menu.maxX-1, minY+2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		// Print the title to the failures header.
		fmt.Fprintln(failHead, centerText("Failures", menu.maxX-minX))
	}

	// Create the failures view.
	if failures, err := gooey.SetView("failures", minX, minY+2, menu.maxX-1, menu.maxY-3); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		fmt.Fprintln(failures, "Failures will be available once the scan has completed.")
	}

	return nil
}

// drawSuccesses draws the successes view.
func drawSuccesses(gooey *gocui.Gui) error {

	// Find minY, which will be the bottom of the header view.
	_, _, _, minY, err := gooey.ViewPosition("title")
	if err != nil {
		log.Fatal(err)
	}

	// Find minX, which will be the right edge of the sidebar view.
	_, _, minX, _, err := gooey.ViewPosition("sidebar")
	if err != nil {
		log.Fatal(err)
	}

	// Create a view to hold the sucesses header.
	if sucHead, err := gooey.SetView("sucHead", minX, minY, menu.maxX-1, minY+2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		// Print the title to the successes header.
		fmt.Fprintln(sucHead, centerText("Successes", menu.maxX-minX))
	}

	// Create the successes view.
	if successes, err := gooey.SetView("successes", minX, minY+2, menu.maxX-1, menu.maxY-3); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		fmt.Fprintln(successes, "Successes will be available once the scan has completed.")
	}

	return nil
}

// TODO:  Move Legend to bottom.
var keybindText = [...]string{
	"KEYBINDINGS",
	"Space: New View",
	"Tab: Next View",
	"← ↑ → ↓: Move View",
	"Backspace: Delete View",
	"t: Set view on top",
	"^C: Exit",
}

// drawLegend draws te legend view.
func drawLegend(gooey *gocui.Gui) error {

	// Create the legend view at the bottom of the screen.
	if legend, err := gooey.SetView("legend", 0, menu.maxY-3, menu.maxX, menu.maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		// Print the key bindings to the legend view.
		fmt.Fprintln(legend, centerText("← ↑ → ↓: Move | ^A: Scan All | ^C: Exit", menu.maxX))
	}

	return nil
}

func uiLayout(gooey *gocui.Gui) error {

	// Find and set gui dimensions.
	menu.maxX, menu.maxY = gooey.Size()

	// Draw the views in the menu.
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

	// After the views have been draw, ensure sidebar is selected.
	if _, err := gooey.SetCurrentView("sidebar"); err != nil {
		return err
	}

	// Read the selected menu item and put the corresponding views on top.
	if sidebar, err := gooey.View("sidebar"); err == nil {
		selection := readSel(sidebar)
		switch selection {

		case "Domains":
			if _, err = gooey.SetViewOnTop("domHead"); err != nil {
				return err
			}
			if _, err = gooey.SetViewOnTop("domains"); err != nil {
				return err
			}

		case "Summary":
			if _, err = gooey.SetViewOnTop("sumHead"); err != nil {
				return err
			}
			if _, err = gooey.SetViewOnTop("summary"); err != nil {
				return err
			}

		case "Details":
			if _, err = gooey.SetViewOnTop("detHead"); err != nil {
				return err
			}
			if _, err = gooey.SetViewOnTop("details"); err != nil {
				return err
			}

		case "Failures":
			if _, err = gooey.SetViewOnTop("failHead"); err != nil {
				return err
			}
			if _, err = gooey.SetViewOnTop("failures"); err != nil {
				return err
			}

		case "Successes":
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

func scanAllTheThings(gooey *gocui.Gui, view *gocui.View) error {

	//
	summary, err := gooey.View("summary")
	if err != nil {
		return err
	}
	summary.Clear()

	initOutWriter(summary)

	//
	scan, err := crawl()
	if err != nil {
		return err
	}

	domains, err := gooey.View("domains")
	if err != nil {
		return err
	}
	domains.Clear()

	for _, result := range scan.results {
		fmt.Fprint(domains, "[", result.Result.Request.IP, "] ", result.Result.Request.URL, "\n")
	}

	//
	details, err := gooey.View("details")
	if err != nil {
		return err
	}
	details.Clear()

	//
	detTmpl := template.Must(template.New("details").Parse(detailsTemp + "\n"))
	for _, result := range scan.results {
		detTmpl.Execute(details, result)
	}

	//
	successes, err := gooey.View("successes")
	if err != nil {
		return err
	}
	successes.Clear()

	//
	successTmpl := template.Must(template.New("successes").Parse(successTemp + "\n"))
	for _, result := range scan.results {
		successTmpl.Execute(successes, result)
	}

	//
	failures, err := gooey.View("failures")
	if err != nil {
		return err
	}
	failures.Clear()

	//
	failTmpl := template.Must(template.New("failures").Parse(failTemp + "\n"))
	for _, result := range scan.results {
		failTmpl.Execute(failures, result)
	}

	gooey.Execute(uiLayout)

	return nil
}

// quit quits the main event loop.
func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

// selUp moves the cursor/selection up one line.
func selUp(gooey *gocui.Gui, view *gocui.View) error {

	// Move the cursor up one line.
	if view != nil {
		view.MoveCursor(0, -1, false)
	}

	return nil
}

// selDown moves the selected menu item down one line, without moving past the last line.
func selDown(gooey *gocui.Gui, view *gocui.View) error {

	// Move the cursor down one line.
	if view != nil {
		view.MoveCursor(0, 1, false)

		// If the cursor moves to an empty line, move it back. :P
		if readSel(view) == "" {
			view.MoveCursor(0, -1, false)
		}
	}

	return nil
}

// setKeyBinds is a necessary evil.
func setKeyBinds(gooey *gocui.Gui) error {

	// Always have an exit strategy.
	if err := gooey.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}

	// Start scanning
	if err := gooey.SetKeybinding("", gocui.KeyCtrlA, gocui.ModNone, scanAllTheThings); err != nil {
		return err
	}

	// If the sidebar is active and ↑ is pressed, move the selection up one.
	if err := gooey.SetKeybinding("sidebar", gocui.KeyArrowUp, gocui.ModNone, selUp); err != nil {
		return err
	}

	// If the sidebar is active and ↓ is pressed, move the selection down one.
	if err := gooey.SetKeybinding("sidebar", gocui.KeyArrowDown, gocui.ModNone, selDown); err != nil {
		return err
	}

	return nil
}

func showMsg(g *gocui.Gui, text string) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("msg", maxX/2-(len(text)/2)-1, maxY/2, maxX/2+(len(text)/2)+1, maxY/2+2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		fmt.Fprintln(v, text)
	}

	if err := g.SetKeybinding("main", gocui.MouseLeft, gocui.ModNone, delMsg); err != nil {
		return err
	}

	if _, err := g.SetCurrentView("msg"); err != nil {
		return err
	}

	return nil
}

func delMsg(g *gocui.Gui, v *gocui.View) error {
	g.DeleteKeybinding("main", gocui.MouseLeft, gocui.ModNone)
	if err := g.DeleteView("msg"); err != nil {
		return err
	}
	return nil
}

func uiInit() error {

	gui := gocui.NewGui()
	if err := gui.Init(); err != nil {
		return err
	}
	defer gui.Close()

	gui.SetLayout(uiLayout)
	if err := setKeyBinds(gui); err != nil {
		return err
	}

	gui.SelBgColor = gocui.ColorGreen
	gui.SelFgColor = gocui.ColorBlack
	gui.Mouse = true

	initOut(ioutil.Discard)
	conf.out.noColors = true

	if err := gui.MainLoop(); err != nil && err != gocui.ErrQuit {
		return err
	}

	return nil
}
