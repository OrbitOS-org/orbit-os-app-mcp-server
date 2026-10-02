package main

import (
	"context"
	"encoding/json"

	orbitclient "github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerPowerTools(s *server.MCPServer, client *orbitclient.Client) {
	s.AddTool(mcp.NewTool("orbit_power_reboot",
		mcp.WithDescription("Request system reboot."),
		mcp.WithBoolean("force", mcp.Description("Force reboot.")),
		mcp.WithString("reason", mcp.Description("Optional reboot reason.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := client.PowerManager.Reboot(getBoolArg(req, "force", false), getStringArg(req, "reason", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{"ok": result.Success, "message": result.Message}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_power_shutdown",
		mcp.WithDescription("Request system shutdown."),
		mcp.WithBoolean("force", mcp.Description("Force shutdown.")),
		mcp.WithString("reason", mcp.Description("Optional shutdown reason.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := client.PowerManager.Shutdown(getBoolArg(req, "force", false), getStringArg(req, "reason", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{"ok": result.Success, "message": result.Message}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})
}
