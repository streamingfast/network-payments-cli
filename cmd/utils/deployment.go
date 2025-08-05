package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/google/uuid"
)

func GenerateDeployment() (string, error) {
	uniqueId, err := uuid.NewUUID()
	if err != nil {
		panic(err)
	}
	template := fmt.Sprintf(`specVersion: 0.0.5
description: "thegraph.market Payment Gateway usage"
usage:
  uid: %s
`, uniqueId.String())

	hash, err := uploadFile([]byte(template))
	if err != nil {
		return "", fmt.Errorf("error uploading file: %w", err)
	}

	return hash, nil
}

// uploadFile uploads a file using its byte contents and returns only the IPFS Hash.
func uploadFile(fileContents []byte) (string, error) {
	// Create a buffer to store multipart form data
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Create form file field
	formFile, err := writer.CreateFormFile("file", "manifest.yaml")
	if err != nil {
		return "", fmt.Errorf("error creating form file: %w", err)
	}

	// Copy the file contents into the multipart writer
	if _, err := io.Copy(formFile, bytes.NewReader(fileContents)); err != nil {
		return "", fmt.Errorf("error copying file content: %w", err)
	}

	// Close the writer to finalize multipart form data
	writer.Close()

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.thegraph.com/ipfs/api/v0/add", &requestBody)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	// Set content type
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	// Response structure for decoding the JSON response
	type uploadResponse struct {
		Name string `json:"Name"`
		Hash string `json:"Hash"`
	}

	// Decode JSON response
	var uploadResp uploadResponse
	if err := json.Unmarshal(responseBody, &uploadResp); err != nil {
		return "", fmt.Errorf("error unmarshalling response JSON %q, %w", string(responseBody), err)
	}

	// Return only the Hash field
	return uploadResp.Hash, nil
}
