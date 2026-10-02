<p align="center">
  <img src="https://www.orbit-os.org/images/vscode/orbit-os-logo.png" width="300" alt="Orbit OS">
</p>

<h1 align="center">Orbit OS MCP Server</h1>

<p align="center"><b>Let AI agents control real hardware — GPIO, I²C, UART, SPI, PWM, Wi-Fi, Bluetooth and more — over the Model Context Protocol.</b></p>

An [MCP](https://modelcontextprotocol.io) server that runs **on the device** as an Orbit OS app and exposes the device's hardware and system services as MCP tools. Connect Claude, Cursor or any MCP client and ask in plain language: *"scan the I²C bus"*, *"turn on relay 2"*, *"what's the CPU temperature?"*.

Runs on Raspberry Pi, Arduino UNO Q and other ARM64 devices with [Orbit OS](https://www.orbit-os.org/?ref=github-mcp) (free Community Edition).

<a href="https://store.orbit-os.org/app/app-mcp-server?ref=github-mcp"><img src="https://www.orbit-os.org/images/badges/get-it-on-orbit-os-store@3x.png" width="200" alt="Get it on Orbit OS Store"></a>

> [!WARNING]
> **Development and testing tool.** The MCP endpoint has **no authentication**: anyone on the same network can call every tool, including reboot, shutdown, OTA install and factory reset. Use it only on trusted networks and stop the app when you're not using it.

## Features

- **84 tools** across the device's services — see the list below
- **Streamable HTTP** transport (MCP spec), reachable from any machine on your network
- **Built-in web page** in the Orbit OS AppHub showing the connection URL and client config
- Uses the official [Orbit OS Go SDK](https://github.com/OrbitOS-org/orbit-os-sdk-go) — every call goes through the Orbit OS permission model declared in the app manifest

## Tools

| Area | Tools |
|---|---|
| System | device summary, live metrics, API version, plus 29 individual getters (CPU, SoC, RAM, board, kernel, OS, runtime, serials…) |
| GPIO | list pins, get/set direction, get/set level |
| I²C | list buses, scan, transfer |
| SPI | list devices, transfer |
| UART | list ports, read, write, transact |
| PWM | list channels, get, set, stop |
| Wi-Fi | scan, connect/disconnect, status, client config, access point |
| Ethernet | list interfaces, status, enable/disable, config |
| Bluetooth | adapter info, enable/disable, BLE/classic scan, connect, bonded devices |
| Packages | list installed, install (incl. chunked upload), remove |
| Power & updates | reboot, shutdown, OTA install, factory reset |

## Requirements

- A device running [Orbit OS](https://www.orbit-os.org/getting_started.html?ref=github-mcp) (Community Edition: Raspberry Pi 3 / 4 / 5 / Zero 2 W, Arduino UNO Q, other ARM64 boards)
- An MCP client on a computer in the same network
- To build from source: Go 1.25+

## Install

**From the Orbit OS Store (recommended):** install [MCP Server](https://store.orbit-os.org/app/app-mcp-server?ref=github-mcp) on your device in one click.

**From source:**
```bash
git clone https://github.com/OrbitOS-org/orbit-os-app-mcp-server
cd orbit-os-app-mcp-server
go build ./cmd/orbit-os-mcp-server
```
Package and deploy it as a signed `.orb` with [Orbit Studio](https://marketplace.visualstudio.com/items?itemName=orbit-os.orbit-studio) (VS Code).

## Connect an MCP client

Open the **MCP Server** page in your device's AppHub — it shows the URL and the exact config to copy. It looks like this:

```json
{
  "mcpServers": {
    "orbit-os": {
      "url": "http://<DEVICE_IP>:9999/mcp"
    }
  }
}
```

- **Cursor:** `.cursor/mcp.json` or *Settings → MCP*
- **Claude Code:** `claude mcp add --transport http orbit-os http://<DEVICE_IP>:9999/mcp`
- **Any client with Streamable HTTP support:** use the URL above
- Replace `<DEVICE_IP>` with your device's address on the local network. The client must run on a machine that can reach the device (same network).

## Development (Orbit Studio)

This project was created with [Orbit Studio](https://marketplace.visualstudio.com/items?itemName=orbit-os.orbit-studio) and follows its standard layout:

| Path | What |
|---|---|
| `cmd/orbit-os-mcp-server/` | app source — `main.go` (MCP server), `tools_*.go` (tools per service), `webui.*` (AppHub page), `metadata.json` (manifest & permissions) |
| `cmd/orbit-os-mcp-server/orb/icon.svg` | launcher / Store icon |
| `orbit.project.json` | Orbit Studio project settings |

- Open the folder in VS Code with Orbit Studio, **Add / Update SDK** (creates the local `orbit-os-sdk-go/` copy and `go.work`, both git-ignored), then **Run** to develop against a device in Developer Mode, or **Build + Deploy** to install the `.orb`.
- Without Orbit Studio, `go build` uses the published SDK module [`github.com/OrbitOS-org/orbit-os-sdk-go/v26`](https://pkg.go.dev/github.com/OrbitOS-org/orbit-os-sdk-go/v26).
- Development TLS certificates live in `cmd/certs/grpc/` and are never committed.

## Security

This server is meant for **development, testing and demos**:

- The endpoint (`http://<DEVICE_IP>:9999/mcp`) is **plain HTTP with no authentication** — anyone who can reach the device on port 9999 can call every tool.
- Destructive tools (shutdown, reboot, OTA install, package install/remove, Wi-Fi/Ethernet config, factory reset) act on the real device — review what your AI agent is allowed to do and keep a human in the loop.
- Run it only on networks you trust, and stop or uninstall the app when you're done. Don't expose port 9999 to the internet.
- The AppHub page itself is protected by the Orbit OS Launcher login.

## Links

[MCP Server in the Store](https://store.orbit-os.org/app/app-mcp-server?ref=github-mcp) · [Orbit OS](https://www.orbit-os.org/?ref=github-mcp) · [Getting started](https://www.orbit-os.org/getting_started.html?ref=github-mcp) · [SDK reference](https://www.orbit-os.org/api-reference.html?ref=github-mcp) · [Store](https://store.orbit-os.org/?ref=github-mcp) · [Forum](https://forum.orbit-os.org/?ref=github-mcp) · info@orbit-os.org

## License

Apache-2.0 — see [LICENSE](LICENSE) and [NOTICE](NOTICE).
