package compress

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"
)

// brotliTestBody 记录底层 Body 是否关闭，并模拟关闭失败。
type brotliTestBody struct {
	io.Reader
	closed bool
	err    error
}

// Close 将释放行为和错误暴露给测试断言。
func (b *brotliTestBody) Close() error {
	b.closed = true
	return b.err
}

// TestBrotliReader 验证参考流兼容、分片输入、错误传播与响应体释放。
func TestBrotliReader(t *testing.T) {
	// Node.js zlib (the C reference encoder), quality 11, 98,304 decoded bytes.
	encoded, err := base64.StdEncoding.DecodeString("W/9/gd8igedI7NcI6DvsSzmh8lQwR3fKgVGcUBQn2unw7QkSR5cgaWxxYZxg0bsASPhZf6/1Ew==")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Repeat("req Brotli 兼容测试\n", 4096)
	sourceErr := errors.New("source failed")
	for _, tc := range []struct {
		name    string
		source  io.Reader
		wantErr bool
	}{
		{"reference", bytes.NewReader(encoded), false},
		{"fragmented", iotest.OneByteReader(bytes.NewReader(encoded)), false},
		{"truncated", bytes.NewReader(encoded[:len(encoded)-1]), true},
		{"invalid", strings.NewReader("not brotli"), true},
		{"source error", iotest.ErrReader(sourceErr), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := &brotliTestBody{Reader: tc.source}
			r := NewCompressReader(body, "br")
			got, err := io.ReadAll(r)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ReadAll error = %v, want error = %v", err, tc.wantErr)
			}
			if !tc.wantErr && string(got) != want {
				t.Fatalf("decoded body mismatch: got %d bytes", len(got))
			}
			if tc.name == "source error" && !errors.Is(err, sourceErr) {
				t.Fatalf("source error lost: %v", err)
			}
			if err := r.Close(); err != nil || !body.closed {
				t.Fatalf("body not closed: %v", err)
			}
		})
	}
	for _, readFirst := range []bool{false, true} {
		body := &brotliTestBody{Reader: bytes.NewReader(encoded), err: sourceErr}
		r := NewBrotliReader(io.NopCloser(iotest.ErrReader(errors.New("old body read"))))
		r.SetUnderlyingBody(body)
		if r.GetUnderlyingBody() != body {
			t.Fatal("underlying body was not replaced")
		}
		if readFirst {
			if _, err := r.Read(make([]byte, 1)); err != nil {
				t.Fatal(err)
			}
		}
		if err := r.Close(); !errors.Is(err, sourceErr) || !body.closed {
			t.Fatalf("Close lost body error: %v", err)
		}
	}
}
