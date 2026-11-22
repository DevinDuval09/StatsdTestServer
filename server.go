package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
)

type StatsTypes struct {
	Type  string
	Count int
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

func StatsListener(host string, port int, stats *sync.Map) {
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

	log.Printf("Listening on %s", conn.LocalAddr().String())

	buffer := make([]byte, 1024)
	for {
		//accept connection
		sizeRead, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Error parsing packet: %v", err)
			continue
		}
		//dump buffer into reader
		reader := bytes.NewReader(buffer[:sizeRead])

		//packet -> individual lines -> process -> counters
		go StatPacketHandler(reader, stats)

	}
}
