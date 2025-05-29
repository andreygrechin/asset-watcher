package main

import (
	"context"
	"io"
	"log/slog"
	"reflect"
	"testing"
	"time"

	"cloud.google.com/go/asset/apiv1/assetpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSplitString(t *testing.T) {
	tests := []struct {
		name      string
		s         string
		separator string
		want      []string
	}{
		{name: "empty string", s: "", separator: ",", want: []string{}},
		{name: "string with no separators", s: "abc", separator: ",", want: []string{"abc"}},
		{name: "string with leading/trailing spaces and spaces around separators", s: "  abc  ,  def  ,ghi,jkl  ", separator: ",", want: []string{"abc", "def", "ghi", "jkl"}},
		{name: "string with multiple separators", s: "abc,,def", separator: ",", want: []string{"abc", "def"}},
		{name: "string with only separators", s: ",,,", separator: ",", want: []string{}},
		{name: "string with different separator", s: "abc;def;ghi", separator: ";", want: []string{"abc", "def", "ghi"}},
		{name: "string with multiple character separator", s: "abc<sep>def<sep>ghi", separator: "<sep>", want: []string{"abc", "def", "ghi"}},
		{name: "empty string with spaces", s: "   ", separator: ",", want: []string{}},
		{name: "separator at the beginning", s: ",abc,def", separator: ",", want: []string{"abc", "def"}},
		{name: "separator at the end", s: "abc,def,", separator: ",", want: []string{"abc", "def"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := splitString(tt.s, tt.separator); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetIPAddress(t *testing.T) {
	tests := []struct {
		name  string
		asset *assetpb.ResourceSearchResult
		want  string
	}{
		{name: "asset with IP address", asset: &assetpb.ResourceSearchResult{AdditionalAttributes: &structpb.Struct{Fields: map[string]*structpb.Value{"address": structpb.NewStringValue("192.168.1.1")}}}, want: "192.168.1.1"},
		{name: "asset with no address field", asset: &assetpb.ResourceSearchResult{AdditionalAttributes: &structpb.Struct{Fields: map[string]*structpb.Value{"other_field": structpb.NewStringValue("some_value")}}}, want: "N/A"},
		{name: "asset with address field not a string", asset: &assetpb.ResourceSearchResult{AdditionalAttributes: &structpb.Struct{Fields: map[string]*structpb.Value{"address": structpb.NewNumberValue(123)}}}, want: "N/A"},
		{name: "asset with nil AdditionalAttributes", asset: &assetpb.ResourceSearchResult{AdditionalAttributes: nil}, want: "N/A"},
		{name: "asset with nil Fields in AdditionalAttributes", asset: &assetpb.ResourceSearchResult{AdditionalAttributes: &structpb.Struct{Fields: nil}}, want: "N/A"},
		{name: "asset with address field being nil structpb.Value", asset: &assetpb.ResourceSearchResult{AdditionalAttributes: &structpb.Struct{Fields: map[string]*structpb.Value{"address": nil}}}, want: "N/A"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getIPAddress(tt.asset); got != tt.want {
				t.Errorf("getIPAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetProjectID(t *testing.T) {
	tests := []struct {
		name  string
		asset *assetpb.ResourceSearchResult
		want  string
	}{
		{name: "asset with correct project ID", asset: &assetpb.ResourceSearchResult{ParentAssetType: "cloudresourcemanager.googleapis.com/Project", ParentFullResourceName: "//cloudresourcemanager.googleapis.com/projects/my-project-123"}, want: "my-project-123"},
		{name: "asset with different parent asset type", asset: &assetpb.ResourceSearchResult{ParentAssetType: "compute.googleapis.com/Instance", ParentFullResourceName: "//cloudresourcemanager.googleapis.com/projects/another-project-456"}, want: "N/A"},
		{name: "asset with project parent type but empty resource name", asset: &assetpb.ResourceSearchResult{ParentAssetType: "cloudresourcemanager.googleapis.com/Project", ParentFullResourceName: ""}, want: ""},
		{name: "asset with project parent type but malformed resource name (no slashes)", asset: &assetpb.ResourceSearchResult{ParentAssetType: "cloudresourcemanager.googleapis.com/Project", ParentFullResourceName: "my-project-malformed"}, want: "my-project-malformed"},
		{name: "asset with project parent type but resource name is just slashes", asset: &assetpb.ResourceSearchResult{ParentAssetType: "cloudresourcemanager.googleapis.com/Project", ParentFullResourceName: "//"}, want: ""},
		{name: "asset with project parent type, resource name ends with slash", asset: &assetpb.ResourceSearchResult{ParentAssetType: "cloudresourcemanager.googleapis.com/Project", ParentFullResourceName: "//cloudresourcemanager.googleapis.com/projects/project-ending-slash/"}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getProjectID(tt.asset); got != tt.want {
				t.Errorf("getProjectID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func newTestAssetHelper(name, projectID, state, ipAddress string, createTime time.Time) *assetpb.ResourceSearchResult {
	asset := &assetpb.ResourceSearchResult{
		DisplayName: name,
		State:       state,
		CreateTime:  timestamppb.New(createTime),
	}
	if projectID != "" {
		asset.ParentAssetType = "cloudresourcemanager.googleapis.com/Project"
		asset.ParentFullResourceName = "//cloudresourcemanager.googleapis.com/projects/" + projectID
	} else {
		asset.ParentAssetType = "organizations/org-id"
		asset.ParentFullResourceName = "//cloudresourcemanager.googleapis.com/organizations/org-id"
	}

	if ipAddress != "N/A" && ipAddress != "" {
		asset.AdditionalAttributes = &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"address": structpb.NewStringValue(ipAddress),
			},
		}
	} else {
		asset.AdditionalAttributes = &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"other_info": structpb.NewStringValue("no_ip"),
			},
		}
	}
	return asset
}

func TestProcessAssets_Conceptual(t *testing.T) {
	// This function remains as a conceptual placeholder due to difficulties in mocking
	// the concrete `*asset.ResourceSearchResultIterator` needed by `ProcessAssets`.
	// Full testing of `ProcessAssets` with varied inputs would require refactoring
	// `processor.go` or using integration tests.

	ctx := context.Background()                              // Required for NewAssetProcessor
	logger := slog.New(slog.NewTextHandler(io.Discard, nil)) // Required for NewAssetProcessor

	baseTime := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)
	expectedFormattedTime := baseTime.Format("2006-01-02 15:04:05")

	// Example asset for demonstrating ProcessedAsset mapping
	assetActive := newTestAssetHelper("active-asset", "proj-A", "ACTIVE", "1.2.3.4", baseTime)
	// Unused asset variables removed: assetReserved, assetExcludedProj, assetIncludedProj

	t.Log("TestProcessAssets_Conceptual: This test is a conceptual placeholder.")
	t.Log("Direct unit testing of ProcessAssets with controlled asset inputs is not currently feasible without refactoring processor.go or using integration tests.")
	t.Log("The function relies on a concrete iterator type from the GCP SDK, which cannot be easily mocked with a slice of test data.")

	// Test the instantiation of AssetProcessor (very minor)
	cfgForProcessor := &Config{OrgID: "test-org"}
	_ = NewAssetProcessor(ctx, logger, cfgForProcessor) // ap variable removed as it was unused

	// Test how default IncludeProjects/ExcludeProjects from config are split
	if len(splitString(cfgForProcessor.IncludeProjects, ",")) != 0 {
		t.Errorf("expected default IncludeProjects to result in empty slice, got %d", len(splitString(cfgForProcessor.IncludeProjects, ",")))
	}
	if len(splitString(cfgForProcessor.ExcludeProjects, ",")) != 0 {
		t.Errorf("expected default ExcludeProjects to result in empty slice, got %d", len(splitString(cfgForProcessor.ExcludeProjects, ",")))
	}

	// Test the manual mapping part for a single asset, simulating what happens inside ProcessAssets loop
	singleAsset := assetActive // Use the one defined asset
	projectID := getProjectID(singleAsset)
	ipAddress := getIPAddress(singleAsset)

	pa := ProcessedAsset{
		Name:      singleAsset.GetDisplayName(),
		Location:  singleAsset.GetLocation(),
		Project:   projectID,
		IPAddress: ipAddress,
		Status:    singleAsset.GetState(),
		CreatedAt: singleAsset.GetCreateTime().AsTime().Format("2006-01-02 15:04:05"),
	}

	if pa.Name != "active-asset" {
		t.Errorf("processedAsset mapping error for Name")
	}
	if pa.Project != "proj-A" {
		t.Errorf("processedAsset mapping error for Project")
	}
	if pa.IPAddress != "1.2.3.4" {
		t.Errorf("processedAsset mapping error for IPAddress")
	}
	if pa.Status != "ACTIVE" {
		t.Errorf("processedAsset mapping error for Status")
	}
	if pa.CreatedAt != expectedFormattedTime {
		t.Errorf("processedAsset mapping error for CreatedAt: got %s, want %s", pa.CreatedAt, expectedFormattedTime)
	}

	t.Log("Verified ProcessedAsset struct population for a single manually-mapped asset.")
}
