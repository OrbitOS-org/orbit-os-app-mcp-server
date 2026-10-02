package main

import (
	"context"
	"encoding/json"
	"fmt"

	orbitclient "github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerWiFiTools(s *server.MCPServer, client *orbitclient.Client) {
	s.AddTool(mcp.NewTool("orbit_wifi_list_interfaces",
		mcp.WithDescription("List all Wi-Fi interfaces and their link properties."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ifaces, err := client.WiFiManager.ListInterfaces()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		var out []map[string]any
		for _, iface := range ifaces {
			b, _ := protoJSON.Marshal(iface)
			item := map[string]any{}
			_ = json.Unmarshal(b, &item)
			out = append(out, item)
		}
		if out == nil {
			out = []map[string]any{}
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_wifi_status",
		mcp.WithDescription("Get connection status and link properties for a Wi-Fi interface."),
		mcp.WithString("interface", mcp.Required(), mcp.Description("Interface name, e.g. wlan0")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ifname, err := req.RequireString("interface")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		connected, _ := client.WiFiManager.IsConnected(ifname)
		mode, _ := client.WiFiManager.GetMode(ifname)
		props, err := client.WiFiManager.GetLinkProperties(ifname)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		pb, _ := protoJSON.Marshal(props)
		propsMap := map[string]any{}
		_ = json.Unmarshal(pb, &propsMap)
		b, _ := json.MarshalIndent(map[string]any{
			"connected":  connected,
			"mode":       fmt.Sprintf("%v", mode),
			"properties": propsMap,
		}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_wifi_scan",
		mcp.WithDescription("Scan for available Wi-Fi networks on an interface."),
		mcp.WithString("interface", mcp.Required(), mcp.Description("Interface name, e.g. wlan0")),
		mcp.WithBoolean("force_rescan", mcp.Description("Force a new hardware scan (~3s). Default: false.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ifname, err := req.RequireString("interface")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		networks, err := client.WiFiManager.Scan(ifname, getBoolArg(req, "force_rescan", false))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		var out []map[string]any
		for _, n := range networks {
			nb, _ := protoJSON.Marshal(n)
			item := map[string]any{}
			_ = json.Unmarshal(nb, &item)
			out = append(out, item)
		}
		if out == nil {
			out = []map[string]any{}
		}
		b, _ := json.MarshalIndent(map[string]any{"networks": out}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_wifi_set_client_config",
		mcp.WithDescription("Store Wi-Fi client settings (SSID, password, IP). Call orbit_wifi_connect to apply."),
		mcp.WithString("interface", mcp.Required(), mcp.Description("Interface name, e.g. wlan0")),
		mcp.WithString("ssid", mcp.Required(), mcp.Description("Network SSID")),
		mcp.WithString("password", mcp.Description("Network password (empty for open networks)")),
		mcp.WithString("security", mcp.Description("Security: WPA2 (default), WPA3, OPEN")),
		mcp.WithBoolean("dhcp", mcp.Description("Use DHCP (default: true)")),
		mcp.WithString("ip", mcp.Description("Static IPv4 with prefix, e.g. 192.168.1.100/24 (ignored if dhcp=true)")),
		mcp.WithString("gateway", mcp.Description("Default gateway (ignored if dhcp=true)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ifname, err := req.RequireString("interface")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ssid, err := req.RequireString("ssid")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ok, err := client.WiFiManager.SetClientConfig(
			ifname, ssid,
			getStringArg(req, "password", ""),
			getStringArg(req, "security", "WPA2"),
			getBoolArg(req, "dhcp", true),
			getStringArg(req, "ip", ""),
			getStringArg(req, "gateway", ""),
			nil,
		)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{"ok": ok}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_wifi_connect",
		mcp.WithDescription("Connect to Wi-Fi using the stored client config."),
		mcp.WithString("interface", mcp.Required(), mcp.Description("Interface name, e.g. wlan0")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ifname, err := req.RequireString("interface")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ok, err := client.WiFiManager.Connect(ifname)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{"ok": ok}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_wifi_disconnect",
		mcp.WithDescription("Disconnect from the current Wi-Fi network."),
		mcp.WithString("interface", mcp.Required(), mcp.Description("Interface name, e.g. wlan0")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ifname, err := req.RequireString("interface")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ok, err := client.WiFiManager.Disconnect(ifname)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{"ok": ok}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_wifi_start_ap",
		mcp.WithDescription("Start a Wi-Fi access point on an interface."),
		mcp.WithString("interface", mcp.Required(), mcp.Description("Interface name, e.g. wlan0")),
		mcp.WithString("ssid", mcp.Required(), mcp.Description("AP network name")),
		mcp.WithString("password", mcp.Required(), mcp.Description("AP password (min 8 chars for WPA2)")),
		mcp.WithString("band", mcp.Description("Frequency band: 2.4GHz (default) or 5GHz")),
		mcp.WithNumber("channel", mcp.Description("Channel number (0 = auto, default)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ifname, err := req.RequireString("interface")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ssid, err := req.RequireString("ssid")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		password, err := req.RequireString("password")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ok, err := client.WiFiManager.StartAP(
			ifname, ssid, password,
			getStringArg(req, "band", "2.4GHz"),
			int32(req.GetInt("channel", 0)),
		)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{"ok": ok}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_wifi_stop_ap",
		mcp.WithDescription("Stop the Wi-Fi access point."),
		mcp.WithString("interface", mcp.Required(), mcp.Description("Interface name, e.g. wlan0")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ifname, err := req.RequireString("interface")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ok, err := client.WiFiManager.StopAP(ifname)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{"ok": ok}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})
}

func registerEthernetTools(s *server.MCPServer, client *orbitclient.Client) {
	s.AddTool(mcp.NewTool("orbit_ethernet_list_interfaces",
		mcp.WithDescription("List all Ethernet interfaces and their link properties."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ifaces, err := client.EthernetManager.ListEthernetInterfaces()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		var out []map[string]any
		for _, iface := range ifaces {
			b, _ := protoJSON.Marshal(iface)
			item := map[string]any{}
			_ = json.Unmarshal(b, &item)
			out = append(out, item)
		}
		if out == nil {
			out = []map[string]any{}
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_ethernet_status",
		mcp.WithDescription("Get link status and properties for an Ethernet interface."),
		mcp.WithString("interface", mcp.Required(), mcp.Description("Interface name, e.g. eth0")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ifname, err := req.RequireString("interface")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		connected, _ := client.EthernetManager.IsEthernetConnected(ifname)
		props, err := client.EthernetManager.GetEthernetLinkProperties(ifname)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		pb, _ := protoJSON.Marshal(props)
		propsMap := map[string]any{}
		_ = json.Unmarshal(pb, &propsMap)
		b, _ := json.MarshalIndent(map[string]any{
			"connected":  connected,
			"properties": propsMap,
		}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_ethernet_enable",
		mcp.WithDescription("Bring up an Ethernet interface."),
		mcp.WithString("interface", mcp.Required(), mcp.Description("Interface name, e.g. eth0")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ifname, err := req.RequireString("interface")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ok, err := client.EthernetManager.EnableEthernet(ifname)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{"ok": ok}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_ethernet_disable",
		mcp.WithDescription("Bring down an Ethernet interface."),
		mcp.WithString("interface", mcp.Required(), mcp.Description("Interface name, e.g. eth0")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ifname, err := req.RequireString("interface")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ok, err := client.EthernetManager.DisableEthernet(ifname)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{"ok": ok}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_ethernet_set_config",
		mcp.WithDescription("Configure Ethernet interface IP settings (DHCP or static)."),
		mcp.WithString("interface", mcp.Required(), mcp.Description("Interface name, e.g. eth0")),
		mcp.WithBoolean("dhcp", mcp.Description("Use DHCP (default: true)")),
		mcp.WithString("ip", mcp.Description("Static IPv4 with prefix, e.g. 192.168.1.100/24 (ignored if dhcp=true)")),
		mcp.WithString("gateway", mcp.Description("Default gateway (ignored if dhcp=true)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ifname, err := req.RequireString("interface")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ok, err := client.EthernetManager.SetEthernetConfig(
			ifname, true,
			getBoolArg(req, "dhcp", true),
			getStringArg(req, "ip", ""),
			getStringArg(req, "gateway", ""),
			nil,
		)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{"ok": ok}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})
}
