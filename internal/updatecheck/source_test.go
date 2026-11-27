package updatecheck

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestCandidateFilters(t *testing.T) {
	c, err := ParseGitHubRelease([]byte(`{"tag_name":"v1.2.4","draft":false,"prerelease":false}`), "admin")
	if err != nil || c.Version != "v1.2.4" {
		t.Fatalf("%+v %v", c, err)
	}
	_, err = ParseGitHubRelease([]byte(`{"tag_name":"v1.3.0-rc.1","draft":false,"prerelease":false}`), "gateway")
	if !errors.Is(err, ErrNoCandidate) {
		t.Fatalf("got %v", err)
	}
	if _, err = ParseFormula([]byte("version \"1.2.3\"\nversion \"1.2.4\"\n")); err == nil {
		t.Fatal("duplicate version must fail")
	}
}

func TestCandidateReleaseParsing(t *testing.T) {
	tests := []struct {
		name      string
		component string
		body      string
		want      Candidate
		noTarget  bool
		bad       bool
	}{
		{
			name: "admin ignores untrusted html URL", component: "admin",
			body: `{"tag_name":"v1.2.4","draft":false,"prerelease":false,"html_url":"https://attacker.invalid/release"}`,
			want: Candidate{"v1.2.4", "https://github.com/tokenlive/tokenlive-admin/releases/tag/v1.2.4"},
		},
		{
			name: "gateway normalizes version but preserves original tag", component: "gateway",
			body: `{"tag_name":"1.2.4+build.7","draft":false,"prerelease":false}`,
			want: Candidate{"v1.2.4", "https://github.com/tokenlive/tokenlive-gateway/releases/tag/1.2.4+build.7"},
		},
		{name: "draft", component: "admin", body: `{"tag_name":"v1.2.4","draft":true,"prerelease":false}`, noTarget: true},
		{name: "prerelease flag", component: "admin", body: `{"tag_name":"v1.2.4","draft":false,"prerelease":true}`, noTarget: true},
		{name: "partial stable version", component: "admin", body: `{"tag_name":"v1.2","draft":false,"prerelease":false}`, noTarget: true},
		{name: "malformed tag", component: "admin", body: `{"tag_name":"v1.2.4/../../evil","draft":false,"prerelease":false}`, noTarget: true},
		{name: "development tag", component: "gateway", body: `{"tag_name":"main","draft":false,"prerelease":false}`, noTarget: true},
		{name: "unknown component", component: "../other", body: `{"tag_name":"v1.2.4","draft":false,"prerelease":false}`, bad: true},
		{name: "broken JSON", component: "admin", body: `{`, bad: true},
		{name: "trailing JSON", component: "admin", body: `{"tag_name":"v1.2.4","draft":false,"prerelease":false}{}`, bad: true},
		{name: "missing tag", component: "admin", body: `{"draft":false,"prerelease":false}`, bad: true},
		{name: "missing draft", component: "admin", body: `{"tag_name":"v1.2.4","prerelease":false}`, bad: true},
		{name: "missing prerelease", component: "admin", body: `{"tag_name":"v1.2.4","draft":false}`, bad: true},
		{name: "wrong field type", component: "admin", body: `{"tag_name":123,"draft":false,"prerelease":false}`, bad: true},
		{name: "null JSON", component: "admin", body: `null`, bad: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGitHubRelease([]byte(tt.body), tt.component)
			assertCandidateResult(t, got, err, tt.want, tt.noTarget, tt.bad)
		})
	}
}

