// Author: Liam Stanley <me@liamstanley.io>
// Docs: https://marill.liam.sh/
// Repo: https://github.com/Liamraystanley/marill

package main

import (
	"fmt"
	"io/ioutil"
	"time"

	ui "github.com/jroimartin/gocui"
)

var keybindText = [...]string{
	"KEYBINDINGS",
	"Space: New View",
	"Tab: Next View",
	"← ↑ → ↓: Move View",
	"Backspace: Delete View",
	"t: Set view on top",
	"^C: Exit",
}

type guiCnf struct {
	currentView string
}

type Gui struct {
	g   *ui.Gui
	cnf guiCnf

	// views and scopes
	main *ui.View
	side *ui.View
}

var gui = Gui{}

func uiLayout(g *ui.Gui) error {
	var err error
	maxX, maxY := g.Size()
	if gui.side, err = g.SetView("side", -1, -1, 15, maxY); err != nil {
		if err != ui.ErrUnknownView {
			return err
		}

		gui.side.Highlight = true
		fmt.Fprintln(gui.side, " ")
		fmt.Fprintln(gui.side, "        Status ")
		fmt.Fprintln(gui.side, "       Results ")
		fmt.Fprintln(gui.side, "    Successful ")
		fmt.Fprint(gui.side, "        Failed ")
		gui.side.MoveCursor(1, 1, true)
	}
	if gui.main, err = g.SetView("main", 15, -1, maxX, maxY); err != nil {
		if err != ui.ErrUnknownView {
			return err
		}

		fmt.Fprint(gui.main, "This is a test")
		gui.main.Wrap = true
		gui.main.Autoscroll = true
		if err := g.SetCurrentView("main"); err != nil {
			return err
		}
	}
	if v, err := g.SetView("legend", maxX-25, 0, maxX-1, 8); err != nil {
		if err != ui.ErrUnknownView {
			return err
		}
		for i := 0; i < len(keybindText); i++ {
			fmt.Fprintln(v, keybindText[i])
		}
	}
	return nil
}

func uiUpdateLog(g *ui.Gui) error {
	gui.main.Clear()

	for i := 0; i < len(out.buffer); i++ {
		fmt.Fprint(gui.main, out.buffer[i])
	}

	return nil
}

func uiKeybindings(g *ui.Gui) error {
	if err := g.SetKeybinding("", ui.KeyCtrlC, ui.ModNone, uiQuit); err != nil {
		return err
	}

	return nil
}

func uiQuit(g *ui.Gui, v *ui.View) error {
	return ui.ErrQuit
}

func uiInit() error {
	gui.g = ui.NewGui()
	if err := gui.g.Init(); err != nil {
		return err
	}
	defer gui.g.Close()

	gui.g.SetLayout(uiLayout)
	if err := uiKeybindings(gui.g); err != nil {
		return err
	}

	gui.g.SelBgColor = ui.ColorGreen
	gui.g.SelFgColor = ui.ColorBlack
	gui.g.Mouse = true

	initOut(ioutil.Discard)
	conf.out.noColors = true
	go func() {
		for {
			time.Sleep(1 * time.Second)

			gui.g.Execute(uiUpdateLog)
		}
	}()

	go run()

	if err := gui.g.MainLoop(); err != nil && err != ui.ErrQuit {
		return err
	}

	return nil
}
