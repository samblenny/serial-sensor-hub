// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2025 Sam Blenny
package main

import (
	"log"
	"net/http"
)

// Chart handler function for the web server
func chartHandler(w http.ResponseWriter, r *http.Request) {
	// Lock the chart cache for reading
	chartCache.mu.Lock()
	defer chartCache.mu.Unlock()

	// If chart cache is empty, return a 404
	if len(chartCache.Bytes) == 0 {
		http.Error(w, "Chart not available", http.StatusNotFound)
		return
	}

	// Set the response header to SVG image type
	w.Header().Set("Content-Type", "image/svg+xml")

	// Send the chart bytes as the response
	w.Write(chartCache.Bytes)
}

// Start the web server to serve the chart
func setupWebServer() {
	http.HandleFunc("/", chartHandler) // Register chart handler
	hostport := "0.0.0.0:8080"
	log.Printf("INFO: Starting web server on %s...", hostport)
	if err := http.ListenAndServe(hostport, nil); err != nil {
		log.Printf("WARN: Failed to start web server: %v", err)
	}
}
