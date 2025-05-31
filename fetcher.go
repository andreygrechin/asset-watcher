package main

import (
	"context"
	"fmt"
	"log/slog"

	asset "cloud.google.com/go/asset/apiv1"
	"cloud.google.com/go/asset/apiv1/assetpb"
	"google.golang.org/api/option"
)

// Fetcher is an interface for fetching assets.
type Fetcher interface {
	FetchAssets(ctx context.Context) *asset.ResourceSearchResultIterator
	Close() error
}

// GoogleAssetFetcher is a client and its configurations.
type GoogleAssetFetcher struct {
	client *asset.Client
	logger *slog.Logger
	cfg    *Config
}

// NewGoogleAssetFetcher creates a new Google Asset fetcher.
func NewGoogleAssetFetcher(
	ctx context.Context,
	logger *slog.Logger,
	cfg *Config,
	opts ...option.ClientOption,
) (*GoogleAssetFetcher, error) {
	c, err := asset.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create asset client: %w", err)
	}

	return &GoogleAssetFetcher{
		client: c,
		logger: logger.With(slog.String("component", "asset-watcher")),
		cfg:    cfg,
	}, nil
}

// FetchAssets fetches the assets from Google Cloud Asset API.
func (f *GoogleAssetFetcher) FetchAssets(ctx context.Context) *asset.ResourceSearchResultIterator {
	req := &assetpb.SearchAllResourcesRequest{
		Scope:      "organizations/" + f.cfg.OrgID,
		OrderBy:    "project,name",
		AssetTypes: []string{"compute.googleapis.com/Address"},
	}

	assets := f.client.SearchAllResources(ctx, req)

	return assets
}

// Close closes the asset client.
func (f *GoogleAssetFetcher) Close() error {
	if err := f.client.Close(); err != nil {
		return fmt.Errorf("failed to close asset client: %w", err)
	}

	return nil
}
