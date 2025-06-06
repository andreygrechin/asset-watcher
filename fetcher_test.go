package main

import (
	"context"
	"errors"
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
	assetsToServe []*assetpb.ResourceSearchResult // New field to hold assets
}

// SearchAllResources is a mock implementation of the SearchAllResources RPC.
// It now returns the assets stored in s.assetsToServe.
func (s *fakeAssetServer) SearchAllResources(_ context.Context, _ *assetpb.SearchAllResourcesRequest) (*assetpb.SearchAllResourcesResponse, error) {
	log.Println("fakeAssetServer.SearchAllResources called")

	response := &assetpb.SearchAllResourcesResponse{
		Results: s.assetsToServe,
	}

	return response, nil
}

// setupFakeAssetServer initializes and starts a fake gRPC asset server
// configured with the provided assets.
// It returns the server's address (string) and a cleanup function (func()).
// The cleanup function should be called via defer to ensure the server is stopped.
func setupFakeAssetServer(t *testing.T, assets []*assetpb.ResourceSearchResult) (string, func()) {
	t.Helper() // Mark as test helper

	testServer := &fakeAssetServer{
		assetsToServe: assets,
	}

	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("setupFakeAssetServer: failed to listen: %v", err)
	}

	serverAddr := l.Addr().String()

	gsrv := grpc.NewServer()
	assetpb.RegisterAssetServiceServer(gsrv, testServer)

	go func() {
		if err := gsrv.Serve(l); err != nil {
			// Log error from goroutine. In a real test, consider more robust error handling
			// like an error channel if the test needs to react to server startup failures.
			log.Printf("setupFakeAssetServer: gsrv.Serve failed: %v", err)
		}
	}()

	cleanupFunc := func() {
		gsrv.Stop()
	}

	return serverAddr, cleanupFunc
}

func TestFetchAssets_WithFakeServer(t *testing.T) {
	// Define the specific assets this test expects the fake server to return
	expectedAssets := []*assetpb.ResourceSearchResult{
		{
			DisplayName:            "Test Asset 1",
			Location:               "global",
			State:                  "ACTIVE",
			ParentFullResourceName: "//cloudresourcemanager.googleapis.com/projects/test-project-1",
		},
	}

	// Setup the fake server using the helper function
	fakeServerAddr, cleanup := setupFakeAssetServer(t, expectedAssets)
	defer cleanup() // Ensure server is stopped after the test

	ctx := t.Context()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	// Assuming Config struct is defined in the main package (e.g. in main.go or config.go)
	cfg := &Config{OrgID: "test-org"}

	fetcher, err := NewGoogleAssetFetcher(ctx, logger, cfg,
		option.WithEndpoint(fakeServerAddr),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		t.Fatalf("NewGoogleAssetFetcher failed: %v", err)
	}

	defer func() {
		if err := fetcher.Close(); err != nil {
			t.Errorf("Failed to close fetcher: %v", err)
		}
	}()

	assetsIterator := fetcher.FetchAssets(ctx)
	assetsFound := 0

	for {
		asset, err := assetsIterator.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			t.Fatalf("assetsIterator.Next() failed: %v", err)
		}

		assetsFound++

		// Assertions based on the expectedAssets provided to the fake server
		// This example assumes we expect one asset, as defined in expectedAssets.
		// For tests with multiple assets, loop through expectedAssets or use a map for lookup.
		switch {
		case len(expectedAssets) == 1:
			if asset.GetDisplayName() != expectedAssets[0].GetDisplayName() {
				t.Errorf("expected DisplayName '%s', got '%s'", expectedAssets[0].GetDisplayName(), asset.GetDisplayName())
			}

			if asset.GetLocation() != expectedAssets[0].GetLocation() {
				t.Errorf("expected Location '%s', got '%s'", expectedAssets[0].GetLocation(), asset.GetLocation())
			}

			if asset.GetState() != expectedAssets[0].GetState() {
				t.Errorf("expected State '%s', got '%s'", expectedAssets[0].GetState(), asset.GetState())
			}

			if asset.GetParentFullResourceName() != expectedAssets[0].GetParentFullResourceName() {
				t.Errorf("expected ParentFullResourceName '%s', got '%s'", expectedAssets[0].GetParentFullResourceName(), asset.GetParentFullResourceName())
			}
		case len(expectedAssets) == 0 && assetsFound > 0:
			t.Errorf("expected 0 assets, but found some.")
		case assetsFound > len(expectedAssets):
			t.Errorf("found more assets than expected. expected %d, found at least %d", len(expectedAssets), assetsFound)
		}
	}

	if assetsFound != len(expectedAssets) {
		t.Errorf("expected to find %d asset(s), found %d", len(expectedAssets), assetsFound)
	}
}
