package scraper

import (
	"fmt"
	"net/url"
	"time"
)

// Timer represents a custom timer, holding start/end nanoseconds
type Timer struct {
	startTime int64
	endTime   int64
	Result    *TimerResult
}

// TimerResult is a result of an ended timer, providing easy millisecond
// and second calculations to the process time
type TimerResult struct {
	Milli   int64
	Seconds int64
}

// Start starts a timer and returns a Timer struct
func (t *Timer) Start() {
	t.startTime = time.Now().UnixNano()
}

// End completes a timer and calculates the differences
func (t *Timer) End() {
	t.endTime = time.Now().UnixNano()

	t.Result = &TimerResult{}

	t.Result.Milli = (t.endTime - t.startTime) / 1000000
	t.Result.Seconds = t.Result.Milli / 1000

	return
}

// NewTimer returns a new Timer struct
func NewTimer() *Timer {
	timer := &Timer{}
	timer.Start()

	return timer
}

func getHost(uri string) (string, error) {
	host, err := url.Parse(uri)

	if err != nil {
		return "", fmt.Errorf("invalid uri: %s", uri)
	}

	return host.Host, nil
}

// stripDups strips all domains that have the same resulting URL/IP
func stripDups(domains *[]*Domain) {
	var tmp []*Domain

	for _, dom := range *domains {
		isIn := false
		for _, other := range tmp {
			if dom.URL.String() == other.URL.String() && dom.IP == other.IP {
				isIn = true
				break
			}
		}
		if !isIn {
			tmp = append(tmp, dom)
		}
	}

	*domains = tmp

	return
}
