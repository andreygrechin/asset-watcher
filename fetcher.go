package main

import (
	"context"
	"fmt"
	"log/slog"

	advisorynotifications "cloud.google.com/go/advisorynotifications/apiv1"
	advisorynotificationspb "cloud.google.com/go/advisorynotifications/apiv1/advisorynotificationspb"
	"google.golang.org/api/iterator"
)

// Fetcher defines the interface for fetching advisory notifications.
type Fetcher interface {
	FetchNotifications(ctx context.Context) ([]*advisorynotificationspb.Notification, error)
	Close() error // Add Close method to the interface
}

// GoogleAdvisoryNotificationFetcher implements the Fetcher interface using the Google Cloud client.
type GoogleAdvisoryNotificationFetcher struct {
	client *advisorynotifications.Client
	orgID  string
	logger *slog.Logger
}

// NewGoogleAdvisoryNotificationFetcher creates a new instance of GoogleAdvisoryNotificationFetcher.
// It initializes the Advisory Notifications client.
func NewGoogleAdvisoryNotificationFetcher(ctx context.Context, orgID string, logger *slog.Logger) (*GoogleAdvisoryNotificationFetcher, error) {
	c, err := advisorynotifications.NewClient(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create advisorynotifications client", slog.Any("error", err))
		return nil, err
	}

	return &GoogleAdvisoryNotificationFetcher{
		client: c,
		orgID:  orgID,
		logger: logger.With(slog.String("component", "advisory_notification_fetcher")),
	}, nil
}

// FetchNotifications fetches notifications from the Google Cloud Advisory Notifications API.
func (f *GoogleAdvisoryNotificationFetcher) FetchNotifications(ctx context.Context) ([]*advisorynotificationspb.Notification, error) {
	req := &advisorynotificationspb.ListNotificationsRequest{
		Parent:       fmt.Sprintf("organizations/%s/locations/global", f.orgID),
		View:         advisorynotificationspb.NotificationView_FULL,
		LanguageCode: "en",
	}

	var notifications []*advisorynotificationspb.Notification
	it := f.client.ListNotifications(ctx, req)
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			f.logger.ErrorContext(ctx, "failed to list Advisory Notifications page", slog.Any("error", err))
			// Decide if we want to return partial results or fail entirely
			// For now, let's fail entirely on page error
			return nil, fmt.Errorf("failed to list Advisory Notifications: %w", err)
		}
		notifications = append(notifications, resp)
	}

	f.logger.InfoContext(ctx, "Successfully fetched notifications", slog.Int("count", len(notifications)))
	return notifications, nil
}

// Close closes the underlying Google Cloud client.
func (f *GoogleAdvisoryNotificationFetcher) Close(ctx context.Context) error {
	if f.client != nil {
		err := f.client.Close()
		if err != nil {
			f.logger.ErrorContext(ctx, "failed to close advisorynotifications client", slog.Any("error", err))
		}
		f.logger.InfoContext(ctx, "Advisory notifications client closed successfully")
	}
	return nil
}
