package format

import "testing"

func TestFloat(t *testing.T) {
	cases := map[float64]string{
		30.0:  "30.0",
		20.0:  "20.0",
		0.0:   "0.0",
		25.5:  "25.5",
		33.33: "33.33",
		0.1:   "0.1",
		100.0: "100.0",
		15.0:  "15.0",
		35.0:  "35.0",
	}
	for in, want := range cases {
		if got := Float(in); got != want {
			t.Errorf("Float(%v) = %q, want %q", in, got, want)
		}
	}
}
