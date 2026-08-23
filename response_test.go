package req

import (
	"bytes"
	"testing"
)

func TestReadResponseBodyUsesContentLengthAsCapacityHintOnly(t *testing.T) {
	tests := []struct {
		name          string
		body          string
		contentLength int64
	}{
		{name: "exact", body: "response body", contentLength: 13},
		{name: "reported shorter", body: "response body", contentLength: 4},
		{name: "reported longer", body: "response body", contentLength: 64},
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
