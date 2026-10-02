package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	orbitclient "github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerGPIOTools(s *server.MCPServer, client *orbitclient.Client) {
	s.AddTool(mcp.NewTool("orbit_gpio_list_pins",
		mcp.WithDescription("List GPIO lines (name, line offset, gpiochip index)."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pins, err := client.GpioManager.ListPins()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		var out []map[string]any
		for _, p := range pins {
			out = append(out, map[string]any{
				"name":     p.Name,
				"line":     p.Number,
				"gpiochip": p.ChipNumber,
			})
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_gpio_get_level",
		mcp.WithDescription("Read logical level. Use `name` (e.g. GPIO17) OR `line` + optional `gpiochip`."),
		mcp.WithString("name", mcp.Description("Line name from orbit_gpio_list_pins (alternative to line+gpiochip).")),
		mcp.WithNumber("line", mcp.Description("Line offset within the chip; required if name is omitted.")),
		mcp.WithNumber("gpiochip", mcp.Description("GPIO chip index (default 0).")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pin, err := gpioPinFromReq(client, req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		lvl, err := client.GpioManager.GetLevel(pin)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{"level": gpioLevelString(lvl), "raw": int(lvl)}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_gpio_set_level",
		mcp.WithDescription("Drive a GPIO output. Pin: `name` OR `line` + optional `gpiochip`. Level: low, high, 0, or 1."),
		mcp.WithString("name", mcp.Description("Line name from orbit_gpio_list_pins (alternative to line+gpiochip).")),
		mcp.WithNumber("line", mcp.Description("Line offset within the chip; required if name is omitted.")),
		mcp.WithNumber("gpiochip", mcp.Description("GPIO chip index (default 0).")),
		mcp.WithString("level", mcp.Required(), mcp.Description(`Logical level: "low", "high", "0", "1", or numeric 0/1.`)),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pin, err := gpioPinFromReq(client, req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		lvl, err := gpioLevelFromReq(req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if err := client.GpioManager.SetLevel(pin, lvl); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(`{"ok": true}`), nil
	})

	s.AddTool(mcp.NewTool("orbit_gpio_get_direction",
		mcp.WithDescription("Read direction (in/out). Use `name` OR `line` + optional `gpiochip`."),
		mcp.WithString("name", mcp.Description("Line name from orbit_gpio_list_pins (alternative to line+gpiochip).")),
		mcp.WithNumber("line", mcp.Description("Line offset within the chip; required if name is omitted.")),
		mcp.WithNumber("gpiochip", mcp.Description("GPIO chip index (default 0).")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pin, err := gpioPinFromReq(client, req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		dir, err := client.GpioManager.GetDirection(pin)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{"direction": gpioDirectionString(dir), "raw": int(dir)}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_gpio_set_direction",
		mcp.WithDescription(`Set GPIO direction: "in" or "out" (aliases: input, output).`),
		mcp.WithNumber("line", mcp.Required(), mcp.Description("Line offset within the chip.")),
		mcp.WithNumber("gpiochip", mcp.Description("GPIO chip index (default 0).")),
		mcp.WithString("direction", mcp.Required(), mcp.Description(`"in" or "out".`)),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pin, err := gpioPinFromReq(client, req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		dirStr, err := req.RequireString("direction")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		dir, err := parseGpioDirection(dirStr)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if err := client.GpioManager.SetDirection(pin, dir); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(`{"ok": true}`), nil
	})
}

func gpioPinFromReq(client *orbitclient.Client, req mcp.CallToolRequest) (*orbitclient.GpioPin, error) {
	name := strings.TrimSpace(req.GetString("name", ""))
	if name != "" {
		return resolveGpioPinByName(client, name)
	}
	line, err := req.RequireInt("line")
	if err != nil {
		return nil, fmt.Errorf("provide `name` or `line` (and optional gpiochip): %w", err)
	}
	return &orbitclient.GpioPin{Number: int32(line), ChipNumber: int32(req.GetInt("gpiochip", 0))}, nil
}

func resolveGpioPinByName(client *orbitclient.Client, want string) (*orbitclient.GpioPin, error) {
	pins, err := client.GpioManager.ListPins()
	if err != nil {
		return nil, err
	}
	for _, p := range pins {
		if p.Name == want || strings.EqualFold(p.Name, want) {
			return p, nil
		}
	}
	return nil, fmt.Errorf("unknown GPIO name %q (use orbit_gpio_list_pins)", want)
}

func gpioLevelFromReq(req mcp.CallToolRequest) (orbitclient.GpioLevel, error) {
	args := req.GetArguments()
	if args == nil {
		return 0, fmt.Errorf(`required argument "level" not found`)
	}
	v, ok := args["level"]
	if !ok || v == nil {
		return 0, fmt.Errorf(`required argument "level" not found`)
	}
	switch x := v.(type) {
	case string:
		return parseGpioLevel(x)
	case float64:
		switch int(x) {
		case 0:
			return orbitclient.GPIO_LEVEL_LOW, nil
		case 1:
			return orbitclient.GPIO_LEVEL_HIGH, nil
		default:
			return 0, fmt.Errorf("invalid level number %v (use 0 or 1)", x)
		}
	case int:
		switch x {
		case 0:
			return orbitclient.GPIO_LEVEL_LOW, nil
		case 1:
			return orbitclient.GPIO_LEVEL_HIGH, nil
		default:
			return 0, fmt.Errorf("invalid level number %v (use 0 or 1)", x)
		}
	case bool:
		if x {
			return orbitclient.GPIO_LEVEL_HIGH, nil
		}
		return orbitclient.GPIO_LEVEL_LOW, nil
	default:
		return 0, fmt.Errorf("level must be string, number, or boolean")
	}
}

func parseGpioLevel(s string) (orbitclient.GpioLevel, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "low", "0", "false":
		return orbitclient.GPIO_LEVEL_LOW, nil
	case "high", "1", "true":
		return orbitclient.GPIO_LEVEL_HIGH, nil
	default:
		return 0, fmt.Errorf("invalid level %q (use low, high, 0, or 1)", s)
	}
}

func parseGpioDirection(s string) (orbitclient.GpioDirection, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "in", "input":
		return orbitclient.GPIO_DIR_IN, nil
	case "out", "output":
		return orbitclient.GPIO_DIR_OUT, nil
	default:
		return 0, fmt.Errorf("invalid direction %q (use in or out)", s)
	}
}

func gpioLevelString(l orbitclient.GpioLevel) string {
	switch l {
	case orbitclient.GPIO_LEVEL_LOW:
		return "low"
	case orbitclient.GPIO_LEVEL_HIGH:
		return "high"
	default:
		return fmt.Sprintf("unknown(%d)", l)
	}
}

func gpioDirectionString(d orbitclient.GpioDirection) string {
	switch d {
	case orbitclient.GPIO_DIR_OUT:
		return "out"
	case orbitclient.GPIO_DIR_IN:
		return "in"
	default:
		return fmt.Sprintf("unknown(%d)", d)
	}
}
