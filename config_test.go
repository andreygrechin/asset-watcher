package main

import (
	"os"
	"os/exec"
	"reflect"
	"testing"
)

// Helper to set environment variables and register cleanup.
func setEnvVar(t *testing.T, key, value string) {
	t.Helper()
	originalValue, originalExists := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("failed to set env var %s: %v", key, err)
	}
	t.Cleanup(func() {
		if originalExists {
			if err := os.Setenv(key, originalValue); err != nil {
				t.Errorf("failed to restore env var %s: %v", key, err)
			}
		} else {
			if err := os.Unsetenv(key); err != nil {
				t.Errorf("failed to unset env var %s: %v", key, err)
			}
		}
	})
}

// Helper to clear environment variables and register cleanup.
func clearEnvVar(t *testing.T, key string) {
	t.Helper()
	originalValue, originalExists := os.LookupEnv(key)
	// No need to check error for Unsetenv, it's fine if it's not set.
	_ = os.Unsetenv(key)
	t.Cleanup(func() {
		if originalExists {
			if err := os.Setenv(key, originalValue); err != nil {
				t.Errorf("failed to restore env var %s after clearing: %v", key, err)
			}
		}
	})
}

// TestGetConfig_Defaults tests the default values for non-required fields.
// Required fields must be set for GetConfig not to call log.Fatalf.
func TestGetConfig_Defaults(t *testing.T) {
	// Clear optional environment variables
	clearEnvVar(t, "ASSET_WATCHER_DEBUG")
	clearEnvVar(t, "ASSET_WATCHER_OUTPUT_FORMAT")
	clearEnvVar(t, "ASSET_WATCHER_EXCLUDE_RESERVED")
	clearEnvVar(t, "ASSET_WATCHER_EXCLUDE_PROJECTS")
	clearEnvVar(t, "ASSET_WATCHER_INCLUDE_PROJECTS")

	// Set required environment variables
	setEnvVar(t, "ASSET_WATCHER_ORG_ID", "test-org-id-defaults")

	// Defer cleanup of required var if it wasn't set before this test
	defer clearEnvVar(t, "ASSET_WATCHER_ORG_ID")

	cfg := GetConfig() // GetConfig calls log.Fatalf on error if ORG_ID is missing

	if cfg.OrgID != "test-org-id-defaults" {
		t.Errorf("expected OrgID to be 'test-org-id-defaults', got '%s'", cfg.OrgID)
	}
	if cfg.Debug != false {
		t.Errorf("expected Debug default to be false, got %t", cfg.Debug)
	}
	if cfg.OutputFormat != "table" {
		t.Errorf("expected OutputFormat default to be 'table', got '%s'", cfg.OutputFormat)
	}
	if cfg.ExcludeReserved != false {
		t.Errorf("expected ExcludeReserved default to be false, got %t", cfg.ExcludeReserved)
	}
	if cfg.ExcludeProjects != "" {
		t.Errorf("expected ExcludeProjects default to be empty string, got '%s'", cfg.ExcludeProjects)
	}
	if cfg.IncludeProjects != "" {
		t.Errorf("expected IncludeProjects default to be empty string, got '%s'", cfg.IncludeProjects)
	}
}

// TestGetConfig_LoadFromEnv tests loading configuration from environment variables.
func TestGetConfig_LoadFromEnv(t *testing.T) {
	expectedConfig := Config{
		OrgID:           "env-org-id",
		Debug:           true,
		OutputFormat:    "json",
		ExcludeReserved: true,
		ExcludeProjects: "proj1,proj2",
		IncludeProjects: "", // Will be empty as ExcludeProjects is set
	}

	setEnvVar(t, "ASSET_WATCHER_ORG_ID", expectedConfig.OrgID)
	setEnvVar(t, "ASSET_WATCHER_DEBUG", "true")
	setEnvVar(t, "ASSET_WATCHER_OUTPUT_FORMAT", expectedConfig.OutputFormat)
	setEnvVar(t, "ASSET_WATCHER_EXCLUDE_RESERVED", "true")
	setEnvVar(t, "ASSET_WATCHER_EXCLUDE_PROJECTS", expectedConfig.ExcludeProjects)
	clearEnvVar(t, "ASSET_WATCHER_INCLUDE_PROJECTS") // Ensure include is not set

	cfg := GetConfig()

	if !reflect.DeepEqual(*cfg, expectedConfig) {
		t.Errorf("expected config %+v, got %+v", expectedConfig, *cfg)
	}
}

