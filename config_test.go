package main

import (
	"errors"
	"os"
	"os/exec"
	"reflect"
	"testing"
)

const (
	defaultOutputFormat = "table"
)

func cleanEnvVars() {
	_ = os.Unsetenv("ASSET_WATCHER_ORG_ID")
	_ = os.Unsetenv("ASSET_WATCHER_DEBUG")
	_ = os.Unsetenv("ASSET_WATCHER_OUTPUT_FORMAT")
	_ = os.Unsetenv("ASSET_WATCHER_EXCLUDE_RESERVED")
	_ = os.Unsetenv("ASSET_WATCHER_EXCLUDE_PROJECTS")
	_ = os.Unsetenv("ASSET_WATCHER_INCLUDE_PROJECTS")
}

// TestGetConfig_Defaults tests the default values for non-required fields.
// Required fields must be set for GetConfig not to call log.Fatalf.
func TestGetConfig_Defaults(t *testing.T) {
	cleanEnvVars()

	t.Setenv("ASSET_WATCHER_ORG_ID", "test-org-id-defaults")

	cfg := GetConfig()

	if cfg.OrgID != "test-org-id-defaults" {
		t.Errorf("expected OrgID to be 'test-org-id-defaults', got '%s'", cfg.OrgID)
	}

	if cfg.Debug != false {
		t.Errorf("expected Debug default to be %t, got %t", ConfigDefaults.Debug, cfg.Debug)
	}

	if cfg.OutputFormat != ConfigDefaults.OutputFormat {
		t.Errorf("expected OutputFormat default to be '%s', got '%s'", ConfigDefaults.OutputFormat, cfg.OutputFormat)
	}

	if cfg.ExcludeReserved != false {
		t.Errorf("expected ExcludeReserved default to be %t, got %t", ConfigDefaults.ExcludeReserved, cfg.ExcludeReserved)
	}

	if cfg.ExcludeProjects != "" {
		t.Errorf("expected ExcludeProjects default to be '%s' string, got '%s'", ConfigDefaults.ExcludeProjects, cfg.ExcludeProjects)
	}

	if cfg.IncludeProjects != "" {
		t.Errorf("expected IncludeProjects default to be '%s' string, got '%s'", ConfigDefaults.IncludeProjects, cfg.IncludeProjects)
	}
}

// TestGetConfig_LoadFromEnv tests loading configuration from environment variables.
func TestGetConfig_LoadFromEnv(t *testing.T) {
	cleanEnvVars()

	expectedConfig := Config{
		OrgID:           "env-org-id",
		Debug:           true,
		OutputFormat:    "json",
		ExcludeReserved: true,
		ExcludeProjects: "proj1,proj2",
		IncludeProjects: "", // Will be empty as ExcludeProjects is set
	}

	t.Setenv("ASSET_WATCHER_ORG_ID", expectedConfig.OrgID)
	t.Setenv("ASSET_WATCHER_DEBUG", "true")
	t.Setenv("ASSET_WATCHER_OUTPUT_FORMAT", expectedConfig.OutputFormat)
	t.Setenv("ASSET_WATCHER_EXCLUDE_RESERVED", "true")
	t.Setenv("ASSET_WATCHER_EXCLUDE_PROJECTS", expectedConfig.ExcludeProjects)

	cfg := GetConfig()

	if !reflect.DeepEqual(*cfg, expectedConfig) {
		t.Errorf("expected config %+v, got %+v", expectedConfig, *cfg)
	}
}

