package main

import (
	"log"
	"strconv"
	"strings"
)

func ParseLine(line string) []string {
	/**
	Split a statsd text line into the stat name, value, tag, sampling rate, tag, etc
	**/
	isStatsdDelimiter := func(c rune) bool {
		return string(c) == ":" || string(c) == "|" || string(c) == "@"
	}

	return strings.FieldsFunc(line, isStatsdDelimiter)
}

func SplitStat(line string) (string, float64, string) {
	/**
	Split a text statsd line into the name, number value, and statsd tag
	**/
	values := ParseLine(line)
	number, err := strconv.ParseFloat(values[1], 64)
	if err != nil {
		log.Fatalf("Error parsing numerical value: %v", err)
	}
	return values[0], number, values[2]
}
