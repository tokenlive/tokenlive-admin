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

func TestSourceRequests(t *testing.T) {
	tests := []struct {
		name string
		new  func(*http.Client) Source
		path string
		body string
		want Candidate
	}{
		{
			name: "admin", new: func(c *http.Client) Source { return NewGitHubSource(c, "admin") },
			path: "/repos/tokenlive/tokenlive-admin/releases/latest",
			body: `{"tag_name":"v1.2.4","draft":false,"prerelease":false}`,
			want: Candidate{"v1.2.4", "https://github.com/tokenlive/tokenlive-admin/releases/tag/v1.2.4"},
		},
		{
			name: "gateway", new: func(c *http.Client) Source { return NewGitHubSource(c, "gateway") },
			path: "/repos/tokenlive/tokenlive-gateway/releases/latest",
			body: `{"tag_name":"v1.2.5","draft":false,"prerelease":false}`,
			want: Candidate{"v1.2.5", "https://github.com/tokenlive/tokenlive-gateway/releases/tag/v1.2.5"},
		},
		{
			name: "formula stays older than standalone release", new: NewHomebrewSource,
			path: "/repos/tokenlive/homebrew-tokenlive/contents/Formula/tokenlive.rb",
			body: formulaContents(t, "version \"1.2.3\"\n"),
			want: Candidate{"v1.2.3", "https://github.com/tokenlive/tokenlive-standalone/releases/tag/v1.2.3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			body := &trackedBody{Reader: strings.NewReader(tt.body)}
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.Path == "/repos/tokenlive/tokenlive-standalone/releases/latest" {
					return response(http.StatusOK, io.NopCloser(strings.NewReader(`{"tag_name":"v9.9.9","draft":false,"prerelease":false}`))), nil
				}
				assertAnonymousRequest(t, req, tt.path)
				return response(http.StatusOK, body), nil
			})}
			got, err := tt.new(client).Latest(context.Background())
			assertCandidateResult(t, got, err, tt.want, false, false)
			if calls != 1 || !body.closed {
				t.Fatalf("requests=%d, body closed=%v; want one request with closed body", calls, body.closed)
			}
		})
	}
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
			name: "exactly one MiB response", new: func(c *http.Client) Source { return NewGitHubSource(c, "admin") },
			body: release + strings.Repeat(" ", (1<<20)-len(release)),
			want: Candidate{"v1.2.4", "https://github.com/tokenlive/tokenlive-admin/releases/tag/v1.2.4"},
		},
		{
			name: "exactly 64 KiB decoded Formula", new: NewHomebrewSource,
			body: formulaContents(t, formula+strings.Repeat("#", (64<<10)-len(formula))),
			want: Candidate{"v1.2.3", "https://github.com/tokenlive/tokenlive-standalone/releases/tag/v1.2.3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
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
		assertAnonymousRequest(t, req, "/repos/tokenlive/tokenlive-gateway/releases/latest")
		return response(http.StatusOK, io.NopCloser(strings.NewReader(`{"tag_name":"v1.3.0-rc.1","draft":false,"prerelease":false}`))), nil
	})}
	got, err := NewGitHubSource(client, "gateway").Latest(context.Background())
	assertCandidateResult(t, got, err, Candidate{}, true, false)
	if calls != 1 {
		t.Fatalf("requests=%d; must not fall back to historical releases", calls)
	}
}

func TestSourceNilClientUsesAnonymousDefaultClone(t *testing.T) {
	previous := http.DefaultClient
	t.Cleanup(func() { http.DefaultClient = previous })
	http.DefaultClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		assertAnonymousRequest(t, req, "/repos/tokenlive/tokenlive-admin/releases/latest")
		return response(http.StatusOK, io.NopCloser(strings.NewReader(`{"tag_name":"v1.2.4","draft":false,"prerelease":false}`))), nil
	})}
	got, err := NewGitHubSource(nil, "admin").Latest(context.Background())
	assertCandidateResult(t, got, err, Candidate{"v1.2.4", "https://github.com/tokenlive/tokenlive-admin/releases/tag/v1.2.4"}, false, false)
	if http.DefaultClient.CheckRedirect != nil {
		t.Fatal("default client's redirect policy was mutated")
	}
}

