package main

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	orbitclient "github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerBluetoothTools(s *server.MCPServer, client *orbitclient.Client) {
	s.AddTool(mcp.NewTool("orbit_bluetooth_adapter_info",
		mcp.WithDescription("Get local Bluetooth adapter info (address, name, powered, discoverable)."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		info, err := client.BluetoothManager.GetAdapterInfo(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{
			"address":      info.Address,
			"name":         info.Name,
			"powered":      info.Powered,
			"discoverable": info.Discoverable,
			"discovering":  info.Discovering,
		}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_bluetooth_enable",
		mcp.WithDescription("Power on the Bluetooth adapter."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ok, err := client.BluetoothManager.EnableBluetooth(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{"ok": ok}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_bluetooth_disable",
		mcp.WithDescription("Power off the Bluetooth adapter."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ok, err := client.BluetoothManager.DisableBluetooth(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{"ok": ok}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_bluetooth_set_name",
		mcp.WithDescription("Set the local Bluetooth adapter advertising name."),
		mcp.WithString("name", mcp.Required(), mcp.Description("New adapter name")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := req.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ok, err := client.BluetoothManager.SetLocalName(ctx, name)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{"ok": ok}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_bluetooth_scan_classic",
		mcp.WithDescription("Scan for classic Bluetooth devices for timeout_sec seconds and return all found devices."),
		mcp.WithNumber("timeout_sec", mcp.Description("Scan duration in seconds (default: 10, max: 30)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		timeoutSec := req.GetInt("timeout_sec", 10)
		if timeoutSec > 30 {
			timeoutSec = 30
		}
		scanCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
		defer cancel()

		var mu sync.Mutex
		var devices []map[string]any
		_ = client.BluetoothManager.ScanClassic(scanCtx, func(dev orbitclient.BtDevice) {
			mu.Lock()
			devices = append(devices, map[string]any{
				"address": dev.Address,
				"name":    dev.Name,
				"type":    dev.Type,
				"bonded":  dev.Bonded,
				"rssi":    dev.RSSI,
			})
			mu.Unlock()
		})
		if devices == nil {
			devices = []map[string]any{}
		}
		b, _ := json.MarshalIndent(map[string]any{"devices": devices}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_bluetooth_scan_ble",
		mcp.WithDescription("Scan for BLE devices for timeout_sec seconds. Deduplicates by address."),
		mcp.WithNumber("timeout_sec", mcp.Description("Scan duration in seconds (default: 10, max: 30)")),
		mcp.WithString("name_prefix", mcp.Description("Filter by device name prefix (optional)")),
		mcp.WithString("service_uuid", mcp.Description("Filter by service UUID (optional)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		timeoutSec := req.GetInt("timeout_sec", 10)
		if timeoutSec > 30 {
			timeoutSec = 30
		}
		scanCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
		defer cancel()

		filter := orbitclient.BLEScanFilter{
			NamePrefix:  getStringArg(req, "name_prefix", ""),
			ServiceUUID: getStringArg(req, "service_uuid", ""),
		}
		var filters []orbitclient.BLEScanFilter
		if filter.NamePrefix != "" || filter.ServiceUUID != "" {
			filters = append(filters, filter)
		}

		var mu sync.Mutex
		seen := map[string]bool{}
		var devices []map[string]any
		_ = client.BluetoothManager.ScanBLE(scanCtx, filters, func(r orbitclient.BLEScanResult) {
			mu.Lock()
			if !seen[r.Address] {
				seen[r.Address] = true
				devices = append(devices, map[string]any{
					"address":       r.Address,
					"name":          r.Name,
					"rssi":          r.RSSI,
					"connectable":   r.Connectable,
					"service_uuids": r.ServiceUUIDs,
				})
			}
			mu.Unlock()
		})
		if devices == nil {
			devices = []map[string]any{}
		}
		b, _ := json.MarshalIndent(map[string]any{"devices": devices}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_bluetooth_bonded_devices",
		mcp.WithDescription("List all paired/bonded Bluetooth devices."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		devices, err := client.BluetoothManager.GetBondedDevices(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		var out []map[string]any
		for _, dev := range devices {
			out = append(out, map[string]any{
				"address": dev.Address,
				"name":    dev.Name,
				"type":    dev.Type,
				"rssi":    dev.RSSI,
			})
		}
		if out == nil {
			out = []map[string]any{}
		}
		b, _ := json.MarshalIndent(map[string]any{"devices": out}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_bluetooth_connect",
		mcp.WithDescription("Connect to a bonded Bluetooth device."),
		mcp.WithString("address", mcp.Required(), mcp.Description("Device MAC address, e.g. AA:BB:CC:DD:EE:FF")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		addr, err := req.RequireString("address")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ok, err := client.BluetoothManager.ConnectDevice(ctx, addr)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{"ok": ok}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_bluetooth_disconnect",
		mcp.WithDescription("Disconnect a connected Bluetooth device."),
		mcp.WithString("address", mcp.Required(), mcp.Description("Device MAC address")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		addr, err := req.RequireString("address")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ok, err := client.BluetoothManager.DisconnectDevice(ctx, addr)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{"ok": ok}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_bluetooth_connection_state",
		mcp.WithDescription("Get the connection state of a Bluetooth device (connected, connecting, disconnecting, disconnected)."),
		mcp.WithString("address", mcp.Required(), mcp.Description("Device MAC address")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		addr, err := req.RequireString("address")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		state, err := client.BluetoothManager.GetConnectionState(ctx, addr)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{"address": addr, "state": state}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})
}
