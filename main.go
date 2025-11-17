// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2025 Sam Blenny
package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
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

// Type for managing sensor report histories of multiple sensor nodes
type NodeHistories map[string]*ReportHistory

// Read sensor log files to get reports from the past `days` number of days
func ReadSensorLogHistory(days int) (NodeHistories, error) {
	if days < 0 {
		return nil, fmt.Errorf("ERROR: Expected days >= 0, got: %d", days)
	}
	log.Printf("INFO: Loading sensor node report history from CSV logs")
	histories := make(NodeHistories)

	// Read logs for the past `days` number of days, oldest log first
	for i := days - 1; i >= 0; i-- {
		path, err := getLogFilePathForTodayPlus(-i)
		if err != nil {
			return nil, fmt.Errorf(
				"ERROR: Generating file path for %d days ago: %v", i, err)
		}

		// Open the log file
		file, err := os.Open(path)
		if err != nil {
			// This is fine. For example, maybe there is only the current
			// day's sensor data available.
			log.Printf("WARN: %v", err)
			continue
		}
		reader := csv.NewReader(file)
		log.Printf("INFO: Loading %s", path)

		// Skip CSV header row
		if _, err := reader.Read(); err != nil {
			log.Printf("WARN: Reading CSV header row: %v", err)
			// If the file is empty, that's fine. Skip this file.
			file.Close()
			continue
		}

		// Parse CSV data rows
		// CAUTION: This attempts to continue after parsing errors
		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("WARN: Reading CSV record: %v", err)
				continue
			}

			// Parse record (format: Timestamp,Node,RSSI,SNR,BatteryV,TempF)
			timestamp, err := time.Parse(time.RFC3339, record[0])
			if err != nil {
				log.Printf("WARN: Parsing timestamp: %v", err)

			}
			node := record[1]
			// ignore: rssi := record[2]
			// ignore: snr := record[3]
			batteryV, err := strconv.ParseFloat(record[4], 64)
			if err != nil {
				log.Printf("WARN: Parsing batteryV: %v", err)
			}
			tempF, err := strconv.ParseFloat(record[5], 64)
			if err != nil {
				log.Printf("WARN: Parsing tempF: %v", err)
			}

			// Ensure history exists for this node
			h, exists := histories[node]
			if !exists {
				h = &ReportHistory{}
				histories[node] = h
			}

			// Add the data to the history for this node
			h.Add(timestamp, batteryV, tempF)
		}
		// Close this log file
		file.Close()
	}

	return histories, nil
}

// Format an IRC summary message for the most recent report of nodes 1 and 2
func FormatReportSummary(histories NodeHistories) string {
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

	// Return string in "!pre /..." format for irc-display-bot
	return "!pre " + strings.Join(lines, "")
}

// main() reads serial sensor reports, maintains a 24-hour rolling history per
// node, and sends report summaries by IRC (sets topic of configured channel).
// Example sensor reports (expected format of USB serial stream):
//
//	LORA: -122, -14.0, 1, 38734ca6, 3.80, 63, DUP
//	ESPNOW: -63, 0.0, 2, 38734b3c, 3.80, 64, OK
//
// IRC topic summaries use `!pre /...` format for multi-line formatted output
// on my IRC display bot (see https://github.com/samblenny/irc-display-bot).
// Example of a formatted 4-line summary to work with irc-display-bot:
//
//	!pre /63 368 63 86/  1  Nov16 23:43/66 376 66 93/  2  Nov17 23:43
//
// After the linebreak substitutions, that would appear on a screen as:
//
//	63 368 63 86
//	  1  Nov16 23:43
//	66 376 66 93
//	  2  Nov16 23:43
func main() {
	log.Printf("INFO: Starting serial-sensor-hub")

	// Load configuration file
	cfg, err := IRCLoadConfig("config.json")
	if err != nil {
		log.Fatalf("ERROR: Failed to load IRC config: %v", err)
	}

	// Shutdown context for clean exit (in case of Ctrl-C or whatever)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Channels
	sensorChan := make(chan string, 32)
	reportChan := make(chan string, 32)
	sensorLogChan := make(chan SensorData, 32)

	// Try to initialize sensor node report history from recent log files.
	// Node histories get used to compute 24-hour rolling min/max temperatures.
	histories, err := ReadSensorLogHistory(2)
	if err != nil {
		// Loading the old log data failed, so start from a clean slate
		log.Print(err)
		histories = make(NodeHistories)
	} else {
		// Loading log data worked, so count how much we got
		totalReports := 0
		totalNodes := len(histories)
		for _, history := range histories {
			totalReports += len(history.Reports)
		}
		log.Printf("INFO: Sensor Log Summary: %d nodes, %d reports",
			totalNodes, totalReports)
	}

	// Start IRC bot goroutine (takes several seconds to connect and join)
	go IRCBot(ctx, cfg, reportChan)

	// Send summary of logged sensor reports for nodes 1 and 2 by IRC
	if len(histories) > 0 {
		// First allow time for IRC connect/register/join finish
		time.Sleep(8 * time.Second)
		// Okay, now send it
		summary := FormatReportSummary(histories)
		reportChan <- summary
	}

	// Start serial port sensor monitor, IRC bot, and sensor data logger
	go SerialConnect(sensorChan)
	go StartLogger(sensorLogChan)

	// Start sensorChan fanout to sensor log and reportChan channel
	for report := range sensorChan {
		log.Printf("SENSOR: %s", report)

		matches := sensorReportRE.FindStringSubmatch(report)
		if matches == nil {
			log.Printf("WARN: SENSOR: Bad report format: %s", report)
			continue
		}

		rssi := matches[2]
		snr := matches[3]
		node := matches[4]
		okdup := matches[8]
		if okdup != "OK" {
			log.Printf("INFO: SENSOR: Duplicate: %s", report)
			continue
		}
		batteryV, err := strconv.ParseFloat(matches[6], 64)
		if err != nil {
			log.Printf("WARN: SENSOR: Bad battery voltage: %s", matches[6])
			continue
		}
		tempF, err := strconv.ParseFloat(matches[7], 64)
		if err != nil {
			log.Printf("WARN: SENSOR: Bad temperature F: %s", matches[7])
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

		// Send summary of latest reports for nodes 1 and 2 by IRC
		summary := FormatReportSummary(histories)
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
	log.Print("WARN: serial-sensor-hub shutting down in 5 seconds...")
	time.Sleep(5 * time.Second)
}
