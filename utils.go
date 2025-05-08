package main

import (
	"context"
	"log/slog"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
)

// convertHTMLToMarkdown converts an HTML string to Markdown.
// It logs a warning and returns "N/A" if the conversion fails.
func convertHTMLToMarkdown(
	ctx context.Context,
	logger *slog.Logger,
	htmlContent string,
	notificationName string,
) string {
	markdown, err := htmltomarkdown.ConvertString(htmlContent)
	if err != nil {
		logger.WarnContext(ctx, "failed to convert HTML body to Markdown",
			slog.Any("error", err),
			slog.String("notification_name", notificationName),
		)
		return "N/A"
	}
	return markdown
}
