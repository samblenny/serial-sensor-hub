// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2025 Sam Blenny
package main

import (
	"time"
)

// One parsed sensor report from a node
type Report struct {
	Timestamp time.Time
	TempF     float64
	BatteryV  float64
}

// 24-hour rolling history of reports for one sensor node
type ReportHistory struct {
	Reports  []Report
	MinTempF float64
	MaxTempF float64
}

// Add a new report and prune anything older than 24 hours.
// Also recomputes min and max temperatures after pruning.
func (h *ReportHistory) Add(timestamp time.Time, batteryV, tempF float64) {
	// Build and append the new report
	r := Report{
		Timestamp: timestamp,
		TempF:     tempF,
		BatteryV:  batteryV,
	}
	h.Reports = append(h.Reports, r)

	// Prune reports older than 24 hours
	cutoff := time.Now().Add(-24 * time.Hour)
	i := 0
	for i < len(h.Reports) && h.Reports[i].Timestamp.Before(cutoff) {
		i++
	}
	if i > 0 {
		h.Reports = h.Reports[i:]
	}

	// Recompute min/max after prune
	if len(h.Reports) == 0 {
		h.MinTempF = 0
		h.MaxTempF = 0
		return
	}
	min := h.Reports[0].TempF
	max := h.Reports[0].TempF
	for _, r := range h.Reports[1:] {
		if r.TempF < min {
			min = r.TempF
		}
		if r.TempF > max {
			max = r.TempF
		}
	}
	h.MinTempF = min
	h.MaxTempF = max
}
