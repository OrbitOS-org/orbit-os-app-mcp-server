package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	orbitclient "github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// parseHexData parses a hex string — accepts spaces, "0x" prefixes, and commas.
func parseHexData(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "0x", "")
	s = strings.ReplaceAll(s, "0X", "")
	if s == "" {
		return nil, nil
	}
	return hex.DecodeString(s)
}

// bytesResult returns a map with hex-encoded bytes and, if valid UTF-8, a text field.
func bytesResult(b []byte) map[string]any {
	out := map[string]any{"hex": hex.EncodeToString(b)}
	if utf8.Valid(b) {
		out["text"] = string(b)
	}
	return out
}

// ── UART ──────────────────────────────────────────────────────────────────────

func registerUARTTools(s *server.MCPServer, client *orbitclient.Client) {
	s.AddTool(mcp.NewTool("orbit_uart_list_ports",
		mcp.WithDescription("List available UART ports on the device."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ports, err := client.UartManager.ListPorts()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{"ports": ports}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_uart_write",
		mcp.WithDescription("Open a UART port, write data, and close. Use data_text for ASCII or data_hex for binary."),
		mcp.WithString("port", mcp.Required(), mcp.Description("Port name from orbit_uart_list_ports, e.g. ttyAMA0")),
		mcp.WithNumber("baudrate", mcp.Description("Baud rate (default: 9600)")),
		mcp.WithString("data_text", mcp.Description("Data to send as plain text")),
		mcp.WithString("data_hex", mcp.Description("Data to send as hex, e.g. '48 65 6c 6c 6f' (priority over data_text)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		portName, err := req.RequireString("port")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		var data []byte
		if hexStr := getStringArg(req, "data_hex", ""); hexStr != "" {
			data, err = parseHexData(hexStr)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("data_hex: %v", err)), nil
			}
		} else if text := getStringArg(req, "data_text", ""); text != "" {
			data = []byte(text)
		} else {
			return mcp.NewToolResultError("provide data_text or data_hex"), nil
		}
		port, err := client.UartManager.Open(orbitclient.UartConfig{
			Port: portName, Baudrate: req.GetInt("baudrate", 9600), DataBits: 8,
		})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		defer port.Close()
		n, err := port.Write(data)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{"bytes_written": n}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_uart_read",
		mcp.WithDescription("Open a UART port, collect incoming data for timeout_ms, and close."),
		mcp.WithString("port", mcp.Required(), mcp.Description("Port name from orbit_uart_list_ports")),
		mcp.WithNumber("baudrate", mcp.Description("Baud rate (default: 9600)")),
		mcp.WithNumber("timeout_ms", mcp.Description("Read window in milliseconds (default: 1000)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		portName, err := req.RequireString("port")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		port, err := client.UartManager.Open(orbitclient.UartConfig{
			Port: portName, Baudrate: req.GetInt("baudrate", 9600), DataBits: 8,
		})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		defer port.Close()
		readCtx, cancel := context.WithTimeout(ctx, time.Duration(req.GetInt("timeout_ms", 1000))*time.Millisecond)
		defer cancel()
		ch, err := port.ListenAsync(readCtx, 4096)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		var buf []byte
		for chunk := range ch {
			buf = append(buf, chunk...)
		}
		result := bytesResult(buf)
		result["bytes_received"] = len(buf)
		b, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_uart_transact",
		mcp.WithDescription("Write to a UART port then read the response. Useful for AT commands and sensor queries."),
		mcp.WithString("port", mcp.Required(), mcp.Description("Port name from orbit_uart_list_ports")),
		mcp.WithNumber("baudrate", mcp.Description("Baud rate (default: 9600)")),
		mcp.WithString("data_text", mcp.Description("Command to send as plain text")),
		mcp.WithString("data_hex", mcp.Description("Command to send as hex (priority over data_text)")),
		mcp.WithNumber("timeout_ms", mcp.Description("Time to wait for response in milliseconds (default: 1000)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		portName, err := req.RequireString("port")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		var data []byte
		if hexStr := getStringArg(req, "data_hex", ""); hexStr != "" {
			data, err = parseHexData(hexStr)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("data_hex: %v", err)), nil
			}
		} else if text := getStringArg(req, "data_text", ""); text != "" {
			data = []byte(text)
		} else {
			return mcp.NewToolResultError("provide data_text or data_hex"), nil
		}
		port, err := client.UartManager.Open(orbitclient.UartConfig{
			Port: portName, Baudrate: req.GetInt("baudrate", 9600), DataBits: 8,
		})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		defer port.Close()
		if _, err := port.Write(data); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		readCtx, cancel := context.WithTimeout(ctx, time.Duration(req.GetInt("timeout_ms", 1000))*time.Millisecond)
		defer cancel()
		ch, err := port.ListenAsync(readCtx, 4096)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		var buf []byte
		for chunk := range ch {
			buf = append(buf, chunk...)
		}
		result := bytesResult(buf)
		result["bytes_received"] = len(buf)
		b, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})
}

// ── I2C ───────────────────────────────────────────────────────────────────────

func registerI2CTools(s *server.MCPServer, client *orbitclient.Client) {
	s.AddTool(mcp.NewTool("orbit_i2c_list_buses",
		mcp.WithDescription("List available I2C bus numbers on the device."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		buses, err := client.I2CManager.ListBuses()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{"buses": buses}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_i2c_scan",
		mcp.WithDescription("Scan an I2C bus and return addresses of all responding devices (probes 0x03–0x77)."),
		mcp.WithNumber("bus", mcp.Required(), mcp.Description("I2C bus number, e.g. 1 for /dev/i2c-1")),
		mcp.WithNumber("clock_hz", mcp.Description("Clock frequency in Hz (default: 100000)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		bus, err := req.RequireInt("bus")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		i2cBus, err := client.I2CManager.Open(uint32(bus), uint32(req.GetInt("clock_hz", 100000)), false, false)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		addrs, err := i2cBus.Scan()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		hexAddrs := make([]string, len(addrs))
		for i, a := range addrs {
			hexAddrs[i] = fmt.Sprintf("0x%02X", a)
		}
		b, _ := json.MarshalIndent(map[string]any{
			"bus":       bus,
			"addresses": hexAddrs,
			"count":     len(hexAddrs),
		}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_i2c_transfer",
		mcp.WithDescription("Perform an I2C write, read, or write-then-read. Omit data_hex for read-only; set read_length=0 for write-only."),
		mcp.WithNumber("bus", mcp.Required(), mcp.Description("I2C bus number")),
		mcp.WithString("address", mcp.Required(), mcp.Description("Device 7-bit address, e.g. '0x48' or '72'")),
		mcp.WithString("data_hex", mcp.Description("Bytes to write as hex, e.g. '0x01 0x00' (omit for read-only)")),
		mcp.WithNumber("read_length", mcp.Description("Bytes to read back (0 = write-only, default: 0)")),
		mcp.WithNumber("clock_hz", mcp.Description("Clock frequency in Hz (default: 100000)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		bus, err := req.RequireInt("bus")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		addrStr, err := req.RequireString("address")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		addr, err := strconv.ParseInt(strings.TrimSpace(addrStr), 0, 32)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid address %q (use decimal '72' or hex '0x48')", addrStr)), nil
		}
		var writeData []byte
		if hexStr := getStringArg(req, "data_hex", ""); hexStr != "" {
			writeData, err = parseHexData(hexStr)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("data_hex: %v", err)), nil
			}
		}
		readLen := req.GetInt("read_length", 0)
		if len(writeData) == 0 && readLen == 0 {
			return mcp.NewToolResultError("provide data_hex to write and/or read_length > 0 to read"), nil
		}
		i2cBus, err := client.I2CManager.Open(uint32(bus), uint32(req.GetInt("clock_hz", 100000)), false, false)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		rxData, err := i2cBus.Transfer(uint32(addr), writeData, uint32(readLen), 0)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		result := map[string]any{
			"bus":        bus,
			"address":    fmt.Sprintf("0x%02X", addr),
			"bytes_sent": len(writeData),
		}
		if len(rxData) > 0 {
			result["received"] = bytesResult(rxData)
		}
		b, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})
}

// ── SPI ───────────────────────────────────────────────────────────────────────

func registerSPITools(s *server.MCPServer, client *orbitclient.Client) {
	s.AddTool(mcp.NewTool("orbit_spi_list_devices",
		mcp.WithDescription("List available SPI devices on the device (e.g. spidev0.0)."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		devices, err := client.SpiManager.ListDevices()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{"devices": devices}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_spi_transfer",
		mcp.WithDescription("Perform a full-duplex SPI transfer. Sends data_hex on MOSI and returns read_length bytes from MISO."),
		mcp.WithNumber("bus", mcp.Required(), mcp.Description("SPI bus number (e.g. 0 for spidev0.x)")),
		mcp.WithNumber("chip_select", mcp.Required(), mcp.Description("Chip select (e.g. 0 for spidev0.0)")),
		mcp.WithString("data_hex", mcp.Required(), mcp.Description("Bytes to send as hex, e.g. '0x01 0x02'")),
		mcp.WithNumber("read_length", mcp.Description("Bytes to read from MISO (default: same as data length)")),
		mcp.WithNumber("speed_hz", mcp.Description("Max clock speed in Hz (default: 1000000)")),
		mcp.WithNumber("mode", mcp.Description("SPI mode 0-3 (default: 0)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		bus, err := req.RequireInt("bus")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		cs, err := req.RequireInt("chip_select")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		hexStr, err := req.RequireString("data_hex")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		dataOut, err := parseHexData(hexStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("data_hex: %v", err)), nil
		}
		readLen := req.GetInt("read_length", len(dataOut))
		dev, err := client.SpiManager.Open(uint32(bus), uint32(cs), uint32(req.GetInt("speed_hz", 1000000)), 8, req.GetInt("mode", 0), false)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		rxData, err := dev.Transfer(dataOut, uint32(readLen))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		result := map[string]any{
			"device":     fmt.Sprintf("spidev%d.%d", bus, cs),
			"bytes_sent": len(dataOut),
		}
		if len(rxData) > 0 {
			result["received"] = bytesResult(rxData)
		}
		b, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})
}

// ── PWM ───────────────────────────────────────────────────────────────────────

func registerPWMTools(s *server.MCPServer, client *orbitclient.Client) {
	s.AddTool(mcp.NewTool("orbit_pwm_list_channels",
		mcp.WithDescription("List all PWM channels available on the device."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		channels, err := client.PwmManager.ListChannels()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		var out []map[string]any
		for _, ch := range channels {
			out = append(out, map[string]any{"channel": ch.Channel, "name": ch.Name})
		}
		if out == nil {
			out = []map[string]any{}
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_pwm_get",
		mcp.WithDescription("Get current PWM properties (enabled, duty cycle, frequency) for a channel."),
		mcp.WithNumber("channel", mcp.Required(), mcp.Description("PWM channel number from orbit_pwm_list_channels")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ch, err := req.RequireInt("channel")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		props, err := client.PwmManager.GetProperties(&orbitclient.PwmChannel{Channel: uint32(ch)})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{
			"channel":      props.Channel.Channel,
			"enabled":      props.Enabled,
			"duty_cycle":   props.DutyCycle,
			"frequency_hz": props.FrequencyHz,
		}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_pwm_set",
		mcp.WithDescription("Configure duty cycle and frequency on a PWM channel and start output."),
		mcp.WithNumber("channel", mcp.Required(), mcp.Description("PWM channel number")),
		mcp.WithNumber("duty_cycle", mcp.Required(), mcp.Description("Duty cycle from 0.0 to 1.0 (e.g. 0.5 = 50%)")),
		mcp.WithNumber("frequency_hz", mcp.Required(), mcp.Description("Output frequency in Hz (e.g. 1000 for 1 kHz)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ch, err := req.RequireInt("channel")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		args := req.GetArguments()
		dutyCycle, ok1 := args["duty_cycle"].(float64)
		freqHz, ok2 := args["frequency_hz"].(float64)
		if !ok1 || !ok2 {
			return mcp.NewToolResultError("duty_cycle and frequency_hz must be numbers"), nil
		}
		if dutyCycle < 0 || dutyCycle > 1 {
			return mcp.NewToolResultError("duty_cycle must be between 0.0 and 1.0"), nil
		}
		if err := client.PwmManager.SetPwm(&orbitclient.PwmChannel{Channel: uint32(ch)}, dutyCycle, freqHz); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{
			"ok": true, "channel": ch, "duty_cycle": dutyCycle, "frequency_hz": freqHz,
		}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_pwm_stop",
		mcp.WithDescription("Stop PWM output on a channel."),
		mcp.WithNumber("channel", mcp.Required(), mcp.Description("PWM channel number")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ch, err := req.RequireInt("channel")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if err := client.PwmManager.StopPwm(&orbitclient.PwmChannel{Channel: uint32(ch)}); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(map[string]any{"ok": true, "channel": ch}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})
}
