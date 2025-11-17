// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2025 Sam Blenny
package main

import (
	"bytes"
	"fmt"
	"time"
)

// GenerateTemperatureChart creates a simple SVG temperature chart
func GenerateTemperatureChart(histories NodeHistories) ([]byte, error) {
	const (
		width      = 800
		height     = 600
		minTempF   = 10.0
		maxTempF   = 110.0
		hours      = 24
	)

	// Right edge is now, left edge is 24 hours ago
	latestTime := time.Now()
	earliestTime := latestTime.Add(-hours * time.Hour)

	// Coordinate transformations
	tempToY := func(temp float64) int {
		return height - int((temp-minTempF)/(maxTempF-minTempF)*float64(height))
	}

	timeToX := func(t time.Time) int {
		elapsed := t.Sub(earliestTime).Hours()
		return int((elapsed / float64(hours)) * float64(width))
	}

	var buf bytes.Buffer

	// SVG header
	buf.WriteString(fmt.Sprintf("<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"%d\" height=\"%d\">\n", width, height))

	// Define reusable dot shape
	buf.WriteString("<defs><circle id=\"d\" r=\"2\"/></defs>\n")

	// White background
	buf.WriteString(fmt.Sprintf("<rect width=\"%d\" height=\"%d\" fill=\"white\"/>\n", width, height))

	// Grid lines group
	buf.WriteString("<g stroke=\"#ddd\" stroke-width=\"1\">\n")

	// Horizontal grid lines (every 10°F)
	for temp := minTempF; temp <= maxTempF; temp += 10 {
		y := tempToY(temp)
		buf.WriteString(fmt.Sprintf("<line x1=\"0\" y1=\"%d\" x2=\"%d\" y2=\"%d\"/>\n", y, width, y))
	}

	// Vertical grid lines (every 4 hours)
	for i := 0; i <= hours/4; i++ {
		t := earliestTime.Add(time.Duration(i*4) * time.Hour)
		x := timeToX(t)
		buf.WriteString(fmt.Sprintf("<line x1=\"%d\" y1=\"0\" x2=\"%d\" y2=\"%d\"/>\n", x, x, height))
	}

	buf.WriteString("</g>\n")

	// Plot data points by node
	colors := map[string]string{
		"1": "#1f77b4", // blue
		"2": "#ff7f0e", // orange
	}

	for nodeID, h := range histories {
		if len(h.Reports) == 0 {
			continue
		}

		color := colors[nodeID]
		if color == "" {
			color = "#888888"
		}

		// Group for this node's data points
		buf.WriteString(fmt.Sprintf("<g fill=\"%s\">\n", color))

		for _, report := range h.Reports {
			if report.Timestamp.Before(earliestTime) {
				continue
			}

			x := timeToX(report.Timestamp)
			y := tempToY(report.TempF)

			buf.WriteString(fmt.Sprintf("<use href=\"#d\" x=\"%d\" y=\"%d\"/>\n", x, y))
		}

		buf.WriteString("</g>\n")
	}

	buf.WriteString("</svg>")

	return buf.Bytes(), nil
}
