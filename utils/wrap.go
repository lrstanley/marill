package utils

import "io"

// Wrap implements the io.Closer, io.Reader, and io.Writer interface
type ioWrapper struct {
	readHandler  func([]byte)
	writeHandler func(*[]byte)
	r            io.Reader
	w            io.Writer
}

// Read implements the io.Reader interface
func (w *ioWrapper) Read(p []byte) (int, error) {
	n, err := w.r.Read(p)
	if n > 0 {
		w.readHandler(p[:n])
	}
	return n, err
}

// Write implements the io.Writer interface
func (w *ioWrapper) Write(p []byte) (int, error) {
	w.writeHandler(&p)
	return w.w.Write(p)
}

// NewFuncReader returns an io.Reader that wraps the given io.Reader with the given handler.
// If any of the parameters are nil, nil is returned.
func NewFuncReader(handler func([]byte), r io.Reader) io.Reader {
	if handler == nil || r == nil {
		return nil
	}
	return &ioWrapper{readHandler: handler, r: r}
}

// NewFuncWriter returns an io.Writer that wraps the given io.Writer with the given handler.
// Any Write() operations will run through the handler before being written. If any of the
// parameters are nil, nil is returned.
func NewFuncWriter(handler func(*[]byte), w io.Writer) io.Writer {
	if handler == nil || w == nil {
		return nil
	}
	return &ioWrapper{writeHandler: handler, w: w}
}
