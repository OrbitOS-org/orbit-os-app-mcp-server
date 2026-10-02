package main

import (
	"context"
	"fmt"

	orbitclient "github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerUpdateTools(s *server.MCPServer, client *orbitclient.Client) {
	s.AddTool(mcp.NewTool("orbit_update_install_ota_from_file",
		mcp.WithDescription("Install OTA update from a local .orbit file path on the device."),
		mcp.WithString("file_path",
			mcp.Required(),
			mcp.Description("Absolute path to OTA image file on the device"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path, err := req.RequireString("file_path")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if err := client.UpdateManager.Update(ctx, path); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(`{"ok": true}`), nil
	})

	s.AddTool(mcp.NewTool("orbit_update_factory_reset",
		mcp.WithDescription("Trigger factory reset and reboot."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ok, err := client.UpdateManager.FactoryReset()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf(`{"ok": %t}`, ok)), nil
	})
}
