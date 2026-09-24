package util

import (
	"encoding/json"
	"testing"
)

func TestRawJSON_Scan(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
		wantErr  bool
	}{
		{
			name:     "Scan from []byte",
			input:    []byte(`{"key":"value"}`),
			expected: `{"key":"value"}`,
		},
		{
			name:     "Scan from string (SQLite TEXT)",
			input:    `{"key":"value"}`,
			expected: `{"key":"value"}`,
		},
		{
			name:     "Scan from nil",
			input:    nil,
			expected: "",
		},
		{
			name:    "Scan from invalid type",
			input:   12345,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r RawJSON
			err := r.Scan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Scan() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && string(r) != tt.expected {
				t.Errorf("Scan() got = %s, want = %s", string(r), tt.expected)
			}
		})
	}
}

func TestRawJSON_Value(t *testing.T) {
	var empty RawJSON
	v, err := empty.Value()
	if err != nil || v != nil {
		t.Errorf("empty.Value() = %v, %v, want nil, nil", v, err)
	}

	nonEmpty := RawJSON(`[{"value":"key1"}]`)
	v, err = nonEmpty.Value()
	if err != nil {
		t.Fatalf("nonEmpty.Value() error = %v", err)
	}
	b, ok := v.([]byte)
	if !ok || string(b) != `[{"value":"key1"}]` {
		t.Errorf("nonEmpty.Value() = %s, want [%s]", string(b), `[{"value":"key1"}]`)
	}
}

func TestRawJSON_MarshalUnmarshal(t *testing.T) {
	type wrapper struct {
		Data RawJSON `json:"data"`
	}

	w := wrapper{Data: RawJSON(`{"foo":"bar"}`)}
	b, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if string(b) != `{"data":{"foo":"bar"}}` {
		t.Errorf("Marshal output mismatch: got %s, want %s", string(b), `{"data":{"foo":"bar"}}`)
	}

	var w2 wrapper
	if err := json.Unmarshal(b, &w2); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if string(w2.Data) != `{"foo":"bar"}` {
		t.Errorf("Unmarshal output mismatch: got %s, want %s", string(w2.Data), `{"foo":"bar"}`)
	}
}