func TestGitHubReleaseRedirectParsing(t *testing.T) {
	tests := []struct {
		name      string
		component string
		location  string
		want      Candidate
		noTarget  bool
		bad       bool
	}{
		{
			name: "admin absolute redirect", component: "admin",
			location: "https://github.com/tokenlive/tokenlive-admin/releases/tag/v1.0.7",
			want:     Candidate{"v1.0.7", "https://github.com/tokenlive/tokenlive-admin/releases/tag/v1.0.7"},
		},
		{
			name: "gateway relative redirect with metadata", component: "gateway",
			location: "/tokenlive/tokenlive-gateway/releases/tag/1.2.4%2Bbuild.7",
			want:     Candidate{"v1.2.4", "https://github.com/tokenlive/tokenlive-gateway/releases/tag/1.2.4+build.7"},
		},
		{
			name: "no stable release redirects to releases index", component: "admin",
			location: "https://github.com/tokenlive/tokenlive-admin/releases",
			noTarget: true,
		},
		{
			name: "prerelease tag", component: "gateway",
			location: "https://github.com/tokenlive/tokenlive-gateway/releases/tag/v1.3.0-rc.1",
			noTarget: true,
		},
		{
			name: "nested path tag", component: "admin",
			location: "https://github.com/tokenlive/tokenlive-admin/releases/tag/v1.2.3/extra",
			noTarget: true,
		},
		{name: "missing location", component: "admin", location: "", bad: true},
		{name: "wrong host", component: "admin", location: "https://attacker.invalid/tokenlive/tokenlive-admin/releases/tag/v1.2.3", bad: true},
		{name: "downgrade http", component: "admin", location: "http://github.com/tokenlive/tokenlive-admin/releases/tag/v1.2.3", bad: true},
		{name: "wrong repository", component: "admin", location: "https://github.com/tokenlive/tokenlive-gateway/releases/tag/v1.2.3", bad: true},
		{name: "query string rejected", component: "admin", location: "https://github.com/tokenlive/tokenlive-admin/releases/tag/v1.2.3?x=1", bad: true},
		{name: "fragment rejected", component: "admin", location: "https://github.com/tokenlive/tokenlive-admin/releases/tag/v1.2.3#top", bad: true},
		{name: "userinfo rejected", component: "admin", location: "https://user@github.com/tokenlive/tokenlive-admin/releases/tag/v1.2.3", bad: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGitHubReleaseRedirect(tt.location, tt.component)
			assertCandidateResult(t, got, err, tt.want, tt.noTarget, tt.bad)
		})
	}
}

func TestFormulaParsing(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		want     Candidate
		noTarget bool
		bad      bool
	}{
		{
			name: "indented declaration", body: "class Tokenlive < Formula\n  version \"1.2.3\"\nend\n",
			want: Candidate{"v1.2.3", "https://github.com/tokenlive/tokenlive-standalone/releases/tag/v1.2.3"},
		},
		{
			name: "CRLF comment and unrelated quoted text",
			body: "# version \"9.9.9\"\r\ndesc 'version \"9.9.9\"'\r\n\tversion \"v1.2.3+build.5\" # release\r\n",
			want: Candidate{"v1.2.3", "https://github.com/tokenlive/tokenlive-standalone/releases/tag/v1.2.3+build.5"},
		},
		{
			name: "metadata without v prefix", body: "version \"1.2.3+build.7\"\n",
			want: Candidate{"v1.2.3", "https://github.com/tokenlive/tokenlive-standalone/releases/tag/v1.2.3+build.7"},
		},
		{name: "no declaration", body: "# version \"1.2.3\"\nurl \"https://example.invalid/v1.2.3\"\n", noTarget: true},
		{name: "prerelease", body: "version \"1.3.0-rc.1\"\n", noTarget: true},
		{name: "development version", body: "version \"main\"\n", noTarget: true},
		{name: "partial version", body: "version \"1.2\"\n", noTarget: true},
		{name: "duplicate declaration", body: "version \"1.2.3\"\n  version \"1.2.3\"\n", bad: true},
		{name: "unterminated quote", body: "version \"1.2.3\n", bad: true},
		{name: "single quote unsupported", body: "version '1.2.3'\n", bad: true},
		{name: "trailing executable text", body: "version \"1.2.3\"; system(\"ignored\")\n", bad: true},
		{name: "malformed carriage return", body: "version \"1.2.3\"\rjunk\n", bad: true},
		{name: "incomplete declaration", body: "version\n", bad: true},
		{name: "invalid UTF8", body: "version \"1.2.3\"\n\xff", bad: true},
		{name: "formula oversized", body: "version \"1.2.3\"\n" + strings.Repeat("#", 64<<10), bad: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFormula([]byte(tt.body))
			assertCandidateResult(t, got, err, tt.want, tt.noTarget, tt.bad)
		})
	}
}

