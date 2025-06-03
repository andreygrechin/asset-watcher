package main

import (
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"testing"
	"time"

	"cloud.google.com/go/asset/apiv1/assetpb"
	"google.golang.org/api/iterator"
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

// createTestAsset is a helper function to create test assets.
func createTestAsset(name, projectID, state, ipAddress string, createTime time.Time) *assetpb.ResourceSearchResult {
	asset := &assetpb.ResourceSearchResult{
		DisplayName: name,
		State:       state,
		CreateTime:  timestamppb.New(createTime),
		Location:    "us-central1",
	}

	if projectID != "" {
		asset.ParentAssetType = "cloudresourcemanager.googleapis.com/Project"
		asset.ParentFullResourceName = "//cloudresourcemanager.googleapis.com/projects/" + projectID
	}

	if ipAddress != "" {
		asset.AdditionalAttributes = &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"address": structpb.NewStringValue(ipAddress),
			},
		}
	}

	return asset
}

var errSimulatedAPI = errors.New("simulated API error")

type mockAssetIterator struct {
	assets []*assetpb.ResourceSearchResult
	index  int
	err    error
}

// Next returns the next asset or an error.
func (m *mockAssetIterator) Next() (*assetpb.ResourceSearchResult, error) {
	if m.err != nil {
		return nil, m.err
	}

	if m.index >= len(m.assets) {
		return nil, iterator.Done
	}

	asset := m.assets[m.index]
	m.index++

	return asset, nil
}

