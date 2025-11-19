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
		width        = 1024  // Total SVG width
		height       = 768   // Total SVG height
		marginLeft   = 100   // Left margin for labels
		marginTop    = 50    // Top margin for title/labels
		marginRight  = 20    // Right margin
		marginBottom = 150   // Bottom margin for time labels
		minTempF     = 10.0  // Minimum temperature
		maxTempF     = 110.0 // Maximum temperature
		tempStep     = 10    // Temperature axis grid step
		hours        = 36    // Time range (36 hours)
		hoursStep    = 4     // Time axis grid step
	)

	// Adjusted dimensions accounting for margins
	chartWidth := width - marginLeft - marginRight
	chartHeight := height - marginTop - marginBottom

	// Right edge is current time rounded up to the next whole hour, left edge
	// is 36 hours before then
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

	// SVG header with styles
	write(&buf,
		`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN"
  "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">
<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg" >
<style type="text/css">
rect{fill:white;}
line{stroke:#777;stroke-width=1px;}
.blue{fill:#2f87b4d8;}
.orange{fill:#ff7f0ed8;}
text{fill:#000;font-size:16px;font-family:"Verdana",sans-serif;font-weight:bold;
text-anchor:end;}
text.legend{text-anchor:start;}
</style>
`, width, height)

	// White background
	write(&buf, `<rect width="%d" height="%d"/>`+"\n", width, height)

	// Horizontal grid lines and labels
	lineFmt := `<line x1="%d" y1="%d" x2="%d" y2="%d"/>` + "\n"
	for temp := minTempF; temp <= maxTempF; temp += tempStep {
		y := tempToY(temp)
		write(&buf, lineFmt, marginLeft, y, width-marginRight, y)
	}

	// Round current time down to a multiple of 4 hours
	hourFloor4 := int(latestTime.Hour()/4) * 4
	lastT := time.Date(latestTime.Year(), latestTime.Month(),
		latestTime.Day(), hourFloor4, 0, 0, 0, latestTime.Location())

	// Vertical grid lines and labels. This is tricky. There are always lines
	// at the left and right margins. But, the position of the interior lines
	// depends on the current time. The lines always get drawn at multiples of
	// 4 hours, so they shift around based on how far you are from noon, 4 PM,
	// 8 PM, midnight, etc.
	write(&buf, lineFmt, marginLeft, marginTop, marginLeft, height-marginBottom)
	subTime := -time.Duration(hoursStep) * time.Hour
	for t := lastT; t.After(earliestTime); t = t.Add(subTime) {
		x := timeToX(t)
		write(&buf, lineFmt, x, marginTop, x, height-marginBottom)
		fmtTime := t.In(time.Local).Format("Mon 2Jan 3pm")
		xx := int(x) + 8
		yy := int(marginTop + chartHeight + 10)
		write(&buf,
			`<text x="%d" y="%d" transform="rotate(-45 %d,%d)">%v</text>`+"\n",
			xx, yy, xx, yy, fmtTime)
	}
	write(&buf, lineFmt, marginLeft+chartWidth, marginTop,
		marginLeft+chartWidth, height-marginBottom)

	// Temperature axis text labels (vertical axis, left margin)
	for temp := minTempF; temp <= maxTempF; temp += tempStep {
		y := tempToY(temp)
		write(&buf, `<text x="%d" y="%d">%d°F</text>`+"\n",
			int(marginLeft-5), int(y+5), int(temp))
	}

	// Define reusable circle shape
	write(&buf, `<defs><circle id="c" cx="0" cy="0" r="2.2"/></defs>`+"\n")

	// Plot data points by node
	for nodeID, h := range histories {
		if len(h.Reports) == 0 {
			continue
		}

		// Select color by node ID
		var colorClass string
		var legendText string
		var nodeIDi int
		if nodeID == "1" {
			colorClass = "blue"
			legendText = cfg.Node1
			nodeIDi = 1
		} else if nodeID == "2" {
			colorClass = "orange"
			legendText = cfg.Node2
			nodeIDi = 2
		} else {
			// skip nodes higher than 2
			continue
		}

		// Enclose scatter plot dots in a group to share the color class
		write(&buf, `<g class="%s">`+"\n", colorClass)

		// Data series legend
		xBase := marginLeft + ((nodeIDi - 1) * (width - marginLeft -
			marginRight) / 2)
		write(&buf, `<circle r="8" cx="%d" cy="%d"/>`+"\n", xBase+40,
			marginTop-25)
		write(&buf, `<text x="%d" y="%d" class="legend">%s: %s</text>`+"\n",
			xBase+54, marginTop-19, nodeID, legendText)

		// Scatter plot dots
		for _, report := range h.Reports {
			if report.Timestamp.Before(earliestTime) {
				continue
			}
			x := timeToX(report.Timestamp)
			y := tempToY(report.TempF)
			write(&buf, `<use href="#c" x="%d" y="%d"/>%s`, x, y, "\n")
		}
		write(&buf, "</g>\n")
	}

	write(&buf, "</svg>")

	return buf.Bytes(), nil
}
