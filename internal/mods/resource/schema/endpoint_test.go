package schema

import (
	"encoding/json"
	"testing"
)

func TestEndpointForm_FillTo(t *testing.T) {
	tests := []struct {
		name             string
		form             EndpointForm
		expectedHeaders  json.RawMessage
		expectedMetadata json.RawMessage
	}{
		{
			name: "Normal non-empty json values",
			form: EndpointForm{
				Headers:  json.RawMessage(`{"X-Custom-Header":"value"}`),
				Metadata: json.RawMessage(`{"timeout":30}`),
			},
			expectedHeaders:  json.RawMessage(`{"X-Custom-Header":"value"}`),
			expectedMetadata: json.RawMessage(`{"timeout":30}`),
		},
		{
			name: "Literal null values",
			form: EndpointForm{
				Headers:  json.RawMessage(`null`),
				Metadata: json.RawMessage(`null`),
			},
			expectedHeaders:  nil,
			expectedMetadata: nil,
		},
		{
			name: "Empty raw message slices",
			form: EndpointForm{
				Headers:  json.RawMessage(nil),
				Metadata: json.RawMessage([]byte{}),
			},
			expectedHeaders:  nil,
			expectedMetadata: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := &Endpoint{}
			err := tt.form.FillTo(endpoint)
			if err != nil {
				t.Fatalf("FillTo failed: %v", err)
			}

			if string(endpoint.Headers) != string(tt.expectedHeaders) {
				t.Errorf("Headers mismatch, got: %s, want: %s", string(endpoint.Headers), string(tt.expectedHeaders))
			}
			if string(endpoint.Metadata) != string(tt.expectedMetadata) {
				t.Errorf("Metadata mismatch, got: %s, want: %s", string(endpoint.Metadata), string(tt.expectedMetadata))
			}
		})
	}
}
