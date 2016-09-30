// Author: Liam Stanley <me@liamstanley.io>
// Docs: https://marill.liam.sh/
// Repo: https://github.com/Liamraystanley/marill

package utils

// Pool example:
// concurrency := 5
// pool := NewPool(concurrency)
// urls := []string{"url1", "url2"}
// for _, url := range urls {
//     pool.Slot() // wait for an open slot
//     go func(url) {
//         defer pool.Free() // free the slot we're occupying
//
//         // get url or other stuff
//     }(url)
// }
//
// pool.Wait()

// Pool represents a go-routine worker pool. This does NOT manage the
// workers, only how many workers are running.
type Pool struct {
	total   int
	threads chan bool
	done    bool
}

// Slot is used to wait for an open slot to start processing
func (p *Pool) Slot() {
	if p.done {
		panic("Slot() called in go-routine on completed pool")
	}

	p.threads <- true
}

// Free is used to free the slot taken by Pool.Slot()
func (p *Pool) Free() {
	if p.done {
		panic("Free() called in go-routine on completed pool")
	}

	<-p.threads
}

// Wait is used to wait for all open Slot()'s to be Free()'d
func (p *Pool) Wait() {
	if p.done {
		panic("Wait() called on completed pool")
	}

	for i := 0; i < cap(p.threads); i++ {
		p.threads <- true
	}

	p.done = true
}

// NewPool returns a new Pool{} method
func NewPool(count int) Pool {
	if count < 1 {
		count = 1
	}

	return Pool{total: count, threads: make(chan bool, count)}
}
