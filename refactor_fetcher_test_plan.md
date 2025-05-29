# Plan to Refactor `fetcher_test.go` for Reusable Fake Server

**1. Objective:**
Refactor [`fetcher_test.go`](fetcher_test.go) to make the fake gRPC server initialization reusable for future tests by introducing a parameterized helper function. This helper will allow tests to specify the data the fake server should return.

**2. Current Structure Analysis (as of initial review):**

* The test `TestFetchAssets_WithFakeServer` (lines [`43-110`](fetcher_test.go:43)) embeds the setup, running, and teardown of a `fakeAssetServer`.
* The `fakeAssetServer` struct (lines [`19-21`](fetcher_test.go:19)) and its `SearchAllResources` method (lines [`23-41`](fetcher_test.go:23)) currently use hardcoded data.

**3. Proposed Refactoring Steps:**

    **a. Modify `fakeAssetServer` Struct:**
    Add a field to `fakeAssetServer` to hold the assets it should return.
    ```go
    // fetcher_test.go
    // fakeAssetServer is a mock implementation of the AssetServiceServer.
    type fakeAssetServer struct {
        assetpb.UnimplementedAssetServiceServer
        assetsToServe []*assetpb.ResourceSearchResult // New field
    }
    ```

    **b. Modify `fakeAssetServer.SearchAllResources` Method:**
    This method will now use the `assetsToServe` field.
    ```go
    // fetcher_test.go
    // SearchAllResources is a mock implementation of the SearchAllResources RPC.
    func (s *fakeAssetServer) SearchAllResources(ctx context.Context, req *assetpb.SearchAllResourcesRequest) (*assetpb.SearchAllResourcesResponse, error) {
        log.Println("fakeAssetServer.SearchAllResources called")
        response := &assetpb.SearchAllResourcesResponse{
            Results: s.assetsToServe, // Use the new field
        }
        return response, nil
    }
    ```

    **c. Create New Helper Function `setupFakeAssetServer`:**
    This function will encapsulate the fake server setup and teardown logic and accept asset data.
    ```go
    // fetcher_test.go
    // setupFakeAssetServer initializes and starts a fake gRPC asset server
    // configured with the provided assets.
    // It returns the server's address (string) and a cleanup function (func()).
    // The cleanup function should be called via defer to ensure the server is stopped.
    func setupFakeAssetServer(t *testing.T, assets []*assetpb.ResourceSearchResult) (serverAddr string, cleanupFunc func()) {
        t.Helper() // Mark as test helper

        testServer := &fakeAssetServer{
            assetsToServe: assets,
        }

        l, err := net.Listen("tcp", "localhost:0")
        if err != nil {
            t.Fatalf("setupFakeAssetServer: failed to listen: %v", err)
        }
        serverAddr = l.Addr().String()

        gsrv := grpc.NewServer()
        assetpb.RegisterAssetServiceServer(gsrv, testServer)

        go func() {
            if err := gsrv.Serve(l); err != nil {
                // Use t.Errorf or similar for goroutine errors in tests
                // to avoid panicking the whole test suite if not desired.
                // For simplicity here, we'll let it panic if Serve fails catastrophically.
                // A more robust solution might involve an error channel.
                log.Printf("setupFakeAssetServer: gsrv.Serve failed: %v", err)
            }
        }()

        cleanupFunc = func() {
            gsrv.Stop()
        }

        return serverAddr, cleanupFunc
    }
    ```

    **d. Update Existing Test `TestFetchAssets_WithFakeServer`:**
    Modify the test to use the new helper function.
    ```go
    // fetcher_test.go
    func TestFetchAssets_WithFakeServer(t *testing.T) {
        // Define the specific assets this test expects
        expectedAssets := []*assetpb.ResourceSearchResult{
            {
                DisplayName:            "Test Asset 1",
                Location:               "global",
                State:                  "ACTIVE",
                ParentFullResourceName: "//cloudresourcemanager.googleapis.com/projects/test-project-1",
            },
        }

        fakeServerAddr, cleanup := setupFakeAssetServer(t, expectedAssets)
        defer cleanup()

        ctx := context.Background()
        logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
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

        assets := fetcher.FetchAssets(ctx)
        assetsFound := 0
        for {
            asset, err := assets.Next()
            if err == iterator.Done {
                break
            }
            if err != nil {
                t.Fatalf("iterator.Next() failed: %v", err)
            }
            assetsFound++

            // Assertions based on the expectedAssets
            if len(expectedAssets) == 1 { // Assuming one asset for this specific test
                if asset.DisplayName != expectedAssets[0].DisplayName {
                    t.Errorf("Expected DisplayName '%s', got '%s'", expectedAssets[0].DisplayName, asset.DisplayName)
                }
                if asset.Location != expectedAssets[0].Location {
                    t.Errorf("Expected Location '%s', got '%s'", expectedAssets[0].Location, asset.Location)
                }
                if asset.State != expectedAssets[0].State {
                    t.Errorf("Expected State '%s', got '%s'", expectedAssets[0].State, asset.State)
                }
                if asset.ParentFullResourceName != expectedAssets[0].ParentFullResourceName {
                    t.Errorf("Expected ParentFullResourceName '%s', got '%s'", expectedAssets[0].ParentFullResourceName, asset.ParentFullResourceName)
                }
            }
        }

        if assetsFound != len(expectedAssets) {
            t.Errorf("Expected to find %d asset(s), found %d", len(expectedAssets), assetsFound)
        }
    }
    ```

**4. Benefits:**

* Enhanced Reusability & Flexibility: Different tests can easily set up the fake server to return specific datasets.
* Improved Clarity: Test cases will explicitly define the data they expect.
* Maintainability: Server setup logic is centralized.

**5. Visual Plan (Mermaid Diagram):**

    ```mermaid
    graph TD
        subgraph Current_File_State["Current fetcher_test.go"]
            direction LR
            F_Test["TestFetchAssets_WithFakeServer()"]
            F_ServerInit["Embedded Fake Server Init"]
            F_FakeAssetServer["fakeAssetServer (hardcoded data)"]
            F_TestLogic["Test Logic"]

            F_Test --> F_ServerInit
            F_ServerInit --> F_FakeAssetServer
            F_Test --> F_TestLogic
        end

        subgraph Planned_Changes["Planned Changes to fetcher_test.go"]
            direction BT
            P_HelperFunc["New: setupFakeAssetServer(t, assetsData)"]
            P_ModifiedFakeAssetServer["Modified: fakeAssetServer (struct + SearchAllResources)"]
            P_RefactoredTest["Refactored: TestFetchAssets_WithFakeServer()"]
            P_TestData["New: Test-Specific Data (e.g., expectedAssets in Test)"]

            P_RefactoredTest -- "Calls with" --> P_TestData
            P_TestData -- "Used by" --> P_HelperFunc
            P_HelperFunc -- "Initializes & Uses" --> P_ModifiedFakeAssetServer
            P_ModifiedFakeAssetServer -- "Serves data for" --> P_RefactoredTest
        end

        F_Test --> P_RefactoredTest
        F_FakeAssetServer --> P_ModifiedFakeAssetServer
        F_ServerInit -- "Replaced by call to" --> P_HelperFunc
    ```
