package req

import (
	"net/url"
	"reflect"
	"testing"
)

func TestEncodeQueryParams(t *testing.T) {
	tests := []struct {
		name    string
		client  url.Values
		request url.Values
		want    string
	}{
		{
			name:   "client only",
			client: url.Values{"a": {"1"}, "tag": {"go", "http"}},
			want:   "a=1&tag=go&tag=http",
		},
		{
			name:    "request only",
			request: url.Values{"b": {"2"}},
			want:    "b=2",
		},
		{
			name:    "request overrides client",
			client:  url.Values{"a": {"client"}, "keep": {"1"}},
			request: url.Values{"a": {"request", "second"}},
			want:    "a=request&a=second&keep=1",
		},
		{
			name:    "empty request value removes client value",
			client:  url.Values{"a": {"client"}, "keep": {"1"}},
			request: url.Values{"a": nil},
			want:    "keep=1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientBefore := cloneURLValues(tt.client)
			requestBefore := cloneURLValues(tt.request)
			if got := encodeQueryParams(tt.client, tt.request); got != tt.want {
				t.Fatalf("encodeQueryParams() = %q, want %q", got, tt.want)
			}
			if !reflect.DeepEqual(tt.client, clientBefore) {
				t.Fatalf("client params mutated: got %#v, want %#v", tt.client, clientBefore)
			}
			if !reflect.DeepEqual(tt.request, requestBefore) {
				t.Fatalf("request params mutated: got %#v, want %#v", tt.request, requestBefore)
			}
		})
	}
}

func cloneURLValues(values url.Values) url.Values {
	if values == nil {
		return nil
	}
	clone := make(url.Values, len(values))
	for key, value := range values {
		clone[key] = append([]string(nil), value...)
	}
	return clone
}
