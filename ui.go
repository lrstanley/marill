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

func uiLayout(g *ui.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("side", -1, -1, 15, maxY); err != nil {
		if err != ui.ErrUnknownView {
			return err
		}

		v.Highlight = true
		fmt.Fprintln(v, " ")
		fmt.Fprintln(v, "        Status ")
		fmt.Fprintln(v, "       Results ")
		fmt.Fprintln(v, "    Successful ")
		fmt.Fprint(v, "        Failed ")
		v.MoveCursor(1, 1, true)
	}
	if v, err := g.SetView("main", 15, -1, maxX, maxY); err != nil {
		if err != ui.ErrUnknownView {
			return err
		}

		v.Wrap = true
		fmt.Fprint(v, "This is a test")
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
	main, err := g.View("main")
	if err != nil {
		return err
	}

	main.Clear()

	for i := 0; i < len(out.buffer); i++ {
		fmt.Fprint(main, out.buffer[i])
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

func showMsg(g *ui.Gui, text string) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("msg", maxX/2-(len(text)/2)-1, maxY/2, maxX/2+(len(text)/2)+1, maxY/2+2); err != nil {
		if err != ui.ErrUnknownView {
			return err
		}
		fmt.Fprintln(v, text)
	}

	if err := g.SetKeybinding("main", ui.MouseLeft, ui.ModNone, delMsg); err != nil {
		return err
	}

	if err := g.SetCurrentView("msg"); err != nil {
		return err
	}

	return nil
}

func delMsg(g *ui.Gui, v *ui.View) error {
	g.DeleteKeybinding("main", ui.MouseLeft, ui.ModNone)
	if err := g.DeleteView("msg"); err != nil {
		return err
	}
	return nil
}

func uiInit() error {
	gui := ui.NewGui()
	if err := gui.Init(); err != nil {
		return err
	}
	defer gui.Close()

	gui.SetLayout(uiLayout)
	if err := uiKeybindings(gui); err != nil {
		return err
	}

	gui.SelBgColor = ui.ColorGreen
	gui.SelFgColor = ui.ColorBlack
	gui.Mouse = true

	initOut(ioutil.Discard)
	conf.out.noColors = true
	go func() {
		for {
			time.Sleep(1 * time.Second)

			gui.Execute(uiUpdateLog)
		}
	}()

	go func() {
		time.Sleep(3 * time.Second)
		showMsg(gui, "this is a test")
	}()

	// go run()

	if err := gui.MainLoop(); err != nil && err != ui.ErrQuit {
		return err
	}

	return nil
}
