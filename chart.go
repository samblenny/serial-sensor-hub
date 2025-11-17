// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2025 Sam Blenny
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"time"
)

// Generates the temperature chart as a PNG from the histories map
func GenerateTemperatureChart(histories NodeHistories) ([]byte, error) {
	// Create a blank image (PNG)
	width := 800
	height := 600
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// White background; blue & orange data points
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{},
		draw.Src)
	blue := color.RGBA{31, 119, 180, 255}   // Node 1
	orange := color.RGBA{255, 127, 14, 255} // Node 2

	// Plot the data for Node 1 and Node 2
	if histories["1"] != nil {
		plotTemperatures(img, histories["1"], blue, width, height)
	}
	if histories["2"] != nil {
		plotTemperatures(img, histories["2"], orange, width, height)
	}

	// Encode the PNG image into the buffer of bytes
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		log.Printf("ERROR: Encoding PNG: %v", err)
		return nil, err
	}
	return buf.Bytes(), nil
}

// Plot the temperature data for a node
func plotTemperatures(img *image.RGBA, h *ReportHistory, col color.Color,
	width, height int) {
	if len(h.Reports) == 0 {
		return
	}

	// Temperature range for vertical axis
	const minTempF = 10.0
	const maxTempF = 110.0

	// Start/Stop times for horizontal axis (recent on right, old on left)
	var hours time.Duration = 24
	latestReportTime := h.Reports[len(h.Reports)-1].Timestamp
	earliestReportTime := latestReportTime.Add(-hours * time.Hour)

	// Draw background grid
	gridColor := color.RGBA{169, 169, 169, 255} // light gray
	drawGrid(img, width, height, minTempF, maxTempF, earliestReportTime,
		latestReportTime, gridColor)

	// Plot the data
	for _, report := range h.Reports {
		// If the report is older than 24 hours, skip it
		if report.Timestamp.Before(earliestReportTime) {
			continue
		}

		temp := report.TempF
		timeDiff := latestReportTime.Sub(report.Timestamp).Hours()

		// Scale and translate x coordinate for time
		x := int((1 - (timeDiff / 24)) * float64(width))
		if x < 0 {
			x = 0
		} else if x >= width {
			x = width - 1
		}

		// Scale and translate y coordinate for temperature
		y := height - int((temp-minTempF)/(maxTempF-minTempF)*float64(height))
		if y < 0 {
			y = 0
		} else if y >= height {
			y = height - 1
		}

		// Draw a 3px square data point
		drawSquare(img, x, y, col)
	}
}

// Draw a 3px square at the given coordinates
func drawSquare(img *image.RGBA, x, y int, c color.Color) {
	for yy := y - 1; yy <= y+2; yy++ {
		img.Set(x-1, yy, c)
		img.Set(x, yy, c)
		img.Set(x+1, yy, c)
	}
}

// Draw grid lines on the image
func drawGrid(img *image.RGBA, width, height int, minTempF, maxTempF float64,
	startTime, endTime time.Time, gridColor color.Color) {
	// Horizontal grid lines every 10°F
	for temp := minTempF; temp <= maxTempF; temp += 10 {
		y := height - int((temp-minTempF)/(maxTempF-minTempF)*float64(height))
		if y >= 0 && y < height {
			for x := 0; x < width; x++ {
				img.Set(x, y, gridColor)
			}
		}
	}

	// Vertical grid lines every 4 hours
	duration := endTime.Sub(startTime)
	for i := 0; i <= int(duration.Hours())/4; i++ {
		gridTime := startTime.Add(time.Duration(i*4) * time.Hour)
		timeDiff := endTime.Sub(gridTime).Hours()
		x := int((1 - (timeDiff / 24)) * float64(width))
		if x >= 0 && x < width {
			for y := 0; y < height; y++ {
				img.Set(x, y, gridColor)
			}
		}
	}
}
