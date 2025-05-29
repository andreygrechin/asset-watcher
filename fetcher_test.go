package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"os"
	"testing"

	"cloud.google.com/go/asset/apiv1/assetpb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// fakeAssetServer is a mock implementation of the AssetServiceServer.
type fakeAssetServer struct {
	assetpb.UnimplementedAssetServiceServer
}

// SearchAllResources is a mock implementation of the SearchAllResources RPC.
func (s *fakeAssetServer) SearchAllResources(ctx context.Context, req *assetpb.SearchAllResourcesRequest) (*assetpb.SearchAllResourcesResponse, error) {
	log.Println("fakeAssetServer.SearchAllResources called")

	// Create a sample asset.
	asset := &assetpb.ResourceSearchResult{
		DisplayName:            "Test Asset 1",
		Location:               "global",
		State:                  "ACTIVE",
		ParentFullResourceName: "//cloudresourcemanager.googleapis.com/projects/test-project-1",
	}

	// Create a response and populate it with the sample asset.
	response := &assetpb.SearchAllResourcesResponse{
		Results: []*assetpb.ResourceSearchResult{asset},
	}

	return response, nil
}

func TestFetchAssets_WithFakeServer(t *testing.T) {
	testServer := &fakeAssetServer{}

	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	fakeServerAddr := l.Addr().String()

	gsrv := grpc.NewServer()
	assetpb.RegisterAssetServiceServer(gsrv, testServer)

	go func() {
		if err := gsrv.Serve(l); err != nil {
			// This will cause the test to fail if the server fails to start.
			// For a more robust solution in complex scenarios, consider error channels.
			panic(err)
		}
	}()
	defer gsrv.Stop() // Ensure server is stopped after the test.

	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	// Assuming Config struct is defined in the main package (e.g. in main.go or config.go)
	// If not, this test might need a minimal local definition or adjustment.
	cfg := &Config{OrgID: "test-org"}

	fetcher, err := NewGoogleAssetFetcher(ctx, logger, cfg,
		option.WithEndpoint(fakeServerAddr),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		t.Fatalf("NewGoogleAssetFetcher failed: %v", err)
	}
	defer fetcher.Close()

	iterator := fetcher.FetchAssets(ctx)
	assetsFound := 0
	for {
		asset, err := iterator.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("iterator.Next() failed: %v", err)
		}
		assetsFound++

		// Assertions based on the fakeAssetServer's response
		if asset.DisplayName != "Test Asset 1" {
			t.Errorf("Expected DisplayName 'Test Asset 1', got '%s'", asset.DisplayName)
		}
		if asset.Location != "global" {
			t.Errorf("Expected Location 'global', got '%s'", asset.Location)
		}
		if asset.State != "ACTIVE" {
			t.Errorf("Expected State 'ACTIVE', got '%s'", asset.State)
		}
		if asset.ParentFullResourceName != "//cloudresourcemanager.googleapis.com/projects/test-project-1" {
			t.Errorf("Expected ParentFullResourceName '//cloudresourcemanager.googleapis.com/projects/test-project-1', got '%s'", asset.ParentFullResourceName)
		}
	}

	if assetsFound != 1 { // fakeAssetServer is configured to return 1 asset
		t.Errorf("Expected to find 1 asset, found %d", assetsFound)
	}
}
