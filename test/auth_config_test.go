package test

import (
	"os"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestPublicRefreshEndpointSkipsAuthAndCasbin(t *testing.T) {
	middlewareFiles := []string{
		"../configs/dev/middleware.toml",
		"../configs/prod/middleware.toml",
	}

	for _, filename := range middlewareFiles {
		t.Run(filename, func(t *testing.T) {
			buf, err := os.ReadFile(filename)
			if err != nil {
				t.Fatalf("read middleware config: %v", err)
			}

			var cfg struct {
				Middleware struct {
					Auth struct {
						SkippedPathPrefixes []string `toml:"SkippedPathPrefixes"`
					} `toml:"Auth"`
					Casbin struct {
						SkippedPathPrefixes []string `toml:"SkippedPathPrefixes"`
					} `toml:"Casbin"`
				} `toml:"Middleware"`
			}
			if err := toml.Unmarshal(buf, &cfg); err != nil {
				t.Fatalf("parse middleware config: %v", err)
			}

			for name, prefixes := range map[string][]string{
				"auth":   cfg.Middleware.Auth.SkippedPathPrefixes,
				"casbin": cfg.Middleware.Casbin.SkippedPathPrefixes,
			} {
				found := false
				for _, prefix := range prefixes {
					if prefix == "/api/v1/refresh-token" {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("%s middleware does not skip /api/v1/refresh-token: %v", name, prefixes)
				}
			}
		})
	}
}
