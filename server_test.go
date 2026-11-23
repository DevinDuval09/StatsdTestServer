package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

func syncMapIsEmpty(m *sync.Map) bool {
	var empty bool = true
	m.Range(func(key, value any) bool {
		val, ok := value.(*StatsTypes)
		if !ok {
			return true
		}
		if val.Count == 0 {
			return true
		}
		empty = false
		return false // stop iteration immediately
	})
	return empty
}

func commandClient(host string, port int, command string) ([]byte, error) {
	serverAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return []byte{}, err
	}

	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		return []byte{}, err
	}

	buffer := make([]byte, 1024)

	conn.SetReadDeadline(time.Now().Add(4 * time.Second))

	_, err = conn.Write([]byte(command))

	if err != nil {
		return []byte{}, err
	}

	n, _, err := conn.ReadFromUDP(buffer)

	if err != nil {
		return []byte{}, err
	}

	return buffer[:n], nil

}

func statClient(host string, port int, stats string) error {
	serverAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		return err
	}

	_, err = conn.Write([]byte(stats))

	if err != nil {
		return err
	}
	return nil

}

func TestStatPacketHandler(t *testing.T) {
	var tests = []struct {
		In     string
		Expect map[string]StatsTypes
	}{
		{"counter:1|c\n", map[string]StatsTypes{"counter": StatsTypes{Count: 1, Type: "c"}}},
		{"counter:6|c\r\nnosample_gauge:0.55|g\r\n", map[string]StatsTypes{"counter": StatsTypes{Count: 7, Type: "c"}, "nosample_gauge": StatsTypes{Count: 1, Type: "g"}}},
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
			stat, success := actual.(*StatsTypes)
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

func TestStatSyncToJson(t *testing.T) {
	testmap := &sync.Map{}

	var counterStat StatsTypes = StatsTypes{Count: 39, Type: "c"}
	var gaugeStat StatsTypes = StatsTypes{Count: 6, Type: "g"}
	var timerStat StatsTypes = StatsTypes{Count: 3, Type: "ms"}

	testmap.Store("counter", &counterStat)
	testmap.Store("gauge", &gaugeStat)
	testmap.Store("timer", &timerStat)

	var expected = make(map[string]StatsTypes)
	expected["counter"] = counterStat
	expected["gauge"] = gaugeStat
	expected["timer"] = timerStat

	marshalled, err := statsSyncMapToJson(testmap)
	if err != nil {
		t.Errorf("Failed to marshal map")
	}
	var actual map[string]StatsTypes

	err = json.Unmarshal(marshalled, &actual)
	if err != nil {
		t.Errorf("Failed to unmarshal bytes")
	}

	for k, ev := range expected {
		av, ok := actual[k]
		if !ok {
			t.Errorf("Missing key %v from actual", k)
			continue
		}
		if av.Type != ev.Type {
			t.Errorf("Got av type %s wanted %s for %s", av.Type, ev.Type, k)
		}
		if av.Count != ev.Count {
			t.Errorf("Got count %d wanted %d for %s", av.Count, ev.Count, k)
		}
	}

}

func TestGetCommandListener(t *testing.T) {

	var configs = Configs{CommandHost: "127.0.0.1", CommandPort: 20000}

	testmap := &sync.Map{}

	var counterStat StatsTypes = StatsTypes{Count: 39, Type: "c"}
	var gaugeStat StatsTypes = StatsTypes{Count: 6, Type: "g"}
	var timerStat StatsTypes = StatsTypes{Count: 3, Type: "ms"}

	testmap.Store("counter", &counterStat)
	testmap.Store("gauge", &gaugeStat)
	testmap.Store("timer", &timerStat)

	var expected = make(map[string]StatsTypes)
	expected["counter"] = counterStat
	expected["gauge"] = gaugeStat
	expected["timer"] = timerStat

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL)
	defer stop()

	go CommandListener(configs.CommandHost, configs.CommandPort, testmap, ctx)

	packet, err := commandClient(configs.CommandHost, configs.CommandPort, "get")

	if err != nil {
		t.Errorf("Error retrieving stats: %v", err)
	}
	var actual map[string]StatsTypes

	err = json.Unmarshal(packet, &actual)
	if err != nil {
		t.Errorf("Failed to unmarshal bytes")
	}

	for k, ev := range expected {
		av, ok := actual[k]
		if !ok {
			t.Errorf("Missing key %v from actual", k)
			continue
		}
		if av.Type != ev.Type {
			t.Errorf("Got av type %s wanted %s for %s", av.Type, ev.Type, k)
		}
		if av.Count != ev.Count {
			t.Errorf("Got count %d wanted %d for %s", av.Count, ev.Count, k)
		}
	}

	if !syncMapIsEmpty(testmap) {
		t.Errorf("Stats map didn't clear")
	}

}

func TestIntegrated(t *testing.T) {

	var configs = Configs{CommandHost: "127.0.0.1", CommandPort: 9000, StatsHost: "127.0.0.1", StatsPort: 10000}

	integratedmap := &sync.Map{}

	var counterStat StatsTypes = StatsTypes{Count: 39, Type: "c"}
	var gaugeStat StatsTypes = StatsTypes{Count: 6, Type: "g"}
	var timerStat StatsTypes = StatsTypes{Count: 3, Type: "ms"}

	//testmap.Store("counter", &counterStat)
	//testmap.Store("gauge", &gaugeStat)
	//testmap.Store("timer", &timerStat)
	//testmap.Store("emptycounter", &emptycounterStat)

	var expected = make(map[string]StatsTypes)
	expected["counter"] = counterStat
	expected["gauge"] = gaugeStat
	expected["timer"] = timerStat

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL)
	defer stop()

	go CommandListener(configs.CommandHost, configs.CommandPort, integratedmap, ctx)
	go StatsListener(configs.StatsHost, configs.StatsPort, integratedmap, ctx)
	time.Sleep(1 * time.Second)

	//send packets
	err := statClient(configs.StatsHost, configs.StatsPort, "counter:39|c\nemptycounter:0|c\ngauge:.53|g\ntimer:150|ms\n")
	if err != nil {
		t.Errorf("Error sending first packet: %v", err)
		return
	}
	err = statClient(configs.StatsHost, configs.StatsPort, "gauge:.53|g\r\ntimer:150|ms\r\ngauge:.66|g\r\ngauge:.95|g\r\ngauge:1.99|g\r\n")
	if err != nil {
		t.Errorf("Error sending second packet: %v", err)
		return
	}

	err = statClient(configs.StatsHost, configs.StatsPort, "gauge:.53|g\r\ntimer:150|ms\r\n")
	if err != nil {
		t.Errorf("Error sending third packet: %v", err)
		return
	}

	packet, err := commandClient(configs.CommandHost, configs.CommandPort, "peek")

	if err != nil {
		t.Errorf("Error retrieving stats: %v", err)
	}
	var actual map[string]StatsTypes

	err = json.Unmarshal(packet, &actual)
	if err != nil {
		t.Errorf("Failed to unmarshal bytes")
	}

	for k, ev := range expected {
		av, ok := actual[k]
		if !ok {
			t.Errorf("Missing key %v from actual (%v)", k, actual)
			continue
		}
		if av.Type != ev.Type {
			t.Errorf("Got av type %s wanted %s for %s", av.Type, ev.Type, k)
		}
		if av.Count != ev.Count {
			t.Errorf("Got count %d wanted %d for %s", av.Count, ev.Count, k)
		}
	}

	_, ok := actual["emptycounter"]

	if ok {
		t.Errorf("Got a value for emptycounter")
	}

	if syncMapIsEmpty(integratedmap) {
		t.Errorf("Stats map cleared from peek: %v", integratedmap)
	}

	time.Sleep(500 * time.Millisecond)

	_, err = commandClient(configs.CommandHost, configs.CommandPort, "reset")

	time.Sleep(1 * time.Second)

	if !syncMapIsEmpty(integratedmap) {
		t.Errorf("Stats map failed to clear from reset")
	}

}