func TestSourceRequestsPrimaryChannelsAvoidGitHubAPI(t *testing.T) {
	tests := []struct {
		name   string
		new    func(*http.Client) Source
		host   string
		path   string
		accept string
		resp   func(io.ReadCloser) *http.Response
		body   string
		want   Candidate
	}{
		{
			name: "admin redirect", new: func(c *http.Client) Source { return NewGitHubSource(c, "admin") },
			host: "github.com", path: "/tokenlive/tokenlive-admin/releases/latest", accept: "text/html",
			resp: func(b io.ReadCloser) *http.Response {
				r := response(http.StatusFound, b)
				r.Header.Set("Location", "https://github.com/tokenlive/tokenlive-admin/releases/tag/v1.2.4")
				return r
			},
			want: Candidate{"v1.2.4", "https://github.com/tokenlive/tokenlive-admin/releases/tag/v1.2.4"},
		},
		{
			name: "gateway redirect", new: func(c *http.Client) Source { return NewGitHubSource(c, "gateway") },
			host: "github.com", path: "/tokenlive/tokenlive-gateway/releases/latest", accept: "text/html",
			resp: func(b io.ReadCloser) *http.Response {
				r := response(http.StatusFound, b)
				r.Header.Set("Location", "https://github.com/tokenlive/tokenlive-gateway/releases/tag/v1.2.5")
				return r
			},
			want: Candidate{"v1.2.5", "https://github.com/tokenlive/tokenlive-gateway/releases/tag/v1.2.5"},
		},
		{
			name: "homebrew raw formula stays older than standalone release", new: NewHomebrewSource,
			host: "raw.githubusercontent.com", path: "/tokenlive/homebrew-tokenlive/main/Formula/tokenlive.rb", accept: "text/plain",
			resp: func(b io.ReadCloser) *http.Response { return response(http.StatusOK, b) },
			body: "version \"1.2.3\"\n",
			want: Candidate{"v1.2.3", "https://github.com/tokenlive/tokenlive-standalone/releases/tag/v1.2.3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			body := &trackedBody{Reader: strings.NewReader(tt.body)}
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.Host == "api.github.com" {
					t.Fatalf("primary channel should not contact api.github.com: %s", req.URL)
				}
				assertAnonymousRequest(t, req, tt.host, tt.path, tt.accept)
				return tt.resp(body), nil
			})}
			got, err := tt.new(client).Latest(context.Background())
			assertCandidateResult(t, got, err, tt.want, false, false)
			if calls != 1 || !body.closed {
				t.Fatalf("requests=%d, body closed=%v; want one request with closed body", calls, body.closed)
			}
		})
	}
}

func TestSourceFallbackToGitHubAPIAndETagCaching(t *testing.T) {
	t.Run("admin falls back to api.github.com and reuses ETag", func(t *testing.T) {
		var requests []*http.Request
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requests = append(requests, req)
			if req.URL.Host == "github.com" {
				return response(http.StatusServiceUnavailable, io.NopCloser(strings.NewReader("unavailable"))), nil
			}
			assertAnonymousRequest(t, req, "api.github.com", "/repos/tokenlive/tokenlive-admin/releases/latest", "application/vnd.github+json")
			if req.Header.Get("If-None-Match") == `"etag-v124"` {
				return response(http.StatusNotModified, io.NopCloser(strings.NewReader(""))), nil
			}
			r := response(http.StatusOK, io.NopCloser(strings.NewReader(`{"tag_name":"v1.2.4","draft":false,"prerelease":false}`)))
			r.Header.Set("ETag", `"etag-v124"`)
			return r, nil
		})}
		src := NewGitHubSource(client, "admin")
		want := Candidate{"v1.2.4", "https://github.com/tokenlive/tokenlive-admin/releases/tag/v1.2.4"}
		got, err := src.Latest(context.Background())
		assertCandidateResult(t, got, err, want, false, false)
		got2, err := src.Latest(context.Background())
		assertCandidateResult(t, got2, err, want, false, false)
		if len(requests) != 4 || requests[3].Header.Get("If-None-Match") != `"etag-v124"` {
			t.Fatalf("expected ETag reuse on second API fallback, got %d requests", len(requests))
		}
	})

	t.Run("homebrew raw ETag 304 and API fallback with multiline base64", func(t *testing.T) {
		rawCalls := 0
		failRaw := false
		formula := "class Tokenlive < Formula\n  version \"1.2.3\"\nend\n" + strings.Repeat("# pad\n", 15)
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == "raw.githubusercontent.com" {
				rawCalls++
				if failRaw {
					return response(http.StatusNotFound, io.NopCloser(strings.NewReader("not found"))), nil
				}
				assertAnonymousRequest(t, req, "raw.githubusercontent.com", "/tokenlive/homebrew-tokenlive/main/Formula/tokenlive.rb", "text/plain")
				if req.Header.Get("If-None-Match") == `"raw-etag"` {
					return response(http.StatusNotModified, io.NopCloser(strings.NewReader(""))), nil
				}
				r := response(http.StatusOK, io.NopCloser(strings.NewReader(formula)))
				r.Header.Set("ETag", `"raw-etag"`)
				return r, nil
			}
			assertAnonymousRequest(t, req, "api.github.com", "/repos/tokenlive/homebrew-tokenlive/contents/Formula/tokenlive.rb", "application/vnd.github+json")
			return response(http.StatusOK, io.NopCloser(strings.NewReader(multilineFormulaContents(t, formula)))), nil
		})}
		src := NewHomebrewSource(client)
		want := Candidate{"v1.2.3", "https://github.com/tokenlive/tokenlive-standalone/releases/tag/v1.2.3"}
		got, err := src.Latest(context.Background())
		assertCandidateResult(t, got, err, want, false, false)
		got2, err := src.Latest(context.Background())
		assertCandidateResult(t, got2, err, want, false, false)
		if rawCalls != 2 {
			t.Fatalf("expected 2 raw calls with 304 cache hit, got %d", rawCalls)
		}
		failRaw = true
		got3, err := src.Latest(context.Background())
		assertCandidateResult(t, got3, err, want, false, false)
	})
}

