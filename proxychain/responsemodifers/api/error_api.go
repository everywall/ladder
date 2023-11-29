package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
)

type Error struct {
	Success bool         `json:"success"`
	Error   ErrorDetails `json:"error"`
}

type ErrorDetails struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Cause   string `json:"cause"`
}

func CreateAPIErrReader(err error) io.ReadCloser {
	if err == nil {
		return io.NopCloser(bytes.NewBufferString(`{"success":false, "error": "No error provided"}`))
	}

	baseErr := getBaseError(err)
	apiErr := Error{
		Success: false,
		Error: ErrorDetails{
			Message: err.Error(),
			Type:    reflect.TypeOf(err).String(),
			Cause:   baseErr.Error(),
		},
	}

	// Serialize the APIError into JSON
	jsonData, jsonErr := json.Marshal(apiErr)
	if jsonErr != nil {
		return io.NopCloser(bytes.NewBufferString(`{"success":false, "error": "Failed to serialize error"}`))
	}

	// Return the JSON data as an io.ReadCloser
	return io.NopCloser(bytes.NewBuffer(jsonData))
}

func getBaseError(err error) error {
	for {
		unwrapped := errors.Unwrap(err)
		if unwrapped == nil {
			return err
		}

		err = unwrapped
	}
}
