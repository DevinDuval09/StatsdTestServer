package main

import (
	"fmt"
	"slices"
	"testing"
)

func TestParseLine(t *testing.T) {
	var tests = []struct {
		In   string
		Want []string
	}{
		{"counter:1|c", []string{"counter", "1", "c"}},
		{"nosample_gauge:0.55|g", []string{"nosample_gauge", "0.55", "g"}},
		{"sample_gauge:.66|g|@2.0", []string{"sample_gauge", ".66", "g", "2.0"}},
		{"sample_timer_tag:156|ms|@4.0|tag", []string{"sample_timer_tag", "156", "ms", "4.0", "tag"}},
	}
	for _, test := range tests {
		testname := fmt.Sprintf("Input: %s", test.In)
		t.Run(testname, func(*testing.T) {
			actual := ParseLine(test.In)
			if !slices.Equal(actual, test.Want) {
				t.Errorf("got: %s, want: %s", actual, test.Want)
			}
		})
	}

}
