// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2025 Sam Blenny
package main

import (
	"log"
	"net/http"
	"strconv"
	"time"
)

// Chart handler function to serve SVG file
func chartHandler(w http.ResponseWriter, r *http.Request) {
	// Lock the chart cache for reading
	chartCache.mu.Lock()
	defer chartCache.mu.Unlock()

	// Set content type and length response headers for SVG image
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Content-Length", strconv.Itoa(len(chartCache.Bytes)))

	// Send response
	w.Write(chartCache.Bytes)
}

// HTML handler function for the root path "/"
func htmlHandler(w http.ResponseWriter, r *http.Request) {
	// HTML content with an <img> tag that sources the SVG from "/chart.svg"
	html := []byte(`<!DOCTYPE html>
<html lang="en"><head><meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Temperature Chart</title>
<style>
:root{color-scheme:light dark;} /* use system's dark mode setting */
img{max-width:100%;height:auto;} /* scale width on narrow screens */
</style>
</head>
<body><img src="/chart.svg" alt="Temperature Chart">
</body></html>
`)

	// Set content type and length response headers for HTML5
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(html)))

	// Send response
	w.Write(html)
}

// Start the web server to serve the chart
func setupWebServer() {
	// Serve the SVG chart at "/chart.svg"
	http.HandleFunc("/chart.svg", chartHandler)

	// Serve the HTML5 page at "/"
	http.HandleFunc("/", htmlHandler)

	for {
		hostport := "0.0.0.0:8080"
		log.Printf("INFO: Starting web server on %s...", hostport)
		if err := http.ListenAndServe(hostport, nil); err != nil {
			log.Printf("WARN: web server: %v", err)
			time.Sleep(10 * time.Second)
		}
	}
}