func TestSourceSizeBoundaries(t *testing.T) {
	release := `{"tag_name":"v1.2.4","draft":false,"prerelease":false}`
	formula := "version \"1.2.3\"\n"
	tests := []struct {
		name string
		new  func(*http.Client) Source
		body string
		want Candidate
	}{
		{
			name: "exactly one MiB API release response", new: func(c *http.Client) Source { return NewGitHubSource(c, "admin") },
			body: release + strings.Repeat(" ", (1<<20)-len(release)),
			want: Candidate{"v1.2.4", "https://github.com/tokenlive/tokenlive-admin/releases/tag/v1.2.4"},
		},
		{
			name: "exactly 64 KiB raw Formula", new: NewHomebrewSource,
			body: formula + strings.Repeat("#", (64<<10)-len(formula)),
			want: Candidate{"v1.2.3", "https://github.com/tokenlive/tokenlive-standalone/releases/tag/v1.2.3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Host == "github.com" {
					return response(http.StatusBadGateway, io.NopCloser(strings.NewReader(""))), nil
				}
				return response(http.StatusOK, io.NopCloser(strings.NewReader(tt.body))), nil
			})}
			got, err := tt.new(client).Latest(context.Background())
			assertCandidateResult(t, got, err, tt.want, false, false)
		})
	}
}

func TestSourceLatestWithoutStableCandidate(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		assertAnonymousRequest(t, req, "github.com", "/tokenlive/tokenlive-gateway/releases/latest", "text/html")
		r := response(http.StatusFound, io.NopCloser(strings.NewReader("")))
		r.Header.Set("Location", "https://github.com/tokenlive/tokenlive-gateway/releases/tag/v1.3.0-rc.1")
		return r, nil
	})}
	got, err := NewGitHubSource(client, "gateway").Latest(context.Background())
	assertCandidateResult(t, got, err, Candidate{}, true, false)
	if calls != 1 {
		t.Fatalf("requests=%d; must not fall back to historical releases or API when primary returns ErrNoCandidate", calls)
	}
}

func TestSourceNilClientUsesAnonymousDefaultClone(t *testing.T) {
	previous := http.DefaultClient
	t.Cleanup(func() { http.DefaultClient = previous })
	http.DefaultClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		assertAnonymousRequest(t, req, "github.com", "/tokenlive/tokenlive-admin/releases/latest", "text/html")
		r := response(http.StatusFound, io.NopCloser(strings.NewReader("")))
		r.Header.Set("Location", "https://github.com/tokenlive/tokenlive-admin/releases/tag/v1.2.4")
		return r, nil
	})}
	got, err := NewGitHubSource(nil, "admin").Latest(context.Background())
	assertCandidateResult(t, got, err, Candidate{"v1.2.4", "https://github.com/tokenlive/tokenlive-admin/releases/tag/v1.2.4"}, false, false)
	if http.DefaultClient.CheckRedirect != nil {
		t.Fatal("default client's redirect policy was mutated")
	}
}

