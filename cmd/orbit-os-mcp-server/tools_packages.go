package main

import (
	"context"
	"encoding/json"
	"strings"

	orbitclient "github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerPackageTools(s *server.MCPServer, client *orbitclient.Client) {
	s.AddTool(mcp.NewTool("orbit_package_list_installed",
		mcp.WithDescription("List installed packages from PackageManagerService."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pkgs, err := client.PackageManager.ListInstalledPackages()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		out := make([]map[string]any, 0, len(pkgs))
		for _, p := range pkgs {
			if p == nil {
				continue
			}
			b, err := protoJSON.Marshal(p)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			item := map[string]any{}
			if err := json.Unmarshal(b, &item); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			out = append(out, item)
		}
		b, _ := json.MarshalIndent(map[string]any{"packages": out}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_package_install_from_file",
		mcp.WithDescription("Install a .orb from a path on the device. For files on your PC use orbit_package_upload_begin / orbit_package_upload_chunk / orbit_package_upload_install instead."),
		mcp.WithString("file_path",
			mcp.Required(),
			mcp.Description("Absolute path to .orb on the device"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path, err := req.RequireString("file_path")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if !strings.HasSuffix(strings.ToLower(path), ".orb") {
			return mcp.NewToolResultError("file_path must point to a .orb file"), nil
		}
		if err := client.PackageManager.InstallPackage(ctx, path); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(`{"ok": true}`), nil
	})

	s.AddTool(mcp.NewTool("orbit_package_remove",
		mcp.WithDescription("Remove an installed package by package_id."),
		mcp.WithString("package_id",
			mcp.Required(),
			mcp.Description("package_id as returned by orbit_package_list_installed"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		packageID, err := req.RequireString("package_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if err := client.PackageManager.RemovePackage(ctx, packageID); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(`{"ok": true}`), nil
	})
}