func TestSourceHTTPFailures(t *testing.T) {
	for _, constructor := range sourceConstructors() {
		t.Run(constructor.name, func(t *testing.T) {
			for _, status := range []int{http.StatusForbidden, http.StatusTooManyRequests, http.StatusNotFound} {
				t.Run(http.StatusText(status), func(t *testing.T) {
					body := &trackedBody{Reader: strings.NewReader("upstream message must not be returned")}
					client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
						return response(status, body), nil
					})}
					got, err := constructor.new(client).Latest(context.Background())
					assertCandidateResult(t, got, err, Candidate{}, false, true)
					if body.read != 0 || !body.closed || strings.Contains(err.Error(), "upstream message") {
						t.Fatalf("status handling read=%d closed=%v err=%v", body.read, body.closed, err)
					}
				})
			}
			t.Run("oversized", func(t *testing.T) {
				body := &trackedBody{Reader: strings.NewReader(strings.Repeat(" ", (1<<20)+10))}
				client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
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
				client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					cancel()
					select {
					case <-req.Context().Done():
						return nil, req.Context().Err()
					case <-time.After(time.Second):
						return nil, errors.New("request did not inherit cancellation")
					}
				})}
				_, err := constructor.new(client).Latest(ctx)
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("got %v; want context.Canceled", err)
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

func TestSourceFormulaContents(t *testing.T) {
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
			calls := 0
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				return redirectResponse(target), nil
			})}
			got, err := NewGitHubSource(client, "admin").Latest(context.Background())
			assertCandidateResult(t, got, err, Candidate{}, false, true)
			if calls != 1 {
				t.Fatalf("followed forbidden redirect: requests=%d", calls)
			}
		})
	}
	t.Run("allowed host", func(t *testing.T) {
		calls := 0
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return redirectResponse("https://api.github.com/redirected"), nil
			}
			assertAnonymousRequest(t, req, "/redirected")
			return response(http.StatusOK, io.NopCloser(strings.NewReader(`{"tag_name":"v1.2.4","draft":false,"prerelease":false}`))), nil
		})}
		got, err := NewGitHubSource(client, "admin").Latest(context.Background())
		assertCandidateResult(t, got, err, Candidate{"v1.2.4", "https://github.com/tokenlive/tokenlive-admin/releases/tag/v1.2.4"}, false, false)
		if calls != 2 {
			t.Fatalf("requests=%d; want 2", calls)
		}
	})
	t.Run("bounded redirect loop", func(t *testing.T) {
		calls := 0
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls > 12 {
				return nil, errors.New("unbounded redirects")
			}
			return redirectResponse("https://api.github.com/loop"), nil
		})}
		got, err := NewGitHubSource(client, "admin").Latest(context.Background())
		assertCandidateResult(t, got, err, Candidate{}, false, true)
		if calls > 10 {
			t.Fatalf("redirect loop made %d requests; want at most 10", calls)
		}
	})
}

func TestSourceDoesNotMutateBorrowedClientOrSendCookies(t *testing.T) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	apiURL, _ := url.Parse("https://api.github.com")
	jar.SetCookies(apiURL, []*http.Cookie{{Name: "business-session", Value: "private"}})
	callerRedirect := errors.New("caller redirect policy")
	originalRedirect := func(req *http.Request, via []*http.Request) error { return callerRedirect }
	calls := 0
	client := &http.Client{
		Jar: jar, Timeout: time.Second, CheckRedirect: originalRedirect,
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			assertAnonymousRequest(t, req, req.URL.Path)
			if calls == 1 {
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

func assertAnonymousRequest(t *testing.T, req *http.Request, path string) {
	t.Helper()
	if req.Method != http.MethodGet || req.URL.Scheme != "https" || req.URL.Host != "api.github.com" || req.URL.Path != path || req.URL.RawQuery != "" || req.URL.User != nil {
		t.Fatalf("unexpected source request: %s %s", req.Method, req.URL)
	}
	if req.Body != nil || req.ContentLength != 0 {
		t.Fatal("source request must not contain a body")
	}
	if req.Header.Get("Accept") != "application/vnd.github+json" || req.Header.Get("User-Agent") != "tokenlive-admin-updatecheck" {
		t.Fatalf("unexpected source headers: %v", req.Header)
	}
	for name := range req.Header {
		if name != "Accept" && name != "User-Agent" {
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
