package main

import (
	"strings"
	"sync"
	"testing"
)

func TestStatPacketHandler(t *testing.T) {
	var tests = []struct {
		In     string
		Expect map[string]StatsTypes
	}{
		{"counter:1|c\n", map[string]StatsTypes{"counter": StatsTypes{Count: 1, Type: "c"}}},
		{"counter:6|c\nnosample_gauge:0.55|g\n", map[string]StatsTypes{"counter": StatsTypes{Count: 7, Type: "c"}, "nosample_gauge": StatsTypes{Count: 1, Type: "g"}}},
		{"new_counter: 4|c\nnosample_gauge:4.3|g\nsample_gauge:.66|g|@2.0\n", map[string]StatsTypes{"sample_gauge": StatsTypes{Count: 1, Type: "g"}, "counter": StatsTypes{Count: 7, Type: "c"}, "new_counter": StatsTypes{Count: 4, Type: "c"}, "nosample_gauge": StatsTypes{Count: 2, Type: "g"}}},
		{"sample_timer_tag:156|ms|@4.0|tag\n", map[string]StatsTypes{"sample_gauge": StatsTypes{Count: 1, Type: "g"}, "counter": StatsTypes{Count: 7, Type: "c"}, "new_counter": StatsTypes{Count: 4, Type: "c"}, "nosample_gauge": StatsTypes{Count: 2, Type: "g"}, "sample_timer_tag": StatsTypes{Count: 1, Type: "ms"}}},
	}
	testmap := &sync.Map{}
	for _, test := range tests {
		//input -> string reader
		reader := strings.NewReader(test.In)
		//execute function
		StatPacketHandler(reader, testmap)
		//check testmap
		for k, st := range test.Expect {
			actual, ok := testmap.Load(k)
			stat, success := actual.(StatsTypes)
			if !success {
				t.Errorf("None StatsTypes loaded into map")
				continue
			}
			if !ok {
				t.Errorf("Missing key %v from stat", stat)
				continue
			}
			if stat.Type != st.Type {
				t.Errorf("Got stat type %s wanted %s for %s", stat.Type, st.Type, k)
			}
			if stat.Count != st.Count {
				t.Errorf("Got count %d wanted %d for %s", stat.Count, st.Count, k)
			}
		}
	}
}
