package http2

import "testing"

func TestTransportInitialStreamID(t *testing.T) {
	tests := []struct {
		name string
		in   uint32
		want uint32
	}{
		{name: "default", in: 0, want: 1},
		{name: "odd", in: 3, want: 3},
		{name: "even", in: 2, want: 3},
		{name: "too large", in: 1 << 31, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Transport{InitialStreamID: tt.in}
			if got := tr.initialStreamID(); got != tt.want {
				t.Fatalf("initialStreamID() = %d, want %d", got, tt.want)
			}
		})
	}
}
