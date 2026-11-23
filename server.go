package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
)

type StatsTypes struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

func statsSyncMapToJson(sm *sync.Map) ([]byte, error) {
	m := make(map[string]StatsTypes)

	sm.Range(func(key, value any) bool {
		strKey, ok := key.(string)
		if !ok {
			strKey = fmt.Sprint(key)
		}
		val, success := value.(*StatsTypes)
		if !success {
			log.Fatalf("Failed to convert value to StatsTypes while converting to json.")
		}
		m[strKey] = StatsTypes{Count: val.Count, Type: val.Type}

		return true
	})
	return json.Marshal(m)
}

func StatPacketHandler(packet io.Reader, stats *sync.Map) {
	//convert to string
	var builder strings.Builder
	_, err := io.Copy(&builder, packet)
	if err != nil {
		log.Printf("Error copying packet: %v", err)
		return
	}

	converted := builder.String()

	//for each line, if the stat name isn't in the map already, add it with a single count for non counters and add the full value for counters
	//if stat does exist, increment counters
	for line := range strings.Lines(converted) {
		name, value, statType := SplitStat(line)
		current, ok := stats.Load(name)
		switch statType {
		case "c":
			if ok {
				stat, success := current.(*StatsTypes)
				if !success {
					log.Printf("Failed to cast StatsTypes")
					continue
				}
				stat.Count += int(value)
			} else {
				stats.Store(name, &StatsTypes{Count: int(value), Type: statType})
			}
		default:
			if ok {
				stat, success := current.(*StatsTypes)
				if !success {
					log.Printf("Failed to cast StatsTypes")
					continue
				}
				stat.Count += 1
			} else {
				stats.Store(name, &StatsTypes{Count: 1, Type: statType})
			}
		}
	}

}

func StatsListener(host string, port int, stats *sync.Map, ctx context.Context) {
	listenAddr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	udpAddr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		log.Fatalf("Error resolving listener: %v", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatalf("Error listening: %v", err)
	}

	defer conn.Close()

	stop := context.AfterFunc(ctx, func() {
		conn.Close()
	})
	defer stop()

	log.Printf("Listening on %s", conn.LocalAddr().String())

	buffer := make([]byte, 1024)
	for {
		//accept connection
		sizeRead, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				log.Printf("Stoping CommandListener")
				return
			}
			log.Printf("Error parsing packet: %v", err)
			continue
		}
		//dump buffer into reader
		reader := bytes.NewReader(buffer[:sizeRead])

		//packet -> individual lines -> process -> counters
		go StatPacketHandler(reader, stats)

	}
}

func CommandListener(host string, port int, stats *sync.Map, ctx context.Context) {
	listenAddr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	udpAddr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		log.Fatalf("Error resolving listener: %v", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatalf("Error listening: %v", err)
	}

	defer conn.Close()

	stop := context.AfterFunc(ctx, func() {
		conn.Close()
	})
	defer stop()

	log.Printf("Listening for commands on %s", conn.LocalAddr().String())

	buffer := make([]byte, 1024)
	for {
		//accept connection
		sizeRead, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				log.Printf("Stoping CommandListener")
				return
			}
			log.Printf("Error parsing packet: %v", err)
			continue
		}
		//dump buffer into reader
		reader := bytes.NewReader(buffer[:sizeRead])
		var builder strings.Builder
		_, err = io.Copy(&builder, reader)
		if err != nil {
			log.Printf("Error copying packet: %v", err)
			continue
		}

		//convert buffer to string
		command := builder.String()

		//switch on command
		switch command {
		//peek -> send json of current stats
		case "peek":
			resp, err := statsSyncMapToJson(stats)
			if err != nil {
				log.Printf("Error creating json of stats: %v", err)
				continue
			}
			_, err = conn.WriteTo(resp, clientAddr)
			//get -> send json of current stats and reset
		case "get":
			resp, err := statsSyncMapToJson(stats)
			if err != nil {
				log.Printf("Error creating json of stats: %v", err)
				continue
			}
			_, err = conn.WriteTo(resp, clientAddr)
			if err != nil {
				log.Printf("Error sending stats: %v", err)
				continue
			}
			stats.Clear()
		case "reset":
			//reset -> don't send, only reset stats
			stats.Clear()
			_, err := conn.WriteTo([]byte("stats cleared"), clientAddr)
			if err != nil {
				log.Printf("Error sending clear confirmation: %v", err)
				continue
			}
		default:
			_, err := conn.WriteTo([]byte("only support commands peek, get, and reset"), clientAddr)
			if err != nil {
				log.Printf("Error sending error notification")
			}
		}

	}
}
