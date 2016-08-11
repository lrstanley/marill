package configfinder

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var webservers = map[string]bool{
	"httpd":   true,
	"apache":  true,
	"nginx":   true,
	"lshttpd": true,
}

type Process struct {
	PID  string
	Name string
	Exe  string
}

func GetProcs() (pl []*Process) {
	ps, _ := filepath.Glob("/proc/[0-9]*")

	for i := range ps {
		proc := &Process{}

		proc.PID = strings.Split(ps[i], "/")[2]

		// command name
		if data, err := ioutil.ReadFile(ps[i] + "/comm"); err != nil {
			continue
		} else {
			proc.Name = strings.Replace(string(data), "\n", "", 1)
		}

		if !webservers[proc.Name] {
			continue
		}

		// executable path
		if data, err := os.Readlink(ps[i] + "/exe"); err != nil {
			continue
		} else {
			proc.Exe = strings.Replace(string(data), "\n", "", 1)
		}

		pl = append(pl, proc)
	}

	return pl
}
