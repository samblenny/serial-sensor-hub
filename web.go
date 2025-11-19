// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2025 Sam Blenny
package main

import (
	"context"
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
func StartWebServer(ctx context.Context) {
	// Map URL paths to handler functions
	mux := http.NewServeMux()
	mux.HandleFunc("/chart.svg", chartHandler)
	mux.HandleFunc("/", htmlHandler)

	// Server will bind to all IP addresses (0.0.0.0)
	srv := &http.Server{Addr: "0.0.0.0:8080", Handler: mux}

	// Handler goroutine will shut down the web server when ctx is canceled
	go func() {
		<-ctx.Done()
		log.Printf("DEBUG: Web server got <-ctx.Done()")
		shutdownCtx, cancel := context.WithTimeout(context.Background(),
			5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("ERROR: Web server shutdown: %v", err)
		}
	}()

	log.Printf("INFO: Web server starting on %s", srv.Addr)

	// This blocks until the server is shut down
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("WARN: Web server: %v", err)
	}

	log.Printf("INFO: Web server exited cleanly")
}
