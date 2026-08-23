package req

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"
)

var errOversizedReadBuffer = errors.New("oversized read buffer")

func TestReadResponseBodyUsesContentLengthAsCapacityHintOnly(t *testing.T) {
	tests := []struct {
		name          string
		body          string
		contentLength int64
	}{
		{name: "zero hint", body: "response body", contentLength: 0},
		{name: "one byte hint", body: "response body", contentLength: 1},
		{name: "exact", body: "response body", contentLength: 13},
		{name: "reported shorter", body: "response body", contentLength: 4},
		{name: "reported longer", body: "response body", contentLength: 64},
		{name: "preallocation boundary", body: string(bytes.Repeat([]byte("x"), maxResponseBodyPreallocateSize)), contentLength: maxResponseBodyPreallocateSize},
		{name: "body grows beyond reserved capacity", body: string(bytes.Repeat([]byte("x"), maxResponseBodyPreallocateSize+bytes.MinRead+1)), contentLength: maxResponseBodyPreallocateSize},
		{name: "hint exceeds boundary", body: string(bytes.Repeat([]byte("x"), maxResponseBodyPreallocateSize+1)), contentLength: maxResponseBodyPreallocateSize + 1},
		{name: "unknown", body: "response body", contentLength: -1},
		{name: "untrusted huge length", body: "response body", contentLength: 1 << 40},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readResponseBody(bytes.NewBufferString(tt.body), tt.contentLength)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.body {
				t.Fatalf("readResponseBody() = %q, want %q", got, tt.body)
			}
		})
	}
}

func TestReadResponseBodyReturnsPartialDataWithReadError(t *testing.T) {
	sentinel := errors.New("sentinel read error")
	for _, contentLength := range []int64{1, maxResponseBodyPreallocateSize + 1} {
		t.Run(byteSizeName(int(contentLength)), func(t *testing.T) {
			got, err := readResponseBody(&partialErrorReader{data: []byte("partial"), err: sentinel}, contentLength)
			if string(got) != "partial" {
				t.Fatalf("readResponseBody() = %q, want partial data", got)
			}
			if !errors.Is(err, sentinel) {
				t.Fatalf("readResponseBody() error = %v, want %v", err, sentinel)
			}
		})
	}
}

func TestResponseToBytesHeadIgnoresAdvertisedLengthHint(t *testing.T) {
	transformerCalled := false
	client := C().SetResponseBodyTransformer(func(rawBody []byte, _ *Request, _ *Response) ([]byte, error) {
		transformerCalled = true
		return append(rawBody, "transformed"...), nil
	})
	request := client.R()
	request.Method = http.MethodHead
	reader := &boundedReadCloser{maxReadSize: 1024}
	response := &Response{
		Request: request,
		Response: &http.Response{
			Body:          reader,
			ContentLength: maxResponseBodyPreallocateSize,
		},
	}

	got, err := response.ToBytes()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "transformed" {
		t.Fatalf("Response.ToBytes() = %q, want transformed body", got)
	}
	if !transformerCalled {
		t.Fatal("response body transformer was not called")
	}
	if !reader.closed {
		t.Fatal("response body was not closed")
	}
}

type partialErrorReader struct {
	data []byte
	err  error
}

func (r *partialErrorReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, r.err
}

type boundedReadCloser struct {
	maxReadSize int
	closed      bool
}

func (r *boundedReadCloser) Read(p []byte) (int, error) {
	if len(p) > r.maxReadSize {
		return 0, errOversizedReadBuffer
	}
	return 0, io.EOF
}

func (r *boundedReadCloser) Close() error {
	r.closed = true
	return nil
}
