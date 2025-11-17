// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2025 Sam Blenny
package main

import (
	"bytes"
	"fmt"
	"time"
)

// Utility function to write formatted strings to a buffer
func write(buf *bytes.Buffer, format string, args ...interface{}) {
	buf.WriteString(fmt.Sprintf(format, args...))
}

// GenerateTemperatureChart creates a simple SVG temperature chart
func GenerateTemperatureChart(histories NodeHistories) ([]byte, error) {
	const (
		width        = 800   // Total SVG width
		height       = 600   // Total SVG height
		marginLeft   = 100    // Left margin for labels
		marginTop    = 20    // Top margin for title/labels
		marginRight  = 20    // Right margin
		marginBottom = 150    // Bottom margin for time labels
		minTempF     = 10.0  // Minimum temperature
		maxTempF     = 110.0 // Maximum temperature
		tempStep     = 10    // Temperature axis grid step
		hours        = 36    // Time range (36 hours)
		hoursStep    = 4     // Time axis grid step
	)

	// Adjusted dimensions accounting for margins
	chartWidth := width - marginLeft - marginRight
	chartHeight := height - marginTop - marginBottom

	// Right edge is now, left edge is 36 hours ago
	latestTime := time.Now()
	earliestTime := latestTime.Add(-hours * time.Hour)

	// Coordinate transformations
	tempToY := func(temp float64) int {
		// Scale temperature to Y position on the chart, considering the margin
		return marginTop + chartHeight -
			int((temp-minTempF)/(maxTempF-minTempF)*float64(chartHeight))
	}

	timeToX := func(t time.Time) int {
		elapsed := t.Sub(earliestTime).Hours()
		// Scale time to X position on the chart, considering the margin
		return marginLeft + int((elapsed/float64(hours))*float64(chartWidth))
	}

	var buf bytes.Buffer

	// SVG header
	write(&buf, "<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"%d\" height=\"%d\">\n", width, height)
	write(&buf, `<style>
path{stroke:#aaa;stroke-width=1px;}
text{stroke:#000;font-size:14px;font-family:Verdana,Arial,sans-serif;text-anchor:end;}
</style>
`)

	// White background
	write(&buf, "<rect width=\"%d\" height=\"%d\" fill=\"white\"/>\n", width, height)

	// Start path element for axis grid lines
	write(&buf, "<path  d=\"")

	// Horizontal grid line segments
	for temp := minTempF; temp <= maxTempF; temp += tempStep {
		y := tempToY(temp)
		write(&buf, "\nM%d %d H %d", marginLeft, y, width-marginRight)
	}

	// Vertical grid line segments
	for i := 0; i <= hours/hoursStep; i++ {
		t := earliestTime.Add(time.Duration(i*hoursStep) * time.Hour)
		x := timeToX(t)
		write(&buf, "\nM%d %d V %d", x, marginTop, height-marginBottom)
	}

	// Close grid line path element
	write(&buf, "\"/>\n")

	// Temperature axis text labels (vertical axis, left margin)
	for temp := minTempF; temp <= maxTempF; temp += tempStep {
		y := tempToY(temp)
		write(&buf, "<text x=\"%d\" y=\"%d\">%d°F</text>\n",
			int(marginLeft-5), int(y+5), int(temp))
	}

	// Time axis text labels (horizontal axis, bottom margin)
	for i := 0; i <= hours/hoursStep; i++ {
		t := earliestTime.Add(time.Duration(i*hoursStep) * time.Hour)
		x := timeToX(t)
		fmtTime := t.Format("Mon 01/02 3PM")
		xx := int(x)+10
		yy := int(marginTop+chartHeight+10)
		write(&buf, "<text x=\"%d\" y=\"%d\" transform=\"rotate(-45 %d,%d)\">%v" +
			"</text>\n", xx, yy, xx, yy, fmtTime)
	}

	// Plot data points by node
	colors := map[string]string{
		"1": "#2f87b4d0", // blue
		"2": "#ff7f0ed0", // orange
	}

	// Define reusable circle shape
	write(&buf, "<defs><circle id=\"c\" r=\"2\"/></defs>\n")

	for nodeID, h := range histories {
		if len(h.Reports) == 0 {
			continue
		}

		color := colors[nodeID]
		if color == "" {
			color = "#888888"
		}

		// Group for this node's data points
		write(&buf, "<g fill=\"%s\">\n", color)

		for _, report := range h.Reports {
			if report.Timestamp.Before(earliestTime) {
				continue
			}

			x := timeToX(report.Timestamp)
			y := tempToY(report.TempF)

			write(&buf, "<use href=\"#c\" x=\"%d\" y=\"%d\"/>\n", x, y)
		}

		write(&buf, "</g>\n")
	}

	write(&buf, "</svg>")

	return buf.Bytes(), nil
}