func TestGetConfig_LoadFromEnv_Include(t *testing.T) {
	cleanEnvVars()

	expectedConfig := Config{
		OrgID:           "env-org-id-include",
		Debug:           false,               // Testing explicit false
		OutputFormat:    defaultOutputFormat, // Testing explicit table
		ExcludeReserved: false,               // Testing explicit false
		ExcludeProjects: "",
		IncludeProjects: "proj3,proj4",
	}

	t.Setenv("ASSET_WATCHER_ORG_ID", expectedConfig.OrgID)
	t.Setenv("ASSET_WATCHER_DEBUG", "false")
	t.Setenv("ASSET_WATCHER_OUTPUT_FORMAT", defaultOutputFormat)
	t.Setenv("ASSET_WATCHER_EXCLUDE_RESERVED", "false")
	t.Setenv("ASSET_WATCHER_INCLUDE_PROJECTS", expectedConfig.IncludeProjects)

	cfg := GetConfig()

	if !reflect.DeepEqual(*cfg, expectedConfig) {
		t.Errorf("expected config %+v, got %+v", expectedConfig, *cfg)
	}
}

func runTestExpectingFatal(t *testing.T, testName string, setupFunc func()) {
	t.Helper()
	// Check if we are in the subprocess
	if os.Getenv("BE_FATAL_TESTER") == "1" {
		setupFunc() // Setup env vars for the specific test case
		GetConfig() // This should call log.Fatalf and exit

		return // Should not be reached in subprocess
	}

	// Prepare to run the test in a subprocess
	cmd := exec.Command(os.Args[0], "-test.run="+testName) //nolint:gosec // Will rerun only the current test function
	cmd.Env = append(os.Environ(), "BE_FATAL_TESTER=1")    // Set marker for subprocess

	// We need to pass through necessary env vars for the test runner itself,
	// but clear/set specific ASSET_WATCHER_* vars within setupFunc in the subprocess.
	// The setupFunc will be responsible for setting the environment that causes GetConfig to fail.

	err := cmd.Run()
	// Check if the command exited with a non-zero status, indicating log.Fatalf was likely called.
	exitErr := &exec.ExitError{}
	if errors.As(err, &exitErr) {
		// ExitError and non-zero exit status means the program exited as expected.
		// We could also check e.Sys().(syscall.WaitStatus).ExitStatus() == 1 if needed for more specific exit codes.
		return // Test passed
	}
	// If err is nil, or it's an ExitError but with success (exit code 0), then GetConfig did not call log.Fatalf.
	var output []byte

	exitErr = &exec.ExitError{}
	if errors.As(err, &exitErr) {
		output = exitErr.Stderr // Stderr often contains the log.Fatalf message
	} else if err != nil { // Other error during cmd.Run
		t.Fatalf("%s: GetConfig did not call log.Fatalf as expected. Command execution failed: %v. Output: %s", testName, err, output)

		return
	}

	// If we reach here, cmd.Run() succeeded (exit code 0) or had an unexpected error.
	t.Fatalf("%s: GetConfig did not call log.Fatalf as expected. Process exited cleanly or with an unexpected error. Output: %s", testName, output)
}

func TestGetConfig_MissingRequiredEnv(t *testing.T) {
	runTestExpectingFatal(t, "TestGetConfig_MissingRequiredEnv", func() {
		cleanEnvVars()
	})
}

func TestGetConfig_ExcludeAndIncludeProjectsSet(t *testing.T) {
	runTestExpectingFatal(t, "TestGetConfig_ExcludeAndIncludeProjectsSet", func() {
		cleanEnvVars()
		t.Setenv("ASSET_WATCHER_ORG_ID", "test-org-for-exclude-include")
		t.Setenv("ASSET_WATCHER_EXCLUDE_PROJECTS", "projA")
		t.Setenv("ASSET_WATCHER_INCLUDE_PROJECTS", "projB")
	})
}

func TestGetConfig_InvalidOutputFormat(t *testing.T) {
	runTestExpectingFatal(t, "TestGetConfig_InvalidOutputFormat", func() {
		cleanEnvVars()
		t.Setenv("ASSET_WATCHER_ORG_ID", "test-org-for-invalid-format")
		t.Setenv("ASSET_WATCHER_OUTPUT_FORMAT", "invalid-format")
	})
}
