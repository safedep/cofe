package pypi

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPypiPackageManager_DownloadAndGetPackageInfo(t *testing.T) {
	// Create a mock HTTP server to simulate PyPI responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pypi/mockpackage/json" {
			// Simulate a successful package info response
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"info": {"version": "1.0"}, "releases": {"1.0": [{"filename": "mockpackage-1.0.tar.gz", "url": "http://example.com/mockpackage-1.0.tar.gz"}]}}`)
		} else if r.URL.Path == "/pypi/nonexistentpackage/json" {
			// Simulate a nonexistent package response
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"message": "Package not found"}`)
		} else {
			// Simulate a successful gzip archive response
			w.Header().Set("Content-Type", "application/gzip")
			w.WriteHeader(http.StatusOK)

			// Create a valid tar.gz archive with a dummy file
			gzipWriter := gzip.NewWriter(w)
			tarWriter := tar.NewWriter(gzipWriter)

			// Create a dummy file entry in the archive
			dummyFileHeader := &tar.Header{
				Name: "dummy.txt",
				Mode: 0644,
				Size: int64(len([]byte("This is a dummy file content"))),
			}
			if err := tarWriter.WriteHeader(dummyFileHeader); err != nil {
				t.Fatalf("Failed to write tar header: %v", err)
			}
			if _, err := tarWriter.Write([]byte("This is a dummy file content")); err != nil {
				t.Fatalf("Failed to write to tar archive: %v", err)
			}

			// Close the tar.gz archive to finalize it
			if err := tarWriter.Close(); err != nil {
				t.Fatalf("Failed to close tar archive: %v", err)
			}
			if err := gzipWriter.Close(); err != nil {
				t.Fatalf("Failed to close gzip archive: %v", err)
			}

		}
	}))
	defer server.Close()

	// Set the server URL as the PyPI endpoint for testing
	pypiURL := server.URL

	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Initialize the package manager
	manager := NewPrivatePypiPackageManager([]string{pypiURL})

	// Test case 1: Successful download and package info retrieval
	data, extractDir, err := manager.DownloadAndGetPackageInfo(tempDir, "mockpackage", "")
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	if len(data) == 0 {
		t.Error("Expected package data, but got an empty map")
	}
	if extractDir == "" {
		t.Error("Expected a non-empty extract directory, but got an empty string")
	}

	// Test case 2: Nonexistent package
	_, _, err = manager.DownloadAndGetPackageInfo(tempDir, "nonexistentpackage", "")
	if err == nil {
		t.Error("Expected an error for nonexistent package, but got nil")
	}

	// Test case 3: Server error
	_, _, err = manager.DownloadAndGetPackageInfo(tempDir, "servererrorpackage", "")
	if err == nil {
		t.Error("Expected an error for server error, but got nil")
	}
}
