// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2025 Sam Blenny
package main

import (
	"fmt"
	"log"
	"os"
	"sync"
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

var logMutex sync.Mutex

func startLogger(sensorLogChan <-chan SensorData) {
	logDir := "./logs"

	// Try to ensure logs directory exists, create it if not
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("ERROR: Failed to create sensor logs directory: %v", err)
		return
	}

	var currentFileName string
	var logFile *os.File
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	// Start goroutine to watch the clock and rotate the log file when needed
	// (rotation happens daily at midnight local time)
	go func() {
		for range ticker.C {
			rotateLogFile(&currentFileName, &logFile)
		}
	}()

	// On startup, poke the log file rotater to open or create log file
	rotateLogFile(&currentFileName, &logFile)

	// Main loop to write sensor data to log
	for sensorData := range sensorLogChan {
		// Lock the mutex to prevent race with log file rotation
		logMutex.Lock()

		// Write to the current log file
		logLine := fmt.Sprintf("%s,%s,%s,%s,%.2f,%.0f\n",
			sensorData.Timestamp.UTC().Format(time.RFC3339),
			sensorData.Node,
			sensorData.RSSI,
			sensorData.SNR,
			sensorData.BatteryV,
			sensorData.TempF)
		if logFile != nil {
			if _, err := (*logFile).WriteString(logLine); err != nil {
				// If write fails, silently ignore it (but unlock mutex!)
				logMutex.Unlock()
				continue
			}
		}
		logMutex.Unlock()
	}
}

// Sensor data log file rotates at midnight local time
func rotateLogFile(currentFileName *string, logFile **os.File) {
	// This is what the log filename should be based on current time
	newFileName := fmt.Sprintf("./logs/%s-sensors.log",
		time.Now().Format("2006-01-02"))

	// If the log file isn't already open, or if the filename is wrong, then
	// open the correct log file
	if *logFile == nil || newFileName != *currentFileName {
		// Use mutex to prevent race condition with sensor data writer
		logMutex.Lock()

		// Close old file if it exists
		if *logFile != nil {
			(*logFile).Close()
		}

		// Open new file
		var err error
		*logFile, err = os.OpenFile(newFileName,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("WARN: Failed to open %s: %v", newFileName, err)
			logMutex.Unlock()
			return
		}

		// Write CSV header only if the file is new (size==0)
		if stat, _ := (*logFile).Stat(); stat.Size() == 0 {
			header := "Timestamp,Node,RSSI,SNR,BatteryV,TempF\n"
			if _, err := (*logFile).WriteString(header); err != nil {
				logMutex.Unlock()
				return
			}
		}

		// Update current file name
		*currentFileName = newFileName
		log.Printf("Logging sensor data to: %s", newFileName)
		logMutex.Unlock()
	}
}
