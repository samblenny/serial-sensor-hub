// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2025 Sam Blenny
package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var sensorReportRE = regexp.MustCompile(
	`^(ESPNOW|LORA):\s*` + // Originating protocol from sensor gateway
		`([^,]+),\s*` + // RSSI (float)
		`([^,]+),\s*` + // SNR (float)
		`([^,]+),\s*` + // Node address (uint8)
		`([^,]+),\s*` + // Timestamp (uint32)
		`([^,]+),\s*` + // Battery voltage (float)
		`([^,]+),\s*` + // Temperature F (float)
		`([^,]+)`) // Monotonic increasing timestamp check: "OK" or "DUP"

// main() reads serial sensor reports, maintains a 24-hour rolling history per
// node, and generates IRC bot summaries for nodes IDs 1 and 2.
//
// Sensor reports are expected in this format (NOTE: timestamp should be
// monotonic increasing, but don't trust it to accurately reflect clock time):
//
//	<PROTOCOL>: <RSSI>, <SNR>, <NODE_ID>, <TIMESTAMP>, <BATTERY_V>, <TEMP_F>, <OK/DUP>
//
// Example reports:
//
//	LORA: -122, -14.0, 1, 38734ca6, 3.80, 63, DUP
//	ESPNOW: -63, 0.0, 2, 38734b3c, 3.80, 64, OK
//
// Each new report for a node updates its rolling 24-hour history, including
// minimum and maximum temperatures observed.
//
// After adding a report, this generates a summary message for the IRC bot to
// send. The summary format uses the `!pre /...` syntax for my IRC display bot
// (see https://github.com/samblenny/irc-display-bot). The format uses
// delimeters to expand one line into up to 4 lines in the format:
//
//	!pre /<line1>/<line2>/<line3>/<line4>
//
// The expanded format is:
//
//	<line1>: <node1.TempF> <node1.Centivolts> <node1.MinTempF> <node1.MaxTempF>
//	<line2>:   <node1 ID>  <node1 timestamp of last report>
//	<line3>: <node2.TempF> <node2.Centivolts> <node2.MinTempF> <node2.MaxTempF>
//	<line4>:   <node2 ID>  <node2 timestamp of last report>
//
// When no data is available for a node, those lines get a "--" placeholder.
// Example summary format sent to the IRC bot:
//
//	!pre /63 379 63 63/ 1  Nov15 01:59/67 389 67 67/ 2  Nov15 01:58
func main() {
	// Load configuration file
	cfg, err := IRCLoadConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to load IRC config: %v", err)
	}

	// Shutdown context for clean exit (in case of Ctrl-C or whatever)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Channels
	sensorChan := make(chan string, 32)
	reportChan := make(chan string, 32)
	sensorLogChan := make(chan SensorData, 32)

	// Start serial port sensor monitor, IRC bot, and sensor data logger
	go SerialConnect(sensorChan)
	go IRCBot(ctx, cfg, reportChan)
	go startLogger(sensorLogChan)

	// This map maintains a ReportHistory for each nodeID to compute 24-hour
	// rolling min/max temperature statistics
	histories := make(map[string]*ReportHistory)

	// Start sensorChan fanout to sensor log and reportChan channel
	for report := range sensorChan {
		log.Printf("SENSOR: %s", report)

		matches := sensorReportRE.FindStringSubmatch(report)
		if matches == nil {
			log.Printf("Bad sensor report format: %s", report)
			continue
		}

		rssi := matches[2]
		snr := matches[3]
		node := matches[4]
		okdup := matches[8]
		if okdup != "OK" {
			log.Printf("Duplicate: %s", report)
			continue
		}
		batteryV, err := strconv.ParseFloat(matches[6], 64)
		if err != nil {
			log.Printf("Bad battery voltage value: %s", matches[6])
			continue
		}
		tempF, err := strconv.ParseFloat(matches[7], 64)
		if err != nil {
			log.Printf("Bad temperature F value: %s", matches[7])
			continue
		}

		// Ensure history exists for this node
		h, exists := histories[node]
		if !exists {
			h = &ReportHistory{}
			histories[node] = h
		}

		// Add report to node's rolling 24h history and recompute min/max
		timestamp := time.Now()
		h.Add(timestamp, batteryV, tempF)

		// Prepare summary for nodes 1 and 2
		lines := []string{}

		for _, nodeID := range []string{"1", "2"} {
			h, exists := histories[nodeID]
			if !exists || len(h.Reports) == 0 {
				// No data yet for this node
				lines = append(lines, "/--/--")
				continue
			}
			last := h.Reports[len(h.Reports)-1]
			// Format timestamp like "Nov15 05:30"
			timestampStr := last.Timestamp.Format("Jan02 15:04")
			lines = append(lines,
				fmt.Sprintf("/%.0f %.0f %.0f %.0f/  %s  %s",
					last.TempF, 100*last.BatteryV, h.MinTempF, h.MaxTempF,
					nodeID, timestampStr))
		}

		// Send report summary by IRC
		summary := "!pre " + strings.Join(lines, "")
		reportChan <- summary

		// Log the report to disk
		sensorData := SensorData{
			Timestamp: timestamp,
			Node:      node,
			RSSI:      rssi,
			SNR:       snr,
			BatteryV:  batteryV,
			TempF:     tempF,
		}
		sensorLogChan <- sensorData
	}

	close(reportChan)
}
