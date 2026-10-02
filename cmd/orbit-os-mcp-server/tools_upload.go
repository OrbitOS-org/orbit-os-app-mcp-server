package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	orbitclient "github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type uploadSession struct {
	filename    string
	totalChunks int
	received    map[int]struct{}
	dir         string
	createdAt   time.Time
}

var (
	uploadMu       sync.Mutex
	uploadSessions = map[string]*uploadSession{}
)

func cleanStaleUploads() {
	cutoff := time.Now().Add(-2 * time.Hour)
	for id, sess := range uploadSessions {
		if sess.createdAt.Before(cutoff) {
			_ = os.RemoveAll(sess.dir)
			delete(uploadSessions, id)
		}
	}
}

func registerChunkedUploadTools(s *server.MCPServer, client *orbitclient.Client) {
	s.AddTool(mcp.NewTool("orbit_package_upload_begin",
		mcp.WithDescription("Begin a chunked .orb upload session. Use instead of install_from_base64 when the file is too large to pass inline. Workflow: (1) split file into ~65 KB chunks with `split -b 65536 file.orb /tmp/orb_chunk_`; (2) base64-encode each chunk with `base64 /tmp/orb_chunk_aa`; (3) send each via orbit_package_upload_chunk; (4) call orbit_package_upload_install. Returns upload_id for subsequent calls."),
		mcp.WithString("filename",
			mcp.Required(),
			mcp.Description("Package filename, e.g. sprinqua_v0.1.0.orb"),
		),
		mcp.WithNumber("total_chunks",
			mcp.Required(),
			mcp.Description("Total number of chunks that will be uploaded"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filename, err := req.RequireString("filename")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		base := filepath.Base(strings.TrimSpace(filename))
		if base == "" || base == "." || base == ".." {
			return mcp.NewToolResultError("filename must be a safe basename"), nil
		}
		if !strings.HasSuffix(strings.ToLower(base), ".orb") {
			return mcp.NewToolResultError("filename must end in .orb"), nil
		}
		totalChunks, err := req.RequireInt("total_chunks")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if totalChunks < 1 || totalChunks > 10000 {
			return mcp.NewToolResultError("total_chunks must be between 1 and 10000"), nil
		}

		workdir, err := mcpServerWorkDir()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		uploadMu.Lock()
		cleanStaleUploads()
		uploadID := fmt.Sprintf("%d", time.Now().UnixNano())
		sessDir := filepath.Join(workdir, "session-"+uploadID)
		uploadSessions[uploadID] = &uploadSession{
			filename:    base,
			totalChunks: totalChunks,
			received:    map[int]struct{}{},
			dir:         sessDir,
			createdAt:   time.Now(),
		}
		uploadMu.Unlock()

		if err := os.MkdirAll(sessDir, 0o700); err != nil {
			uploadMu.Lock()
			delete(uploadSessions, uploadID)
			uploadMu.Unlock()
			return mcp.NewToolResultError(err.Error()), nil
		}

		b, _ := json.MarshalIndent(map[string]any{
			"upload_id":    uploadID,
			"filename":     base,
			"total_chunks": totalChunks,
		}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_package_upload_chunk",
		mcp.WithDescription("Upload one base64-encoded chunk of a .orb file (0-based index). Get each chunk's base64 with: `base64 /tmp/orb_chunk_XX` (or equivalent). Call orbit_package_upload_install after all chunks are sent."),
		mcp.WithString("upload_id",
			mcp.Required(),
			mcp.Description("Upload ID from orbit_package_upload_begin"),
		),
		mcp.WithNumber("chunk_index",
			mcp.Required(),
			mcp.Description("0-based chunk index"),
		),
		mcp.WithString("data_base64",
			mcp.Required(),
			mcp.Description("Standard base64 of this chunk's raw bytes (no data: prefix)"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		uploadID, err := req.RequireString("upload_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		chunkIndex, err := req.RequireInt("chunk_index")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b64, err := req.RequireString("data_base64")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		uploadMu.Lock()
		sess, ok := uploadSessions[uploadID]
		uploadMu.Unlock()
		if !ok {
			return mcp.NewToolResultError(fmt.Sprintf("upload_id %q not found (expired or not started)", uploadID)), nil
		}
		if chunkIndex < 0 || chunkIndex >= sess.totalChunks {
			return mcp.NewToolResultError(fmt.Sprintf("chunk_index %d out of range [0, %d)", chunkIndex, sess.totalChunks)), nil
		}

		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("base64 decode: %v", err)), nil
		}
		chunkPath := filepath.Join(sess.dir, fmt.Sprintf("chunk-%05d", chunkIndex))
		if err := os.WriteFile(chunkPath, raw, 0o600); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		uploadMu.Lock()
		sess.received[chunkIndex] = struct{}{}
		received := len(sess.received)
		total := sess.totalChunks
		uploadMu.Unlock()

		b, _ := json.MarshalIndent(map[string]any{
			"ok":               true,
			"chunk_index":      chunkIndex,
			"chunks_received":  received,
			"chunks_total":     total,
			"chunks_remaining": total - received,
		}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("orbit_package_upload_install",
		mcp.WithDescription("Assemble all uploaded chunks into the .orb file and install it on the device. Cleans up temp files automatically."),
		mcp.WithString("upload_id",
			mcp.Required(),
			mcp.Description("Upload ID from orbit_package_upload_begin"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		uploadID, err := req.RequireString("upload_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		uploadMu.Lock()
		sess, ok := uploadSessions[uploadID]
		if ok {
			delete(uploadSessions, uploadID)
		}
		uploadMu.Unlock()
		if !ok {
			return mcp.NewToolResultError(fmt.Sprintf("upload_id %q not found", uploadID)), nil
		}
		defer func() { _ = os.RemoveAll(sess.dir) }()

		if len(sess.received) != sess.totalChunks {
			missing := sess.totalChunks - len(sess.received)
			return mcp.NewToolResultError(fmt.Sprintf("%d of %d chunks are missing — upload all chunks first", missing, sess.totalChunks)), nil
		}

		orbPath := filepath.Join(sess.dir, sess.filename)
		out, err := os.OpenFile(orbPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		for i := 0; i < sess.totalChunks; i++ {
			data, err := os.ReadFile(filepath.Join(sess.dir, fmt.Sprintf("chunk-%05d", i)))
			if err != nil {
				out.Close()
				return mcp.NewToolResultError(fmt.Sprintf("read chunk %d: %v", i, err)), nil
			}
			if _, err := out.Write(data); err != nil {
				out.Close()
				return mcp.NewToolResultError(fmt.Sprintf("assemble chunk %d: %v", i, err)), nil
			}
		}
		out.Close()

		if err := client.PackageManager.InstallPackage(ctx, orbPath); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(`{"ok": true}`), nil
	})

	s.AddTool(mcp.NewTool("orbit_package_upload_abort",
		mcp.WithDescription("Abort a chunked upload session and delete all temp files."),
		mcp.WithString("upload_id",
			mcp.Required(),
			mcp.Description("Upload ID from orbit_package_upload_begin"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		uploadID, err := req.RequireString("upload_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		uploadMu.Lock()
		sess, ok := uploadSessions[uploadID]
		if ok {
			delete(uploadSessions, uploadID)
		}
		uploadMu.Unlock()
		if ok {
			_ = os.RemoveAll(sess.dir)
		}
		b, _ := json.MarshalIndent(map[string]any{"ok": true, "upload_id": uploadID}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})
}
