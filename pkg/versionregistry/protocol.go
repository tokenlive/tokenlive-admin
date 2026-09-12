// Package versionregistry stores short-lived, deployment-scoped Gateway versions.
package versionregistry

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
)

// TTL is the window in which a Gateway report remains active.
const TTL = 3 * time.Minute

var namespacePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// Node is the version-report wire protocol, independent of request metrics.
type Node struct {
	SchemaVersion int    `json:"schema_version"`
	Namespace     string `json:"namespace"`
	InstanceID    string `json:"instance_id"`
	Version       string `json:"version"`
	BuildKind     string `json:"build_kind"`
}

// Store manages active reports in one deployment namespace.
type Store interface {
	Upsert(context.Context, Node) error
	List(context.Context) ([]Node, error)
	Delete(context.Context, string) error
}

// Validate checks a wire report against the configured deployment namespace.
// The wire namespace must be explicit. Empty or unrecognized version text is
// valid information; version comparison is separate from protocol validation.
func Validate(node Node, expectedNamespace string) error {
	namespace, err := configuredNamespace(expectedNamespace)
	if err != nil {
		return err
	}
	if node.SchemaVersion != 1 {
		return fmt.Errorf("versionregistry: schema_version must be 1")
	}
	if !namespacePattern.MatchString(node.Namespace) || node.Namespace != namespace {
		return fmt.Errorf("versionregistry: namespace must match the configured deployment")
	}
	if _, err := canonicalInstanceID(node.InstanceID); err != nil {
		return err
	}
	if len(node.Version) > 128 {
		return fmt.Errorf("versionregistry: version exceeds 128 bytes")
	}
	if node.BuildKind != "release" && node.BuildKind != "dev" {
		return fmt.Errorf("versionregistry: build_kind must be release or dev")
	}
	return nil
}

func configuredNamespace(namespace string) (string, error) {
	if namespace == "" {
		namespace = "default"
	}
	if !namespacePattern.MatchString(namespace) {
		return "", fmt.Errorf("versionregistry: namespace must match [A-Za-z0-9_-]{1,64}")
	}
	return namespace, nil
}

func canonicalInstanceID(instanceID string) (string, error) {
	if len(instanceID) != 36 {
		return "", fmt.Errorf("versionregistry: instance_id must be a hyphenated UUID")
	}
	id, err := uuid.Parse(instanceID)
	if err != nil {
		return "", fmt.Errorf("versionregistry: instance_id must be a valid UUID: %w", err)
	}
	return id.String(), nil
}
