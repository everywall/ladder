package responsemodifers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"testing"
)

func TestCreateAPIErrReader(t *testing.T) {
	_, baseErr := url.Parse("://this is an invalid url")
	wrappedErr := fmt.Errorf("wrapped error: %w", baseErr)

	readCloser := CreateAPIErrReader(wrappedErr)
	defer readCloser.Close()

	// Read and unmarshal the JSON output
	data, err := io.ReadAll(readCloser)
	if err != nil {
		t.Fatalf("Failed to read from ReadCloser: %v", err)
	}
	fmt.Println(string(data))

	var apiErr APIError
	err = json.Unmarshal(data, &apiErr)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify the structure of the APIError
	if apiErr.Success {
		t.Errorf("Expected Success to be false, got true")
	}

	if apiErr.Error.Message != wrappedErr.Error() {
		t.Errorf("Expected error message to be '%v', got '%v'", wrappedErr.Error(), apiErr.Error.Message)
	}
}

func TestCreateAPIErrReader2(t *testing.T) {
	_, baseErr := url.Parse("://this is an invalid url")

	readCloser := CreateAPIErrReader(baseErr)
	defer readCloser.Close()

	// Read and unmarshal the JSON output
	data, err := io.ReadAll(readCloser)
	if err != nil {
		t.Fatalf("Failed to read from ReadCloser: %v", err)
	}
	fmt.Println(string(data))

	var apiErr APIError
	err = json.Unmarshal(data, &apiErr)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify the structure of the APIError
	if apiErr.Success {
		t.Errorf("Expected Success to be false, got true")
	}

	if apiErr.Error.Message != baseErr.Error() {
		t.Errorf("Expected error message to be '%v', got '%v'", baseErr.Error(), apiErr.Error.Message)
	}
}
