package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

var (
	Version   = "unknown"
	BuildTime = "unknown"
	Commit    = "unknown"
)

func main() {
	ctx := context.Background()

	cfg, err := LoadConfig()
	if err != nil {
		// Use standard log here as slog is not configured yet
		fmt.Fprintf(os.Stderr, "failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	logger := setupLogging(cfg)

	logger.InfoContext(
		ctx, "version information",
		slog.String("version", Version),
		slog.String("build_time", BuildTime),
		slog.String("commit", Commit),
	)

	fetcher, err := NewGoogleAdvisoryNotificationFetcher(ctx, cfg.OrgID, logger)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create notification fetcher", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := fetcher.Close(ctx); err != nil {
			// Error already logged in Close method
			os.Exit(1) // Exit if closing fails, as it might indicate resource leaks
		}
	}()

	notifications, err := fetcher.FetchNotifications(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "failed to fetch notifications", slog.Any("error", err))
		os.Exit(1)
	}

	processor := NewNotificationProcessor(logger, cfg)
	processedNotifications, err := processor.ProcessNotifications(ctx, notifications)
	if err != nil {
		logger.ErrorContext(ctx, "failed to process notifications",
			slog.Any("error", err),
		)
		os.Exit(1)
	}

	var sendErrors bool

	notifier := NewSlackNotifier(cfg.SlackToken, cfg.SlackChannelID, logger)

	for _, pNotification := range processedNotifications {
		if err := notifier.SendNotification(ctx, pNotification); err != nil {
			// Log error but continue processing other notifications
			logger.ErrorContext(ctx, "failed to send one notification",
				slog.Any("error", err),
				slog.String("notification_name", pNotification.OriginalName),
			)
			sendErrors = true // Mark that at least one error occurred
		}
	}

	if sendErrors {
		logger.WarnContext(ctx, "encountered errors while sending some notifications")
		os.Exit(1)
	}
}