// TestProcessAssets tests the ProcessAssets function with various configurations.
func TestProcessAssets(t *testing.T) {
	ctx := t.Context()
	logger := slog.New(slog.DiscardHandler)
	baseTime := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		config         *Config
		assets         []*assetpb.ResourceSearchResult
		expectedCount  int
		expectedAssets []ProcessedAsset
	}{
		{
			name: "no filtering",
			config: &Config{
				OrgID:           "test-org",
				ExcludeReserved: false,
				ExcludeProjects: "",
				IncludeProjects: "",
			},
			assets: []*assetpb.ResourceSearchResult{
				createTestAsset("asset1", "proj-A", "ACTIVE", "1.2.3.4", baseTime),
				createTestAsset("asset2", "proj-B", "RESERVED", "5.6.7.8", baseTime),
			},
			expectedCount: 2,
			expectedAssets: []ProcessedAsset{
				{
					Name:      "asset1",
					Location:  "us-central1",
					Project:   "proj-A",
					IPAddress: "1.2.3.4",
					Status:    "ACTIVE",
					CreatedAt: "2024-01-10 12:00:00",
				},
				{
					Name:      "asset2",
					Location:  "us-central1",
					Project:   "proj-B",
					IPAddress: "5.6.7.8",
					Status:    "RESERVED",
					CreatedAt: "2024-01-10 12:00:00",
				},
			},
		},
		{
			name: "exclude reserved IPs",
			config: &Config{
				OrgID:           "test-org",
				ExcludeReserved: true,
				ExcludeProjects: "",
				IncludeProjects: "",
			},
			assets: []*assetpb.ResourceSearchResult{
				createTestAsset("asset1", "proj-A", "ACTIVE", "1.2.3.4", baseTime),
				createTestAsset("asset2", "proj-B", "RESERVED", "5.6.7.8", baseTime),
				createTestAsset("asset3", "proj-C", "IN_USE", "9.10.11.12", baseTime),
			},
			expectedCount: 2,
			expectedAssets: []ProcessedAsset{
				{
					Name:      "asset1",
					Location:  "us-central1",
					Project:   "proj-A",
					IPAddress: "1.2.3.4",
					Status:    "ACTIVE",
					CreatedAt: "2024-01-10 12:00:00",
				},
				{
					Name:      "asset3",
					Location:  "us-central1",
					Project:   "proj-C",
					IPAddress: "9.10.11.12",
					Status:    "IN_USE",
					CreatedAt: "2024-01-10 12:00:00",
				},
			},
		},
		{
			name: "exclude specific projects",
			config: &Config{
				OrgID:           "test-org",
				ExcludeReserved: false,
				ExcludeProjects: "proj-B,proj-D",
				IncludeProjects: "",
			},
			assets: []*assetpb.ResourceSearchResult{
				createTestAsset("asset1", "proj-A", "ACTIVE", "1.2.3.4", baseTime),
				createTestAsset("asset2", "proj-B", "ACTIVE", "5.6.7.8", baseTime),
				createTestAsset("asset3", "proj-C", "ACTIVE", "9.10.11.12", baseTime),
				createTestAsset("asset4", "proj-D", "ACTIVE", "13.14.15.16", baseTime),
			},
			expectedCount: 2,
			expectedAssets: []ProcessedAsset{
				{
					Name:      "asset1",
					Location:  "us-central1",
					Project:   "proj-A",
					IPAddress: "1.2.3.4",
					Status:    "ACTIVE",
					CreatedAt: "2024-01-10 12:00:00",
				},
				{
					Name:      "asset3",
					Location:  "us-central1",
					Project:   "proj-C",
					IPAddress: "9.10.11.12",
					Status:    "ACTIVE",
					CreatedAt: "2024-01-10 12:00:00",
				},
			},
		},
		{
			name: "include specific projects only",
			config: &Config{
				OrgID:           "test-org",
				ExcludeReserved: false,
				ExcludeProjects: "",
				IncludeProjects: "proj-A,proj-C",
			},
			assets: []*assetpb.ResourceSearchResult{
				createTestAsset("asset1", "proj-A", "ACTIVE", "1.2.3.4", baseTime),
				createTestAsset("asset2", "proj-B", "ACTIVE", "5.6.7.8", baseTime),
				createTestAsset("asset3", "proj-C", "ACTIVE", "9.10.11.12", baseTime),
				createTestAsset("asset4", "proj-D", "ACTIVE", "13.14.15.16", baseTime),
			},
			expectedCount: 2,
			expectedAssets: []ProcessedAsset{
				{
					Name:      "asset1",
					Location:  "us-central1",
					Project:   "proj-A",
					IPAddress: "1.2.3.4",
					Status:    "ACTIVE",
					CreatedAt: "2024-01-10 12:00:00",
				},
				{
					Name:      "asset3",
					Location:  "us-central1",
					Project:   "proj-C",
					IPAddress: "9.10.11.12",
					Status:    "ACTIVE",
					CreatedAt: "2024-01-10 12:00:00",
				},
			},
		},
		{
			name: "combined filtering - exclude reserved and include specific projects",
			config: &Config{
				OrgID:           "test-org",
				ExcludeReserved: true,
				ExcludeProjects: "",
				IncludeProjects: "proj-A,proj-B",
			},
			assets: []*assetpb.ResourceSearchResult{
				createTestAsset("asset1", "proj-A", "ACTIVE", "1.2.3.4", baseTime),
				createTestAsset("asset2", "proj-B", "RESERVED", "5.6.7.8", baseTime),
				createTestAsset("asset3", "proj-C", "ACTIVE", "9.10.11.12", baseTime),
				createTestAsset("asset4", "proj-A", "RESERVED", "13.14.15.16", baseTime),
			},
			expectedCount: 1,
			expectedAssets: []ProcessedAsset{
				{
					Name:      "asset1",
					Location:  "us-central1",
					Project:   "proj-A",
					IPAddress: "1.2.3.4",
					Status:    "ACTIVE",
					CreatedAt: "2024-01-10 12:00:00",
				},
			},
		},
		{
			name: "empty iterator",
			config: &Config{
				OrgID:           "test-org",
				ExcludeReserved: false,
				ExcludeProjects: "",
				IncludeProjects: "",
			},
			assets:         []*assetpb.ResourceSearchResult{},
			expectedCount:  0,
			expectedAssets: []ProcessedAsset{},
		},
		{
			name: "asset without IP address",
			config: &Config{
				OrgID:           "test-org",
				ExcludeReserved: false,
				ExcludeProjects: "",
				IncludeProjects: "",
			},
			assets: []*assetpb.ResourceSearchResult{
				createTestAsset("asset1", "proj-A", "ACTIVE", "", baseTime),
			},
			expectedCount: 1,
			expectedAssets: []ProcessedAsset{
				{
					Name:      "asset1",
					Location:  "us-central1",
					Project:   "proj-A",
					IPAddress: "N/A",
					Status:    "ACTIVE",
					CreatedAt: "2024-01-10 12:00:00",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewAssetProcessor(ctx, logger, tt.config)
			iterator := &mockAssetIterator{
				assets: tt.assets,
			}

			results, err := processor.ProcessAssets(ctx, iterator)
			if err != nil {
				t.Fatalf("ProcessAssets failed: %v", err)
			}

			if len(results) != tt.expectedCount {
				t.Errorf("expected %d assets, got %d", tt.expectedCount, len(results))
			}

			// Verify each asset matches expected
			for i, expected := range tt.expectedAssets {
				if i >= len(results) {
					t.Errorf("missing expected asset at index %d", i)

					continue
				}

				actual := results[i]
				if actual.Name != expected.Name {
					t.Errorf("asset[%d] Name = %v, want %v", i, actual.Name, expected.Name)
				}

				if actual.Location != expected.Location {
					t.Errorf("asset[%d] Location = %v, want %v", i, actual.Location, expected.Location)
				}

				if actual.Project != expected.Project {
					t.Errorf("asset[%d] Project = %v, want %v", i, actual.Project, expected.Project)
				}

				if actual.IPAddress != expected.IPAddress {
					t.Errorf("asset[%d] IPAddress = %v, want %v", i, actual.IPAddress, expected.IPAddress)
				}

				if actual.Status != expected.Status {
					t.Errorf("asset[%d] Status = %v, want %v", i, actual.Status, expected.Status)
				}

				if actual.CreatedAt != expected.CreatedAt {
					t.Errorf("asset[%d] CreatedAt = %v, want %v", i, actual.CreatedAt, expected.CreatedAt)
				}
			}
		})
	}
}

// TestProcessAssets_Error tests error handling in ProcessAssets.
func TestProcessAssets_Error(t *testing.T) {
	ctx := t.Context()
	logger := slog.New(slog.DiscardHandler)
	config := &Config{
		OrgID: "test-org",
	}

	processor := NewAssetProcessor(ctx, logger, config)
	iterator := &mockAssetIterator{
		assets: []*assetpb.ResourceSearchResult{
			createTestAsset("asset1", "proj-A", "ACTIVE", "1.2.3.4", time.Now()),
		},
		err: errSimulatedAPI,
	}

	_, err := processor.ProcessAssets(ctx, iterator)
	if err == nil {
		t.Error("expected error, got nil")
	}

	expectedErr := fmt.Sprintf("failed to create asset client: %v", errSimulatedAPI)
	if err.Error() != expectedErr {
		t.Errorf("unexpected error message: got %v, want %v", err, expectedErr)
	}
}
