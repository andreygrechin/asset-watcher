package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"text/tabwriter"
)

const tabWriterPadding = 3

func outputToStdOut(ctx context.Context, logger *slog.Logger, processedAssets []ProcessedAsset, outputFormat string) {
	switch outputFormat {
	case "table":
		outputToStdOutTable(ctx, logger, processedAssets)
	case "json":
		outputToStdOutJSON(ctx, logger, processedAssets)
	default:
		fmt.Fprintf(os.Stderr, "unknown output format: %s\n", outputFormat)
		outputToStdOutTable(ctx, logger, processedAssets)
	}
}

func outputToStdOutTable(ctx context.Context, logger *slog.Logger, processedAssets []ProcessedAsset) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, tabWriterPadding, ' ', tabwriter.Debug)
	_, _ = fmt.Fprintln(w, "Display Name\tLocation\tProject ID\tIP Address\tState\tCreated At")
	_, _ = fmt.Fprintln(w, "------------\t--------\t----------\t----------\t-----\t----------")

	for _, asset := range processedAssets {
		resource := asset

		_, _ = fmt.Fprintf(
			w,
			"%s\t%s\t%s\t%s\t%s\t%s\n",
			resource.Name,
			resource.Location,
			resource.Project,
			resource.IPAddress,
			resource.Status,
			resource.CreatedAt,
		)
	}

	err := w.Flush()
	if err != nil {
		logger.ErrorContext(ctx, "failed to flush output", slog.Any("error", err))
		os.Exit(1)
	}
}

func outputToStdOutJSON(ctx context.Context, logger *slog.Logger, processedAssets []ProcessedAsset) {
	jsonData, err := json.MarshalIndent(processedAssets, "", "  ")
	if err != nil {
		logger.ErrorContext(ctx, "failed to marshal JSON: %v", slog.Any("error", err))
		os.Exit(1)
	}

	fmt.Println(string(jsonData))
}