func TestSourceHTTPFailuresAndRateLimitClassification(t *testing.T) {
	for _, constructor := range sourceConstructors() {
		t.Run(constructor.name, func(t *testing.T) {
			for _, tc := range []struct {
				name        string
				status      int
				remaining   string
				rateLimited bool
			}{
				{"Forbidden", http.StatusForbidden, "", false},
				{"ForbiddenRateLimited", http.StatusForbidden, "0", true},
				{"TooManyRequests", http.StatusTooManyRequests, "", true},
				{"NotFound", http.StatusNotFound, "", false},
			} {
				t.Run(tc.name, func(t *testing.T) {
					var bodies []*trackedBody
					client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
						body := &trackedBody{Reader: strings.NewReader("upstream message must not be returned")}
						bodies = append(bodies, body)
						r := response(tc.status, body)
						if tc.remaining != "" {
							r.Header.Set("X-RateLimit-Remaining", tc.remaining)
						}
						return r, nil
					})}
					got, err := constructor.new(client).Latest(context.Background())
					assertCandidateResult(t, got, err, Candidate{}, false, true)
					if errors.Is(err, ErrRateLimited) != tc.rateLimited {
						t.Fatalf("rateLimited=%v, err=%v", tc.rateLimited, err)
					}
					for _, body := range bodies {
						if body.read != 0 || !body.closed || strings.Contains(err.Error(), "upstream message") {
							t.Fatalf("status handling read=%d closed=%v err=%v", body.read, body.closed, err)
						}
					}
				})
			}
			t.Run("oversized", func(t *testing.T) {
				body := &trackedBody{Reader: strings.NewReader(strings.Repeat(" ", (1<<20)+10))}
				client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					if req.URL.Host == "github.com" {
						return response(http.StatusBadGateway, io.NopCloser(strings.NewReader(""))), nil
					}
					return response(http.StatusOK, body), nil
				})}
				got, err := constructor.new(client).Latest(context.Background())
				assertCandidateResult(t, got, err, Candidate{}, false, true)
				if body.read > (1<<20)+1 || !body.closed {
					t.Fatalf("unbounded response read=%d closed=%v", body.read, body.closed)
				}
			})
			t.Run("read error", func(t *testing.T) {
				wantErr := errors.New("read failed")
				body := &trackedBody{Reader: errorReader{wantErr}}
				client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					if req.URL.Host == "github.com" || req.URL.Host == "raw.githubusercontent.com" {
						return response(http.StatusBadGateway, io.NopCloser(strings.NewReader(""))), nil
					}
					return response(http.StatusOK, body), nil
				})}
				_, err := constructor.new(client).Latest(context.Background())
				if !errors.Is(err, wantErr) || !body.closed {
					t.Fatalf("error=%v closed=%v", err, body.closed)
				}
			})
			t.Run("context cancellation", func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				calls := 0
				client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					calls++
					cancel()
					select {
					case <-req.Context().Done():
						return nil, req.Context().Err()
					case <-time.After(time.Second):
						return nil, errors.New("request did not inherit cancellation")
					}
				})}
				_, err := constructor.new(client).Latest(ctx)
				if !errors.Is(err, context.Canceled) || calls != 1 {
					t.Fatalf("got %v (calls=%d); want context.Canceled with 1 call", err, calls)
				}
			})
			t.Run("parent deadline", func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				wantDeadline, _ := ctx.Deadline()
				client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					if got, ok := req.Context().Deadline(); !ok || !got.Equal(wantDeadline) {
						t.Fatalf("request deadline=%v exists=%v; want %v", got, ok, wantDeadline)
					}
					return nil, context.DeadlineExceeded
				})}
				_, err := constructor.new(client).Latest(ctx)
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("got %v; want context.DeadlineExceeded", err)
				}
			})
		})
	}
}

func TestSourceFormulaContentsFallback(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		noTarget bool
	}{
		{name: "wrong type", body: `{"type":"dir","encoding":"base64","content":"dmVyc2lvbiAiMS4yLjMi"}`},
		{name: "wrong encoding", body: `{"type":"file","encoding":"none","content":"version \"1.2.3\""}`},
		{name: "missing content", body: `{"type":"file","encoding":"base64"}`},
		{name: "invalid base64", body: `{"type":"file","encoding":"base64","content":"!!!"}`},
		{name: "invalid JSON", body: `{`},
		{name: "null JSON", body: `null`},
		{name: "decoded limit", body: formulaContents(t, "version \"1.2.3\"\n"+strings.Repeat("#", 64<<10))},
		{name: "no stable candidate", body: formulaContents(t, "version \"1.3.0-rc.1\"\n"), noTarget: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Host == "raw.githubusercontent.com" {
					return response(http.StatusServiceUnavailable, io.NopCloser(strings.NewReader(""))), nil
				}
				return response(http.StatusOK, io.NopCloser(strings.NewReader(tt.body))), nil
			})}
			got, err := NewHomebrewSource(client).Latest(context.Background())
			assertCandidateResult(t, got, err, Candidate{}, tt.noTarget, !tt.noTarget)
		})
	}
}

