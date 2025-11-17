// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2025 Sam Blenny
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

type SensorData struct {
	Timestamp time.Time
	RSSI      string
	SNR       string
	Node      string
	BatteryV  float64
	TempF     float64
}

type CurrentLogFile struct {
	FilePath string
	File     *os.File
}

// Does log file already have a non-zero amount of data?
func (c *CurrentLogFile) IsEmpty() bool {
	if c.File != nil {
		if stat, err := c.File.Stat(); err == nil {
			return stat.Size() == 0
		}
	}
	return false
}

// Prepare for logging to new file, closing the old one if needed
func (c *CurrentLogFile) Rotate(filePath string) error {
	// Close old file if it exists
	if c.File != nil {
		c.File.Close()
	}

	// Open new file
	var err error
	c.File, err = os.OpenFile(filePath,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	// Update current file name
	c.FilePath = filePath
	return nil
}

// Log sensor data from incoming channel to daily rotating log file
// CAUTION: This will return early for file IO errors
func startLogger(sensorLogChan <-chan SensorData) {
	// Build absolute path for ./logs under server's current working directory
	cwd, err := os.Getwd()
	if err != nil {
		log.Printf("ERROR: Checking working directory for ./logs failed")
		return // CAUTION!
	}
	logDir := filepath.Join(cwd, "logs")

	// Ensure logs directory exists, creating it if needed
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("ERROR: Creating sensor logs directory failed: %v", err)
		return // CAUTION!
	}
	log.Printf("INFO: Sensor data log directory: %s", logDir)

	logFile := CurrentLogFile{}

	// Log incoming sensor data channel messages
	for sensorData := range sensorLogChan {

		// This is what the log filename should be based on current time
		logFilePath := filepath.Join(logDir,
			fmt.Sprintf("%s-sensors.log", time.Now().Format("2006-01-02")))

		// Make sure the correct log file is open and ready
		if logFile.File == nil || logFilePath != logFile.FilePath {
			if err := logFile.Rotate(logFilePath); err != nil {
				log.Print("ERROR: Rotating log file failed: %v", err)
				return // CAUTION!
			}
			log.Printf("INFO: Logging sensor data to: %s", logFilePath)

			// Only write CSV header if this is a new empty file. For example,
			// the server could be stopped then restarted on the same day.
			if logFile.IsEmpty() {
				header := "Timestamp,Node,RSSI,SNR,BatteryV,TempF\n"
				if _, err := logFile.File.WriteString(header); err != nil {
					log.Print("ERROR: Writing sensor log data failed: %v", err)
					return // CAUTION!
				}
			}
		}

		// Write sensor data to log file
		logLine := fmt.Sprintf("%s,%s,%s,%s,%.2f,%.0f\n",
			sensorData.Timestamp.UTC().Format(time.RFC3339),
			sensorData.Node,
			sensorData.RSSI,
			sensorData.SNR,
			sensorData.BatteryV,
			sensorData.TempF)
		if _, err := logFile.File.WriteString(logLine); err != nil {
			log.Print("ERROR: Writing sensor log data failed: %v", err)
			return // CAUTION!
		}
	}
}
