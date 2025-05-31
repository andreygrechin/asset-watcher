package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"cloud.google.com/go/asset/apiv1/assetpb"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/types/known/structpb"
)

// AssetIterator is an interface for iterating over assets.
type AssetIterator interface {
	Next() (*assetpb.ResourceSearchResult, error)
}

// ProcessedAsset represents the processed asset information.
type ProcessedAsset struct {
	Name      string `json:"name"`
	Location  string `json:"location"`
	Status    string `json:"status"`
	IPAddress string `json:"ipAddress"`
	Project   string `json:"project"`
	CreatedAt string `json:"createdAt"`
}

// AssetProcessor is a client for processing assets.
type AssetProcessor struct {
	logger *slog.Logger
	cfg    *Config
}

// NewAssetProcessor creates a new AssetProcessor instance.
func NewAssetProcessor(_ context.Context, logger *slog.Logger, cfg *Config) *AssetProcessor {
	return &AssetProcessor{
		logger: logger.With(slog.String("component", "asset-watcher")),
		cfg:    cfg,
	}
}

func splitString(s string, separator string) []string {
	if strings.TrimSpace(s) == "" {
		return []string{}
	}

	tempResult := strings.Split(s, separator)
	result := make([]string, 0, len(tempResult))

	for _, str := range tempResult {
		trimmedStr := strings.TrimSpace(str)
		if trimmedStr != "" {
			result = append(result, trimmedStr)
		}
	}

	return result
}

// ProcessAssets processes the assets and filters them based on the configuration.
func (p *AssetProcessor) ProcessAssets(ctx context.Context,
	assets AssetIterator,
) ([]ProcessedAsset, error) {
	totalAssets := 0

	includeProjects := splitString(p.cfg.IncludeProjects, ",")
	excludeProjects := splitString(p.cfg.ExcludeProjects, ",")

	p.logger.DebugContext(ctx, "Processing assets...")

	processedResults := make([]ProcessedAsset, 0, totalAssets)

	for {
		asset, err := assets.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("failed to create asset client: %w", err)
		}

		totalAssets++
		projectID := getProjectID(asset)
		ipAddress := getIPAddress(asset)

		if p.cfg.ExcludeReserved && asset.GetState() == "RESERVED" {
			continue
		}

		if slices.Contains(excludeProjects, projectID) {
			continue
		}

		var include bool
		if len(includeProjects) > 0 {
			include = slices.Contains(includeProjects, projectID)
		} else {
			include = true
		}

		if include {
			processedResults = append(processedResults, ProcessedAsset{
				Name:      asset.GetDisplayName(),
				Location:  asset.GetLocation(),
				Project:   projectID,
				IPAddress: ipAddress,
				Status:    asset.GetState(),
				CreatedAt: asset.GetCreateTime().AsTime().Format("2006-01-02 15:04:05"),
			})
		}
	}

	p.logger.DebugContext(ctx, "Finished processing assets",
		slog.Int("total_assets", totalAssets),
		slog.Int("total_filtered", totalAssets-len(processedResults)),
	)

	return processedResults, nil
}

func getIPAddress(asset *assetpb.ResourceSearchResult) string {
	ipAddress := "N/A"

	isFieldsExists := asset.GetAdditionalAttributes() != nil && asset.GetAdditionalAttributes().GetFields() != nil
	if !isFieldsExists {
		return ipAddress
	}

	if addressField, ok := asset.GetAdditionalAttributes().GetFields()["address"]; ok {
		if addressField != nil {
			if sv, ok := addressField.GetKind().(*structpb.Value_StringValue); ok {
				ipAddress = sv.StringValue
			}
		}
	}

	return ipAddress
}

func getProjectID(asset *assetpb.ResourceSearchResult) string {
	projectID := "N/A"

	if asset.GetParentAssetType() == "cloudresourcemanager.googleapis.com/Project" {
		parts := strings.Split(asset.GetParentFullResourceName(), "/")
		if len(parts) > 0 {
			projectID = parts[len(parts)-1]
		}
	}

	return projectID
}
