package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/slack-go/slack"
)

// Notifier defines the interface for sending notifications.
type Notifier interface {
	SendNotification(ctx context.Context, notification ProcessedNotification) error
}

// SlackNotifier implements the Notifier interface for Slack.
type SlackNotifier struct {
	client    *slack.Client
	channelID string
	logger    *slog.Logger
}

// NewSlackNotifier creates a new SlackNotifier.
func NewSlackNotifier(token, channelID string, logger *slog.Logger) *SlackNotifier {
	client := slack.New(token)
	return &SlackNotifier{
		client:    client,
		channelID: channelID,
		logger:    logger.With(slog.String("component", "slack_notifier")),
	}
}

// SendNotification sends a processed notification to Slack.
func (s *SlackNotifier) SendNotification(ctx context.Context, notification ProcessedNotification) error {
	// Construct the message using Block Kit for better formatting
	headerText := notification.Subject
	headerBlock := slack.NewHeaderBlock(slack.NewTextBlockObject(slack.PlainTextType, headerText, false, false))
	bodyBlock := slack.NewSectionBlock(slack.NewTextBlockObject(slack.MarkdownType,
		notification.FormattedBody, false, false), nil, nil)
	dividerBlock := slack.NewDividerBlock()

	msgOptions := []slack.MsgOption{
		slack.MsgOptionBlocks(headerBlock, dividerBlock, bodyBlock),
		slack.MsgOptionText(fmt.Sprintf("New Advisory Notification: %s", notification.Subject), true), // Fallback text for notifications
	}

	channelID, timestamp, err := s.client.PostMessageContext(ctx, s.channelID, msgOptions...)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to send notification to Slack",
			slog.Any("error", err),
			slog.String("notification_name", notification.OriginalName),
			slog.String("subject", notification.Subject),
		)
		return fmt.Errorf("failed to post message for notification %s: %w", notification.OriginalName, err)
	}

	s.logger.InfoContext(ctx, "Notification sent successfully to Slack",
		slog.String("channel_id", channelID),
		slog.String("timestamp", timestamp),
		slog.String("notification_name", notification.OriginalName),
	)
	return nil
}
