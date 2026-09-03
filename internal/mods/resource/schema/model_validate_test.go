package schema

import (
	"strings"
	"testing"
)

func TestModelForm_Validate(t *testing.T) {
	validCodes := []struct {
		input    string
		expected string
	}{
		{"gpt-4o", "gpt-4o"},
		{"gpt-3.5-turbo", "gpt-3.5-turbo"},
		{"claude-3.5-sonnet", "claude-3.5-sonnet"},
		{"gemini-1.5-pro", "gemini-1.5-pro"},
		{"qwen2.5-72b-instruct", "qwen2.5-72b-instruct"},
		{"a", "a"},
		{"1", "1"},
		{"model.v1-beta.2", "model.v1-beta.2"},
		{"  gpt-4o  ", "gpt-4o"},
		{"a--b", "a--b"},
		{"a..b", "a..b"},
	}

	for _, tc := range validCodes {
		form := &ModelForm{
			ModelCode: tc.input,
		}
		err := form.Validate()
		if err != nil {
			t.Errorf("expected valid for code %q, got error: %v", tc.input, err)
		}
		if form.ModelCode != tc.expected {
			t.Errorf("expected trimmed code %q, got %q", tc.expected, form.ModelCode)
		}
	}

	invalidCodes := []struct {
		input string
		desc  string
	}{
		{"", "empty string"},
		{"   ", "whitespace only"},
		{"-gpt", "starts with hyphen"},
		{"gpt-", "ends with hyphen"},
		{".gpt", "starts with dot"},
		{"gpt.", "ends with dot"},
		{"-", "hyphen only"},
		{".", "dot only"},
		{"gpt 4o", "contains internal space"},
		{"gpt\t4o", "contains tab"},
		{"gpt\n4o", "contains newline"},
		{"gpt_4o", "contains underscore"},
		{"meta/llama", "contains slash"},
		{"gpt:4o", "contains colon"},
		{"gpt@4o", "contains special character"},
		{strings.Repeat("a", 65), "exceeds 64 characters"},
	}

	for _, tc := range invalidCodes {
		form := &ModelForm{
			ModelCode: tc.input,
		}
		err := form.Validate()
		if err == nil {
			t.Errorf("expected invalid for code %q (%s), got nil error", tc.input, tc.desc)
		}
	}
}