func TestGetConfig_LoadFromEnv_Include(t *testing.T) {
	expectedConfig := Config{
		OrgID:           "env-org-id-include",
		Debug:           false,   // Testing explicit false
		OutputFormat:    "table", // Testing explicit table
		ExcludeReserved: false,   // Testing explicit false
		ExcludeProjects: "",
		IncludeProjects: "proj3,proj4",
	}

	setEnvVar(t, "ASSET_WATCHER_ORG_ID", expectedConfig.OrgID)
	setEnvVar(t, "ASSET_WATCHER_DEBUG", "false")
	setEnvVar(t, "ASSET_WATCHER_OUTPUT_FORMAT", "table")
	setEnvVar(t, "ASSET_WATCHER_EXCLUDE_RESERVED", "false")
	clearEnvVar(t, "ASSET_WATCHER_EXCLUDE_PROJECTS")
	setEnvVar(t, "ASSET_WATCHER_INCLUDE_PROJECTS", expectedConfig.IncludeProjects)

	cfg := GetConfig()

	if !reflect.DeepEqual(*cfg, expectedConfig) {
		t.Errorf("expected config %+v, got %+v", expectedConfig, *cfg)
	}
}

// runTestExpectingFatal is a helper to test functions that should call log.Fatalf
func runTestExpectingFatal(t *testing.T, testName string, setupFunc func()) {
	t.Helper()
	// Check if we are in the subprocess
	if os.Getenv("BE_FATAL_TESTER") == "1" {
		setupFunc() // Setup env vars for the specific test case
		GetConfig() // This should call log.Fatalf and exit
		return      // Should not be reached in subprocess
	}

	// Prepare to run the test in a subprocess
	cmd := exec.Command(os.Args[0], "-test.run="+testName) // Will rerun only the current test function
	cmd.Env = append(os.Environ(), "BE_FATAL_TESTER=1")    // Set marker for subprocess

	// We need to pass through necessary env vars for the test runner itself,
	// but clear/set specific ASSET_WATCHER_* vars within setupFunc in the subprocess.
	// The setupFunc will be responsible for setting the environment that causes GetConfig to fail.

	err := cmd.Run()
	// Check if the command exited with a non-zero status, indicating log.Fatalf was likely called.
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		// ExitError and non-zero exit status means the program exited as expected.
		// We could also check e.Sys().(syscall.WaitStatus).ExitStatus() == 1 if needed for more specific exit codes.
		return // Test passed
	}
	// If err is nil, or it's an ExitError but with success (exit code 0), then GetConfig did not call log.Fatalf.
	var output []byte
	if e, ok := err.(*exec.ExitError); ok {
		output = e.Stderr // Stderr often contains the log.Fatalf message
	} else if err != nil { // Other error during cmd.Run
		t.Fatalf("%s: GetConfig did not call log.Fatalf as expected. Command execution failed: %v. Output: %s", testName, err, output)
		return
	}

	// If we reach here, cmd.Run() succeeded (exit code 0) or had an unexpected error.
	t.Fatalf("%s: GetConfig did not call log.Fatalf as expected. Process exited cleanly or with an unexpected error. Output: %s", testName, output)
}

func TestGetConfig_MissingRequiredEnv(t *testing.T) {
	runTestExpectingFatal(t, "TestGetConfig_MissingRequiredEnv", func() {
		// Inside the subprocess, clear the required env var
		// All other ASSET_WATCHER vars should ideally be cleared or set to defaults
		// to ensure this is the specific condition causing failure.
		// The `env` package will pick up other existing ASSET_WATCHER_* vars if not cleared.
		os.Unsetenv("ASSET_WATCHER_ORG_ID")
		os.Unsetenv("ASSET_WATCHER_DEBUG")
		os.Unsetenv("ASSET_WATCHER_OUTPUT_FORMAT")
		os.Unsetenv("ASSET_WATCHER_EXCLUDE_RESERVED")
		os.Unsetenv("ASSET_WATCHER_EXCLUDE_PROJECTS")
		os.Unsetenv("ASSET_WATCHER_INCLUDE_PROJECTS")
	})
}

func TestGetConfig_ExcludeAndIncludeProjectsSet(t *testing.T) {
	runTestExpectingFatal(t, "TestGetConfig_ExcludeAndIncludeProjectsSet", func() {
		os.Setenv("ASSET_WATCHER_ORG_ID", "test-org-for-exclude-include") // Required
		os.Setenv("ASSET_WATCHER_EXCLUDE_PROJECTS", "projA")
		os.Setenv("ASSET_WATCHER_INCLUDE_PROJECTS", "projB")
	})
}

func TestGetConfig_InvalidOutputFormat(t *testing.T) {
	runTestExpectingFatal(t, "TestGetConfig_InvalidOutputFormat", func() {
		os.Setenv("ASSET_WATCHER_ORG_ID", "test-org-for-invalid-format") // Required
		os.Setenv("ASSET_WATCHER_OUTPUT_FORMAT", "invalid-format")
		// Ensure other conflicting settings are not present
		os.Unsetenv("ASSET_WATCHER_EXCLUDE_PROJECTS")
		os.Unsetenv("ASSET_WATCHER_INCLUDE_PROJECTS")
	})
}

// (Removed unused `contains` function)
