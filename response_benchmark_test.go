package req

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"testing"
)

var benchmarkResponseBody []byte

func BenchmarkResponseToBytes(b *testing.B) {
	for _, size := range []int{1024, 8 * 1024, 64 * 1024} {
		payload := bytes.Repeat([]byte("x"), size)
		b.Run(byteSizeName(size), func(b *testing.B) {
			request := C().R()
			b.ReportAllocs()
			b.SetBytes(int64(size))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				resp := &Response{
					Request: request,
					Response: &http.Response{
						Body:          io.NopCloser(bytes.NewReader(payload)),
						ContentLength: int64(len(payload)),
					},
				}
				body, err := resp.ToBytes()
				if err != nil {
					b.Fatal(err)
				}
				benchmarkResponseBody = body
			}
		})
	}
}

func byteSizeName(size int) string {
	if size >= 1024 {
		return strconv.Itoa(size/1024) + "KiB"
	}
	return "small"
}
