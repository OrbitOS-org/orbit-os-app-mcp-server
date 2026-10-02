package main

import (
	"context"
	"encoding/json"
	"fmt"

	orbitclient "github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerSystemTools(s *server.MCPServer, client *orbitclient.Client) {
	addSystemStringTool(s, "orbit_system_api_version_info", "Get API version info string.", client.SystemManager.GetApiVersionInfo)
	addSystemStringTool(s, "orbit_system_device_name", "Get device name.", client.SystemManager.GetDeviceName)
	addSystemStringTool(s, "orbit_system_soc_model", "Get SoC model.", client.SystemManager.GetSocModel)
	addSystemStringTool(s, "orbit_system_soc_vendor", "Get SoC vendor.", client.SystemManager.GetSocVendor)
	addSystemStringTool(s, "orbit_system_board_model", "Get board model.", client.SystemManager.GetBoardModel)
	addSystemStringTool(s, "orbit_system_board_vendor", "Get board vendor.", client.SystemManager.GetBoardVendor)
	addSystemStringTool(s, "orbit_system_hardware_version", "Get hardware version.", client.SystemManager.GetHardwareVersion)
	addSystemStringTool(s, "orbit_system_hardware_model", "Get hardware model.", client.SystemManager.GetHardwareModel)
	addSystemStringTool(s, "orbit_system_system_uuid", "Get system UUID.", client.SystemManager.GetSystemUuid)
	addSystemStringTool(s, "orbit_system_board_serial", "Get board serial.", client.SystemManager.GetBoardSerial)
	addSystemStringTool(s, "orbit_system_cpu_serial", "Get CPU serial.", client.SystemManager.GetCpuSerial)
	addSystemStringTool(s, "orbit_system_machine_id", "Get machine ID.", client.SystemManager.GetMachineId)
	addSystemStringTool(s, "orbit_system_architecture", "Get CPU architecture.", client.SystemManager.GetArchitecture)
	addSystemStringTool(s, "orbit_system_cpu_model", "Get CPU model.", client.SystemManager.GetCpuModel)
	addSystemStringTool(s, "orbit_system_os_name", "Get OS name.", client.SystemManager.GetOsName)
	addSystemStringTool(s, "orbit_system_os_version", "Get OS version.", client.SystemManager.GetOsVersion)
	addSystemStringTool(s, "orbit_system_kernel_version", "Get kernel version.", client.SystemManager.GetKernelVersion)
	addSystemStringTool(s, "orbit_system_distro", "Get distro name.", client.SystemManager.GetDistro)
	addSystemStringTool(s, "orbit_system_distro_version", "Get distro version.", client.SystemManager.GetDistroVersion)
	addSystemStringTool(s, "orbit_system_runtime_version", "Get runtime version.", client.SystemManager.GetRuntimeVersion)
	addSystemStringTool(s, "orbit_system_os_revision", "Get OS revision.", client.SystemManager.GetOsRevision)
	addSystemStringTool(s, "orbit_system_build_version", "Get build version alias.", client.SystemManager.GetBuildVersion)
	addSystemStringTool(s, "orbit_system_runtime_build_date", "Get runtime build date.", client.SystemManager.GetRuntimeBuildDate)
	addSystemStringTool(s, "orbit_system_build_date", "Get build date alias.", client.SystemManager.GetBuildDate)

	addSystemInt64Tool(s, "orbit_system_cpu_cores", "Get CPU cores count.", client.SystemManager.GetCpuCores)
	addSystemInt64Tool(s, "orbit_system_cpu_threads", "Get CPU threads count.", client.SystemManager.GetCpuThreads)
	addSystemFloat64Tool(s, "orbit_system_cpu_min_mhz", "Get CPU minimum MHz.", client.SystemManager.GetCpuMinMhz)
	addSystemFloat64Tool(s, "orbit_system_cpu_max_mhz", "Get CPU maximum MHz.", client.SystemManager.GetCpuMaxMhz)
	addSystemUint64Tool(s, "orbit_system_total_ram_bytes", "Get total RAM bytes.", client.SystemManager.GetTotalRAM)

	s.AddTool(mcp.NewTool("orbit_system_api_version",
		mcp.WithDescription("Get API version and revision as JSON."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		version, revision, err := client.SystemManager.GetApiVersion()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]int64{"version": version, "revision": revision}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_system_metrics",
		mcp.WithDescription("Get full SystemService metrics response (protobuf JSON)."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		metrics, err := client.SystemManager.GetMetrics()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, err := protoJSON.Marshal(metrics)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_system_device_summary",
		mcp.WithDescription("Get basic device summary (name, architecture, OS)."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		deviceName, err := client.SystemManager.GetDeviceName()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		arch, err := client.SystemManager.GetArchitecture()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		osName, err := client.SystemManager.GetOsName()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		osVersion, err := client.SystemManager.GetOsVersion()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]string{
			"device_name":  deviceName,
			"architecture": arch,
			"os_name":      osName,
			"os_version":   osVersion,
		}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})
}

func addSystemStringTool(s *server.MCPServer, name, description string, fn func() (string, error)) {
	s.AddTool(mcp.NewTool(name, mcp.WithDescription(description)),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			val, err := fn()
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(val), nil
		},
	)
}

func addSystemInt64Tool(s *server.MCPServer, name, description string, fn func() (int64, error)) {
	s.AddTool(mcp.NewTool(name, mcp.WithDescription(description)),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			val, err := fn()
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("%d", val)), nil
		},
	)
}

func addSystemUint64Tool(s *server.MCPServer, name, description string, fn func() (uint64, error)) {
	s.AddTool(mcp.NewTool(name, mcp.WithDescription(description)),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			val, err := fn()
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("%d", val)), nil
		},
	)
}

func addSystemFloat64Tool(s *server.MCPServer, name, description string, fn func() (float64, error)) {
	s.AddTool(mcp.NewTool(name, mcp.WithDescription(description)),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			val, err := fn()
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("%f", val)), nil
		},
	)
}