func TestSourceRejectsUnknownComponentWithoutRequest(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatal("unknown component must not make a request")
		return nil, errors.New("unexpected request")
	})}
	got, err := NewGitHubSource(client, "https://attacker.invalid").Latest(context.Background())
	assertCandidateResult(t, got, err, Candidate{}, false, true)
}

func TestSourceRedirectPolicy(t *testing.T) {
	for _, target := range []string{
		"http://api.github.com/redirected",
		"https://attacker.invalid/redirected",
		"https://api.github.com.attacker.invalid/redirected",
		"https://api.github.com:8443/redirected",
		"https://user:password@api.github.com/redirected",
		"https://github.com/redirected",
	} {
		t.Run(target, func(t *testing.T) {
			apiCalls := 0
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Host == "github.com" {
					return response(http.StatusBadGateway, io.NopCloser(strings.NewReader(""))), nil
				}
				apiCalls++
				return redirectResponse(target), nil
			})}
			got, err := NewGitHubSource(client, "admin").Latest(context.Background())
			assertCandidateResult(t, got, err, Candidate{}, false, true)
			if apiCalls != 1 {
				t.Fatalf("followed forbidden redirect: api requests=%d", apiCalls)
			}
		})
	}
	t.Run("allowed host", func(t *testing.T) {
		apiCalls := 0
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == "github.com" {
				return response(http.StatusBadGateway, io.NopCloser(strings.NewReader(""))), nil
			}
			apiCalls++
			if apiCalls == 1 {
				return redirectResponse("https://api.github.com/redirected"), nil
			}
			assertAnonymousRequest(t, req, "api.github.com", "/redirected", "application/vnd.github+json")
			return response(http.StatusOK, io.NopCloser(strings.NewReader(`{"tag_name":"v1.2.4","draft":false,"prerelease":false}`))), nil
		})}
		got, err := NewGitHubSource(client, "admin").Latest(context.Background())
		assertCandidateResult(t, got, err, Candidate{"v1.2.4", "https://github.com/tokenlive/tokenlive-admin/releases/tag/v1.2.4"}, false, false)
		if apiCalls != 2 {
			t.Fatalf("api requests=%d; want 2", apiCalls)
		}
	})
	t.Run("bounded redirect loop", func(t *testing.T) {
		apiCalls := 0
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == "github.com" {
				return response(http.StatusBadGateway, io.NopCloser(strings.NewReader(""))), nil
			}
			apiCalls++
			if apiCalls > 12 {
				return nil, errors.New("unbounded redirects")
			}
			return redirectResponse("https://api.github.com/loop"), nil
		})}
		got, err := NewGitHubSource(client, "admin").Latest(context.Background())
		assertCandidateResult(t, got, err, Candidate{}, false, true)
		if apiCalls > 10 {
			t.Fatalf("redirect loop made %d api requests; want at most 10", apiCalls)
		}
	})
}

