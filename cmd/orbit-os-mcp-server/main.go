// Orbit MCP server: Gravity client + MCP over Streamable HTTP or stdio (see const mcpUseStreamableHTTP).
package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	orbitclient "github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"
	"github.com/OrbitOS-org/orbit-os-sdk-go/v26/metadata"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"google.golang.org/protobuf/encoding/protojson"
)

var protoJSON = protojson.MarshalOptions{
	UseProtoNames:   true,
	EmitUnpopulated: false,
	Multiline:       true,
	Indent:          "  ",
}

// Hardcoded Orbit / MCP configuration (edit here; no CLI flags).
const (
	// mcpUseStreamableHTTP: true = MCP Streamable HTTP (remote TCP); false = MCP over stdio (e.g. Cursor subprocess).
	mcpUseStreamableHTTP = true
	mcpListenAddr        = "0.0.0.0:9999"
	mcpHTTPPath          = "/mcp"
	// Leave both empty for plain HTTP. Set both for HTTPS (paths on the device / next to binary).
	mcpTLSCertFile = ""
	mcpTLSKeyFile  = ""
)

//go:embed metadata.json
var metadataJSON []byte

var appManifest = metadata.MustParseAppManifestJSON(metadataJSON)

func main() {
	// -host is only used off-device (development from a laptop over TCP + mTLS).
	// On the device the SDK connects to Gravity through the local Unix socket.
	host := flag.String("host", "192.168.1.229", "Device IP address (development only)")
	flag.Parse()

	if (mcpTLSCertFile == "") != (mcpTLSKeyFile == "") {
		fmt.Fprintln(os.Stderr, "orbit-mcp: set both mcpTLSCertFile and mcpTLSKeyFile, or leave both empty")
		os.Exit(2)
	}

	client, err := orbitclient.NewClientAuto(*host)
	if err != nil {
		fmt.Fprintf(os.Stderr, "orbit client error: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()
	defer func() { _ = client.AppHubManager.UnregisterService() }()

	s := server.NewMCPServer(
		"OrbitOS Device MCP",
		appManifest.Version,
		server.WithRecovery(),
	)

	registerSystemTools(s, client)
	registerPackageTools(s, client)
	registerUpdateTools(s, client)
	registerPowerTools(s, client)
	registerGPIOTools(s, client)
	registerWiFiTools(s, client)
	registerEthernetTools(s, client)
	registerUARTTools(s, client)
	registerI2CTools(s, client)
	registerSPITools(s, client)
	registerPWMTools(s, client)
	registerBluetoothTools(s, client)
	registerChunkedUploadTools(s, client)

	startWebUI(client)

	if mcpUseStreamableHTTP {
		runStreamableHTTPServer(s, mcpListenAddr, mcpHTTPPath, mcpTLSCertFile, mcpTLSKeyFile)
		return
	}

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("mcp stdio server error: %v", err)
	}
}

func runStreamableHTTPServer(s *server.MCPServer, addr, path, certFile, keyFile string) {
	opts := []server.StreamableHTTPOption{
		server.WithEndpointPath(path),
	}
	if certFile != "" && keyFile != "" {
		opts = append(opts, server.WithTLSCert(certFile, keyFile))
	}
	httpSrv := server.NewStreamableHTTPServer(s, opts...)

	scheme := "http"
	if certFile != "" {
		scheme = "https"
	}
	log.Printf("Orbit MCP Streamable %s — %s://%s%s", strings.ToUpper(scheme), scheme, addr, path)
	printMCPClientConfigExample(scheme, addr, path)

	errCh := make(chan error, 1)
	go func() { errCh <- httpSrv.Start(addr) }()

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	shutdownStarted := false
	for {
		select {
		case err := <-errCh:
			if err != nil && err != http.ErrServerClosed {
				log.Fatalf("mcp http server: %v", err)
			}
			return
		case sig := <-sigCh:
			if !shutdownStarted {
				shutdownStarted = true
				log.Println("interrupt: shutting down MCP server (Ctrl+C again to exit immediately)")
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
					defer cancel()
					if err := httpSrv.Shutdown(ctx); err != nil {
						log.Printf("mcp http shutdown: %v", err)
					}
				}()
				continue
			}
			log.Printf("interrupt (%v): forcing exit", sig)
			os.Exit(0)
		}
	}
}

func mcpServerWorkDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("workdir: %w", err)
	}
	wd := filepath.Join(filepath.Dir(exe), "workdir")
	if err := os.MkdirAll(wd, 0o700); err != nil {
		return "", fmt.Errorf("workdir: %w", err)
	}
	return wd, nil
}

func printMCPClientConfigExample(scheme, listenAddr, path string) {
	host, port, err := net.SplitHostPort(listenAddr)
	if err != nil {
		host, port = "", listenAddr
	}
	placeholder := false
	urlHost := host
	if host == "" || host == "0.0.0.0" || host == "[::]" {
		urlHost = "<DEVICE_IP>"
		placeholder = true
	}
	fullURL := fmt.Sprintf("%s://%s%s", scheme, net.JoinHostPort(urlHost, port), path)

	cfg := map[string]any{
		"mcpServers": map[string]any{
			"orbit-os": map[string]string{"url": fullURL},
		},
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	_ = enc.Encode(cfg)
	out := bytes.TrimSpace(buf.Bytes())

	fmt.Println()
	fmt.Println("--- MCP client config (e.g. Cursor → .cursor/mcp.json or Settings → MCP) ---")
	fmt.Println(string(out))
	fmt.Println()
	if placeholder {
		fmt.Println("Replace <DEVICE_IP> with the device's IP address on the network (e.g. 192.168.1.50). Do not use 0.0.0.0 on the client.")
		fmt.Println()
	}
	fmt.Println("In Cursor: file .cursor/mcp.json or MCP settings; you can rename \"orbit-os\".")
	fmt.Println("------------------------------------------------------------------")
}

// ── Shared arg helpers ────────────────────────────────────────────────────────

func getStringArg(req mcp.CallToolRequest, key, def string) string {
	args := req.GetArguments()
	if args == nil {
		return def
	}
	v, ok := args[key]
	if !ok || v == nil {
		return def
	}
	s, ok := v.(string)
	if !ok {
		return def
	}
	return s
}

func getBoolArg(req mcp.CallToolRequest, key string, def bool) bool {
	args := req.GetArguments()
	if args == nil {
		return def
	}
	v, ok := args[key]
	if !ok || v == nil {
		return def
	}
	b, ok := v.(bool)
	if !ok {
		return def
	}
	return b
}
