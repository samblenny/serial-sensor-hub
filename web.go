// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2025 Sam Blenny
package main

import (
	"log"
	"net/http"
)

// Chart handler function to serve SVG file
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

// HTML handler function for the root path "/"
func htmlHandler(w http.ResponseWriter, r *http.Request) {
	// Return an HTML5 page that includes the SVG image
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	// HTML content with an <img> tag that sources the SVG from "/chart.svg"
	htmlContent := `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Temperature Chart</title></head>
<body><img src="/chart.svg" alt="Temperature Chart" /></body></html>`

	w.Write([]byte(htmlContent))
}

// Start the web server to serve the chart
func setupWebServer() {
	// Serve the SVG chart at "/chart.svg"
	http.HandleFunc("/chart.svg", chartHandler)

	// Serve the HTML5 page at "/"
	http.HandleFunc("/", htmlHandler)

	hostport := "0.0.0.0:8080"
	log.Printf("INFO: Starting web server on %s...", hostport)
	if err := http.ListenAndServe(hostport, nil); err != nil {
		log.Printf("WARN: Failed to start web server: %v", err)
	}
}
