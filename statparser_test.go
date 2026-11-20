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

func TestSplitStat(t *testing.T) {
	var tests = []struct {
		In           string
		WantStatName string
		WantValue    float64
		WantStatType string
	}{
		{In: "counter:1|c", WantStatName: "counter", WantValue: 1, WantStatType: "c"},
		{In: "nosample_gauge:0.55|g", WantStatName: "nosample_gauge", WantValue: 0.55, WantStatType: "g"},
		{In: "sample_gauge:.66|g|@2.0", WantStatName: "sample_gauge", WantValue: .66, WantStatType: "g"},
		{In: "sample_timer_tag:156|ms|@4.0|tag", WantStatName: "sample_timer_tag", WantValue: 156, WantStatType: "ms"},
	}
	for _, test := range tests {
		testname := fmt.Sprintf("Input: %s", test.In)
		t.Run(testname, func(*testing.T) {
			actualName, actualValue, actualType := SplitStat(test.In)
			if actualName != test.WantStatName {
				t.Errorf("got: %s, want: %s", actualName, test.WantStatName)
			}
			if actualValue != test.WantValue {
				t.Errorf("got: %f, want: %f", actualValue, test.WantValue)
			}
			if actualType != test.WantStatType {
				t.Errorf("got: %s, want: %s", actualType, test.WantStatType)
			}
		})
	}

}
