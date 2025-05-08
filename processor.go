package main

import (
	"context"
	"log/slog"
	"strings"
	"time"

	advisorynotificationspb "cloud.google.com/go/advisorynotifications/apiv1/advisorynotificationspb"
)

// ProcessedNotification holds the formatted content ready for sending.
type ProcessedNotification struct {
	OriginalName  string // Identifier for the original notification
	Subject       string
	FormattedBody string
	MsgCreateTime time.Time
}

// NotificationProcessor handles the processing of fetched notifications.
type NotificationProcessor struct {
	logger *slog.Logger
	cfg    *Config
}

// NewNotificationProcessor creates a new NotificationProcessor.
func NewNotificationProcessor(logger *slog.Logger, config *Config) *NotificationProcessor {
	return &NotificationProcessor{
		logger: logger.With(slog.String("component", "notification_processor")),
		cfg:    config,
	}
}

// ProcessNotifications iterates through notifications, converts content, logs details,
// and returns formatted messages suitable for sending.
func (p *NotificationProcessor) ProcessNotifications(ctx context.Context,
	notifications []*advisorynotificationspb.Notification,
) ([]ProcessedNotification, error) {
	totalNotifications := len(notifications)
	p.logger.InfoContext(ctx, "Processing notifications", slog.Int("count", totalNotifications))

	processedResults := make([]ProcessedNotification, 0, totalNotifications) // Pre-allocate slice

	for _, resp := range notifications {
		p.logger.DebugContext(ctx, "Processing notification",
			slog.String("name", resp.Name),
			slog.String("subject", resp.Subject.Text.EnText),
			slog.String("notification_type", resp.NotificationType.String()),
			slog.Time("create_time", resp.CreateTime.AsTime()),
		)

		// Process messages within a notification.
		var combinedMarkdown strings.Builder
		p.logger.DebugContext(ctx, "Processing messages for notification",
			slog.String("notification_name", resp.Name))
		for i, msg := range resp.Messages {
			markdown := convertHTMLToMarkdown(ctx, p.logger, msg.Body.Text.EnText, resp.Name)

			combinedMarkdown.WriteString(markdown)
			if i < len(resp.Messages)-1 {
				combinedMarkdown.WriteString("\n\n---\n\n") // Separator between messages
			}

			p.logger.DebugContext(ctx, "Message details",
				slog.Int("message_index", i+1),
				slog.String("markdown_body", markdown),
				slog.Time("message_create_time", msg.CreateTime.AsTime()),
				slog.Bool("has_attachment", len(msg.Attachments) > 0),
			)
		}

		// Check if the message creation time is within the configured max notification age
		now := time.Now()
		maxAgeThreshold := now.Add(-time.Duration(p.cfg.MaxNotificationAgeSeconds) * time.Second)
		if resp.CreateTime.AsTime().After(maxAgeThreshold) {
			processed := ProcessedNotification{
				OriginalName:  resp.Name,
				Subject:       resp.Subject.Text.EnText,
				FormattedBody: combinedMarkdown.String(),
				MsgCreateTime: resp.CreateTime.AsTime(),
			}
			processedResults = append(processedResults, processed)
		}

		p.logger.DebugContext(ctx, "Finished processing messages for notification",
			slog.String("notification_name", resp.Name),
			slog.Int("message_count", len(resp.Messages)),
		)
	}

	p.logger.InfoContext(ctx, "Finished processing notifications",
		slog.Int("total_processed", len(processedResults)),
		slog.Int("total_filtered", totalNotifications-len(processedResults)),
	)
	return processedResults, nil // Indicate success
}
