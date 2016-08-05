package scraper

import "time"

// Timer represents a custom timer, holding start/end nanoseconds
type Timer struct {
	startTime int64
	endTime   int64
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
func (t *Timer) End() *TimerResult {
	t.endTime = time.Now().UnixNano()

	result := &TimerResult{}

	result.Milli = (t.endTime - t.startTime) / 1000000
	result.Seconds = result.Milli / 1000

	return result
}

// NewTimer returns a new Timer struct
func NewTimer() *Timer {
	timer := &Timer{}
	timer.Start()

	return timer
}
