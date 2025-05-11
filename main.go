// Project: asset-watcher
package main

import (
	"context"
	"log/slog"
	"os"
)

var (
	Version   = "unknown" // unexported
	BuildTime = "unknown" // BuildTime is set at build time using -X flag
	Commit    = "unknown" // Commit is set at build time using -X flag
)

func main() {
	cfg := GetConfig()

	ctx := context.Background()

	logger := setupLogging(cfg)

	logger.DebugContext(
		ctx, "version information",
		slog.String("version", Version),
		slog.String("build_time", BuildTime),
		slog.String("commit", Commit),
	)

	fetcher, err := NewGoogleAssetFetcher(ctx, logger, cfg)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create an asset fetcher", slog.Any("error", err))
		os.Exit(1)
	}

	defer func() {
		if err := fetcher.Close(); err != nil {
			logger.ErrorContext(ctx, "failed to close asset client", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	assets := fetcher.FetchAssets(ctx)
	processor := NewAssetProcessor(ctx, logger, cfg)

	processedAssets, err := processor.ProcessAssets(ctx, assets)
	if err != nil {
		logger.ErrorContext(ctx, "failed to process assets", slog.Any("error", err))
	}

	logger.DebugContext(ctx, "Processed asset:", slog.Int("number_of_asset", len(processedAssets)))
	logger.WarnContext(ctx, "Processed asset:", slog.Int("number_of_asset", len(processedAssets)))

	outputToStdOut(ctx, logger, processedAssets, cfg.OutputFormat)
}