func TestSourceDoesNotMutateBorrowedClientOrSendCookies(t *testing.T) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	apiURL, _ := url.Parse("https://api.github.com")
	ghURL, _ := url.Parse("https://github.com")
	jar.SetCookies(apiURL, []*http.Cookie{{Name: "business-session", Value: "private"}})
	jar.SetCookies(ghURL, []*http.Cookie{{Name: "business-session", Value: "private"}})
	callerRedirect := errors.New("caller redirect policy")
	originalRedirect := func(req *http.Request, via []*http.Request) error { return callerRedirect }
	apiCalls := 0
	client := &http.Client{
		Jar: jar, Timeout: time.Second, CheckRedirect: originalRedirect,
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == "github.com" {
				assertAnonymousRequest(t, req, "github.com", "/tokenlive/tokenlive-admin/releases/latest", "text/html")
				return response(http.StatusBadGateway, io.NopCloser(strings.NewReader(""))), nil
			}
			apiCalls++
			assertAnonymousRequest(t, req, "api.github.com", req.URL.Path, "application/vnd.github+json")
			if apiCalls == 1 {
				return redirectResponse("https://api.github.com/redirected"), nil
			}
			return response(http.StatusOK, io.NopCloser(strings.NewReader(`{"tag_name":"v1.2.4","draft":false,"prerelease":false}`))), nil
		}),
	}
	got, err := NewGitHubSource(client, "admin").Latest(context.Background())
	assertCandidateResult(t, got, err, Candidate{"v1.2.4", "https://github.com/tokenlive/tokenlive-admin/releases/tag/v1.2.4"}, false, false)
	if client.Jar != jar || client.Timeout != time.Second || !errors.Is(client.CheckRedirect(nil, nil), callerRedirect) {
		t.Fatal("borrowed client's settings were mutated")
	}
	if cookies := jar.Cookies(apiURL); len(cookies) != 1 || cookies[0].Value != "private" {
		t.Fatalf("borrowed cookie jar mutated: %v", cookies)
	}
}

func assertCandidateResult(t *testing.T, got Candidate, err error, want Candidate, noTarget, bad bool) {
	t.Helper()
	if noTarget {
		if !errors.Is(err, ErrNoCandidate) || got != (Candidate{}) {
			t.Fatalf("got %+v, %v; want empty candidate and ErrNoCandidate", got, err)
		}
		return
	}
	if bad {
		if err == nil || errors.Is(err, ErrNoCandidate) || got != (Candidate{}) {
			t.Fatalf("got %+v, %v; want empty candidate and check error", got, err)
		}
		return
	}
	if err != nil || got != want {
		t.Fatalf("got %+v, %v; want %+v, nil", got, err, want)
	}
}

func assertAnonymousRequest(t *testing.T, req *http.Request, host, path, accept string) {
	t.Helper()
	if req.Method != http.MethodGet || req.URL.Scheme != "https" || req.URL.Host != host || req.URL.Path != path || req.URL.RawQuery != "" || req.URL.User != nil {
		t.Fatalf("unexpected source request: %s %s", req.Method, req.URL)
	}
	if req.Body != nil || req.ContentLength != 0 {
		t.Fatal("source request must not contain a body")
	}
	if req.Header.Get("Accept") != accept || req.Header.Get("User-Agent") != "tokenlive-admin-updatecheck" {
		t.Fatalf("unexpected source headers: %v", req.Header)
	}
	for name := range req.Header {
		if name != "Accept" && name != "User-Agent" && name != "If-None-Match" {
			t.Fatalf("unexpected business or credential header: %s", name)
		}
	}
}

func formulaContents(t *testing.T, formula string) string {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"type": "file", "encoding": "base64", "name": "tokenlive.rb",
		"path": "Formula/tokenlive.rb", "content": base64.StdEncoding.EncodeToString([]byte(formula)) + "\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func multilineFormulaContents(t *testing.T, formula string) string {
	t.Helper()
	raw := base64.StdEncoding.EncodeToString([]byte(formula))
	var lines []string
	for i := 0; i < len(raw); i += 60 {
		end := i + 60
		if end > len(raw) {
			end = len(raw)
		}
		lines = append(lines, raw[i:end])
	}
	body, err := json.Marshal(map[string]any{
		"type": "file", "encoding": "base64", "name": "tokenlive.rb",
		"path": "Formula/tokenlive.rb", "content": strings.Join(lines, "\n") + "\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func sourceConstructors() []struct {
	name string
	new  func(*http.Client) Source
} {
	return []struct {
		name string
		new  func(*http.Client) Source
	}{
		{"release", func(c *http.Client) Source { return NewGitHubSource(c, "admin") }},
		{"formula", NewHomebrewSource},
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func response(status int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: body}
}

func redirectResponse(target string) *http.Response {
	resp := response(http.StatusFound, io.NopCloser(strings.NewReader("")))
	resp.Header.Set("Location", target)
	return resp
}

type trackedBody struct {
	io.Reader
	read   int
	closed bool
}

func (b *trackedBody) Read(p []byte) (int, error) {
	n, err := b.Reader.Read(p)
	b.read += n
	return n, err
}

func (b *trackedBody) Close() error {
	b.closed = true
	return nil
}

type errorReader struct{ err error }

func (r errorReader) Read(p []byte) (int, error) { return 0, r.err }
