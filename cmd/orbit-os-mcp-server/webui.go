package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net"
	"net/http"
	"strconv"

	orbitclient "github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"
)

//go:embed webui.html
var webuiHTML []byte

//go:embed orb/icon.svg
var iconSVG []byte

const (
	webUIHost         = "127.0.0.1"
	webUIPortMin      = 50000
	webUIPortMax      = 60000
	webUIBindAttempts = 50
	webUIRoute        = "/mcp-server"
)

func startWebUI(client *orbitclient.Client) {
	_, mcpPort, _ := net.SplitHostPort(mcpListenAddr)

	mux := http.NewServeMux()

	mux.HandleFunc("/favicon.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		_, _ = w.Write(iconSVG)
	})

	mux.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		scheme := "http"
		if mcpTLSCertFile != "" {
			scheme = "https"
		}
		mcpURL := fmt.Sprintf("%s://%s%s", scheme, net.JoinHostPort(detectLocalIP(), mcpPort), mcpHTTPPath)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"mcp_url": mcpURL, "version": appManifest.Version})
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(webuiHTML)
	})

	ln, err := listenRandomPort()
	if err != nil {
		log.Printf("WebUI disabled: %v", err)
		return
	}
	listenAddr := ln.Addr().String()

	if err := client.AppHubManager.RegisterWebUI(listenAddr, webUIRoute); err != nil {
		log.Printf("AppHub registration failed: %v", err)
	} else {
		log.Printf("WebUI registered in AppHub — listening on %s", listenAddr)
	}

	go func() {
		if err := http.Serve(ln, mux); err != nil {
			log.Printf("WebUI server error: %v", err)
		}
	}()
}

// listenRandomPort binds to a random port in [webUIPortMin, webUIPortMax],
// picking a new random port whenever the bind fails.
func listenRandomPort() (net.Listener, error) {
	var lastErr error
	for range webUIBindAttempts {
		port := webUIPortMin + rand.IntN(webUIPortMax-webUIPortMin+1)
		ln, err := net.Listen("tcp", net.JoinHostPort(webUIHost, strconv.Itoa(port)))
		if err == nil {
			return ln, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("no free port in %d-%d after %d attempts: %w", webUIPortMin, webUIPortMax, webUIBindAttempts, lastErr)
}

// detectLocalIP returns the preferred outbound IP of this machine without
// sending any packet (UDP connect only resolves the routing table).
func detectLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "localhost"
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}
