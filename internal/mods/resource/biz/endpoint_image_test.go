package biz

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBuildImageGenerationProbe(t *testing.T) {
	reqURL, reqBody := buildImageGenerationProbe(
		"https://api.x.ai/v1/",
		"grok-imagine-image-2.0",
	)

	if reqURL != "https://api.x.ai/v1/images/generations" {
		t.Fatalf("unexpected request URL: %s", reqURL)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(reqBody, &payload); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	if payload["model"] != "grok-imagine-image-2.0" {
		t.Fatalf("unexpected model: %v", payload["model"])
	}
	if payload["prompt"] == "" {
		t.Fatal("image probe prompt must not be empty")
	}
	if payload["response_format"] != "url" {
		t.Fatalf("unexpected response_format: %v", payload["response_format"])
	}
}

func TestBuildImageGenerationProbeKeepsFullEndpointURL(t *testing.T) {
	reqURL, _ := buildImageGenerationProbe(
		"https://api.x.ai/v1/images/generations",
		"grok-imagine-image-2.0",
	)
	if reqURL != "https://api.x.ai/v1/images/generations" {
		t.Fatalf("unexpected request URL: %s", reqURL)
	}
}

func TestValidateImageGenerationProbeResponse(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name: "accepts URL image",
			body: `{"data":[{"url":"https://example.com/image.png"}]}`,
		},
		{
			name: "accepts base64 image",
			body: `{"data":[{"b64_json":"aW1hZ2U="}]}`,
		},
		{
			name:    "rejects empty data",
			body:    `{"data":[]}`,
			wantErr: true,
		},
		{
			name:    "rejects error response",
			body:    `{"error":{"message":"model unavailable"}}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateImageGenerationProbeResponse([]byte(tt.body))
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateImageGenerationProbeResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEndpointProbeTimeoutAllowsImageGeneration(t *testing.T) {
	if got := endpointProbeTimeout(true); got < 60*time.Second {
		t.Fatalf("image generation timeout is too short: %s", got)
	}
	if got := endpointProbeTimeout(false); got != 10*time.Second {
		t.Fatalf("default endpoint probe timeout changed: %s", got)
	}
}
