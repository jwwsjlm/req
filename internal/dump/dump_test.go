package dump

import (
	"bytes"
	"io"
	"testing"
	"time"
)

type testDumpOptions struct {
	output io.Writer
	async  bool
}

func (o testDumpOptions) Output() io.Writer {
	return o.output
}

func (o testDumpOptions) RequestHeaderOutput() io.Writer {
	return o.output
}

func (o testDumpOptions) RequestBodyOutput() io.Writer {
	return o.output
}

func (o testDumpOptions) ResponseHeaderOutput() io.Writer {
	return o.output
}

func (o testDumpOptions) ResponseBodyOutput() io.Writer {
	return o.output
}

func (o testDumpOptions) RequestHeader() bool {
	return true
}

func (o testDumpOptions) RequestBody() bool {
	return true
}

func (o testDumpOptions) ResponseHeader() bool {
	return true
}

func (o testDumpOptions) ResponseBody() bool {
	return true
}

func (o testDumpOptions) Async() bool {
	return o.async
}

func (o testDumpOptions) Clone() Options {
	return o
}

func TestDumperStopUnblocksAsyncDump(t *testing.T) {
	var buf bytes.Buffer
	d := NewDumper(testDumpOptions{
		output: &buf,
		async:  true,
	})
	d.Stop()
	d.Stop()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < cap(d.ch)+1; i++ {
			d.DumpDefault([]byte("x"))
		}
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("async dump blocked after Stop")
	}
}
