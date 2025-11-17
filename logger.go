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

// Generate sensor data log file directory from current working directory
func getSensorLogDir() (string, error) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf(
			"ERROR: Checking current working directory failed: %v", err)
	}
	// CAUTION: This uses ./sensor-logs rather than something configurable
	return filepath.Join(cwd, "sensor-logs"), nil
}

// Generate a log file path based on the number of `days` offset from today.
// NOTE: This uses UTC to avoid timezone and daylight savings time troubles
func getLogFilePathForTodayPlus(days int) (string, error) {
	// Log file directory
	logDir, err := getSensorLogDir()
	if err != nil {
		return "", err
	}

	// Calculate relative UTC date for the log file (days=0 is today)
	logDate := time.Now().UTC().AddDate(0, 0, days)

	// Format log file path (e.g. ".../2025-11-17-UTC.csv")
	name := fmt.Sprintf("%s-UTC.csv", logDate.Format("2006-01-02"))
	logFilePath := filepath.Join(logDir, name)
	return logFilePath, nil
}

// Log sensor data from incoming channel to daily rotating log file
// CAUTION: This will return early for file IO errors
func StartLogger(sensorLogChan <-chan SensorData) {
	logFile := CurrentLogFile{}

	// Ensure sensor log directory exists
	logDir, err := getSensorLogDir()
	if err != nil {
		log.Print(err)
		return // CAUTION!
	}
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("ERROR: Creating logs directory failed: %v", err)
		return // CAUTION!
	}

	// Log incoming sensor data channel messages
	for sensorData := range sensorLogChan {

		// Get the current log file path (0 means use today's log file path)
		logFilePath, err := getLogFilePathForTodayPlus(0)
		if err != nil {
			log.Print(err)
			return // CAUTION!
		}

		// Ensure correct log file is open and ready
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
