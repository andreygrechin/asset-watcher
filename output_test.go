package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
)

// captureStdout is a helper function to capture standard output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	fn() // Execute the function whose output we want to capture

	if err := w.Close(); err != nil {
		if !strings.Contains(err.Error(), "file already closed") {
			t.Logf("error closing writer pipe: %v", err)
		}
	}
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Logf("error reading from pipe: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Logf("error closing reader pipe: %v", err)
	}
	return buf.String()
}

// TestOutputToStdOutTable tests the outputToStdOutTable function.
func TestOutputToStdOutTable(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	sampleAssets := []ProcessedAsset{
		{Name: "Asset1", Location: "loc1", Project: "proj1", IPAddress: "1.1.1.1", Status: "ACTIVE", CreatedAt: "2023-01-01"},
		{Name: "Asset2", Location: "loc2", Project: "proj2", IPAddress: "2.2.2.2", Status: "RESERVED", CreatedAt: "2023-01-02"},
	}

	// Expected header column names (keywords)
	expectedHeaderKeywords := []string{"Display Name", "Location", "Project ID", "IP Address", "State", "Created At"}

	t.Run("No assets", func(t *testing.T) {
		output := captureStdout(t, func() {
			outputToStdOutTable(ctx, logger, []ProcessedAsset{})
		})

		// Check for header keywords
		for _, keyword := range expectedHeaderKeywords {
			if !strings.Contains(output, keyword) {
				t.Errorf("table header keyword '%s' not found in output with no assets. Output:\n%s", keyword, output)
			}
		}

		lines := strings.Split(strings.TrimSpace(output), "\n")
		// Expect at least 2 lines for header and separator, possibly more due to tabwriter.Debug
		if len(lines) < 2 {
			t.Errorf("expected at least 2 lines for header and separator, got %d. Output:\n%s", len(lines), output)
		}

		// Check that specific asset data is not present
		if strings.Contains(output, "Asset1") || strings.Contains(output, "proj1") {
			t.Errorf("found asset data in output when no assets were provided. Output:\n%s", output)
		}
	})

	t.Run("With assets", func(t *testing.T) {
		output := captureStdout(t, func() {
			outputToStdOutTable(ctx, logger, sampleAssets)
		})

		// Check for header keywords
		for _, keyword := range expectedHeaderKeywords {
			if !strings.Contains(output, keyword) {
				t.Errorf("table header keyword '%s' not found in output with assets. Output:\n%s", keyword, output)
			}
		}

		// Check for asset details
		for _, asset := range sampleAssets {
			if !strings.Contains(output, asset.Name) {
				t.Errorf("asset name %s not found in table output. Output:\n%s", asset.Name, output)
			}
			if !strings.Contains(output, asset.Project) {
				t.Errorf("asset project %s not found in table output. Output:\n%s", asset.Project, output)
			}
			if !strings.Contains(output, asset.IPAddress) {
				t.Errorf("asset IPAddress %s not found in table output. Output:\n%s", asset.IPAddress, output)
			}
		}
	})
}

// TestOutputToStdOutJSON tests the outputToStdOutJSON function.
func TestOutputToStdOutJSON(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	sampleAssets := []ProcessedAsset{
		{Name: "Asset1", Location: "loc1", Project: "proj1", IPAddress: "1.1.1.1", Status: "ACTIVE", CreatedAt: "2023-01-01"},
		{Name: "Asset2", Location: "loc2", Project: "proj2", IPAddress: "2.2.2.2", Status: "RESERVED", CreatedAt: "2023-01-02"},
	}

	t.Run("No assets", func(t *testing.T) {
		output := captureStdout(t, func() {
			outputToStdOutJSON(ctx, logger, []ProcessedAsset{})
		})

		var unmarshalledOutput []ProcessedAsset
		err := json.Unmarshal([]byte(output), &unmarshalledOutput)
		if err != nil {
			t.Fatalf("output with no assets is not valid JSON: %v\nOutput was: %s", err, output)
		}
		if len(unmarshalledOutput) != 0 {
			t.Errorf("expected empty JSON array, got %d elements", len(unmarshalledOutput))
		}
	})

	t.Run("With assets", func(t *testing.T) {
		output := captureStdout(t, func() {
			outputToStdOutJSON(ctx, logger, sampleAssets)
		})

		var processedOutput []ProcessedAsset
		err := json.Unmarshal([]byte(output), &processedOutput)
		if err != nil {
			t.Fatalf("output with assets is not valid JSON: %v\nOutput was: %s", err, output)
		}

		if len(processedOutput) != len(sampleAssets) {
			t.Errorf("expected %d assets in JSON output, got %d", len(sampleAssets), len(processedOutput))
		}

		for i, asset := range sampleAssets {
			if i < len(processedOutput) {
				if processedOutput[i].Name != asset.Name {
					t.Errorf("asset name mismatch in JSON output. Expected %s, got %s", asset.Name, processedOutput[i].Name)
				}
				if processedOutput[i].Project != asset.Project {
					t.Errorf("asset project mismatch in JSON output. Expected %s, got %s", asset.Project, processedOutput[i].Project)
				}
			}
		}
	})
}
