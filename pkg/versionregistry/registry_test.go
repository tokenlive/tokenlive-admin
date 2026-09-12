package versionregistry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func fixtureNode(t *testing.T) Node {
	t.Helper()
	payload, err := os.ReadFile("testdata/node-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var node Node
	if err := json.Unmarshal(payload, &node); err != nil {
		t.Fatal(err)
	}
	return node
}

func TestMemoryExpiryAndDedup(t *testing.T) {
	now := time.Unix(100, 0)
	store := NewMemoryStore("default", func() time.Time { return now })
	node := fixtureNode(t)
	ctx := context.Background()
	if err := store.Upsert(ctx, node); err != nil {
		t.Fatal(err)
	}
	if err := store.Upsert(ctx, node); err != nil {
		t.Fatal(err)
	}
	nodes, err := store.List(ctx)
	if err != nil || !reflect.DeepEqual(nodes, []Node{node}) {
		t.Fatalf("List() = %v, %v; want one fixture node", nodes, err)
	}
	now = now.Add(3 * time.Minute)
	nodes, err = store.List(ctx)
	if err != nil || len(nodes) != 0 {
		t.Fatalf("List() at expiry = %v, %v; want no nodes", nodes, err)
	}
}

func TestValidate(t *testing.T) {
	base := fixtureNode(t)
	tests := []struct {
		name      string
		change    func(*Node)
		namespace string
		wantError string
	}{
		{name: "fixture", namespace: "default"},
		{name: "default namespace", namespace: ""},
		{name: "unknown version", change: func(n *Node) { n.Version = "" }, namespace: "default"},
		{name: "uncomparable version", change: func(n *Node) { n.Version = "local-vendor" }, namespace: "default"},
		{name: "development", change: func(n *Node) { n.BuildKind = "dev" }, namespace: "default"},
		{name: "version byte limit", change: func(n *Node) { n.Version = strings.Repeat("x", 128) }, namespace: "default"},
		{name: "schema", change: func(n *Node) { n.SchemaVersion = 2 }, namespace: "default", wantError: "schema_version"},
		{name: "missing namespace", change: func(n *Node) { n.Namespace = "" }, namespace: "default", wantError: "namespace"},
		{name: "other namespace", change: func(n *Node) { n.Namespace = "other" }, namespace: "default", wantError: "namespace"},
		{name: "namespace wildcard", change: func(n *Node) { n.Namespace = "*" }, namespace: "*", wantError: "namespace"},
		{name: "namespace colon", change: func(n *Node) { n.Namespace = "a:b" }, namespace: "a:b", wantError: "namespace"},
		{name: "namespace whitespace", change: func(n *Node) { n.Namespace = " a " }, namespace: " a ", wantError: "namespace"},
		{name: "namespace too long", namespace: strings.Repeat("a", 65), wantError: "namespace"},
		{name: "namespace maximum", change: func(n *Node) { n.Namespace = strings.Repeat("a", 64) }, namespace: strings.Repeat("a", 64)},
		{name: "namespace alphabet", change: func(n *Node) { n.Namespace = "A-Z_a-9" }, namespace: "A-Z_a-9"},
		{name: "missing id", change: func(n *Node) { n.InstanceID = "" }, namespace: "default", wantError: "instance_id"},
		{name: "id wildcard", change: func(n *Node) { n.InstanceID = "*" }, namespace: "default", wantError: "instance_id"},
		{name: "id compact", change: func(n *Node) { n.InstanceID = strings.ReplaceAll(n.InstanceID, "-", "") }, namespace: "default", wantError: "instance_id"},
		{name: "id malformed", change: func(n *Node) { n.InstanceID = "00000000-0000-4000-8000-00000000000x" }, namespace: "default", wantError: "instance_id"},
		{name: "version too long", change: func(n *Node) { n.Version = strings.Repeat("x", 129) }, namespace: "default", wantError: "version"},
		{name: "version bytes not runes", change: func(n *Node) { n.Version = strings.Repeat("界", 43) }, namespace: "default", wantError: "version"},
		{name: "missing kind", change: func(n *Node) { n.BuildKind = "" }, namespace: "default", wantError: "build_kind"},
		{name: "unknown kind", change: func(n *Node) { n.BuildKind = "nightly" }, namespace: "default", wantError: "build_kind"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			node := base
			if test.change != nil {
				test.change(&node)
			}
			err := Validate(node, test.namespace)
			if test.wantError == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("Validate() = %v; want %q error", err, test.wantError)
			}
		})
	}
}

func TestMemoryRefreshAndDelete(t *testing.T) {
	now := time.Unix(100, 0)
	store := NewMemoryStore("default", func() time.Time { return now })
	ctx := context.Background()
	node := fixtureNode(t)
	if err := store.Upsert(ctx, node); err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Minute)
	node.Version = "v2.0.0"
	if err := store.Upsert(ctx, node); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	nodes, err := store.List(ctx)
	if err != nil || !reflect.DeepEqual(nodes, []Node{node}) {
		t.Fatalf("refreshed List() = %v, %v", nodes, err)
	}
	second := node
	second.InstanceID = "00000000-0000-4000-8000-000000000002"
	if err := store.Upsert(ctx, second); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, node.InstanceID); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, node.InstanceID); err != nil {
		t.Fatalf("repeated delete: %v", err)
	}
	nodes, err = store.List(ctx)
	if err != nil || !reflect.DeepEqual(nodes, []Node{second}) {
		t.Fatalf("List() after exact delete = %v, %v", nodes, err)
	}
	now = now.Add(3 * time.Minute)
	nodes, err = store.List(ctx)
	if err != nil || len(nodes) != 0 {
		t.Fatalf("List() after refreshed expiry = %v, %v", nodes, err)
	}
}

func TestMemoryRejectsInvalidRecordsAndDelete(t *testing.T) {
	store := NewMemoryStore("default", time.Now)
	node := fixtureNode(t)
	ctx := context.Background()
	node.Namespace = "other"
	if err := store.Upsert(ctx, node); err == nil {
		t.Fatal("cross-namespace upsert succeeded")
	}
	if err := store.Delete(ctx, "*"); err == nil {
		t.Fatal("invalid UUID delete succeeded")
	}
	nodes, err := store.List(ctx)
	if err != nil || len(nodes) != 0 {
		t.Fatalf("rejected node was stored: %v, %v", nodes, err)
	}
}

func TestMemoryDefaultsAndInvalidNamespace(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore("", nil)
	if err := store.Upsert(ctx, fixtureNode(t)); err != nil {
		t.Fatal(err)
	}
	nodes, err := store.List(ctx)
	if err != nil || len(nodes) != 1 {
		t.Fatalf("default store List() = %v, %v", nodes, err)
	}
	for _, namespace := range []string{"*", "a:b", strings.Repeat("x", 65)} {
		store := NewMemoryStore(namespace, nil)
		if err := store.Upsert(ctx, fixtureNode(t)); err == nil {
			t.Errorf("invalid namespace %q upsert succeeded", namespace)
		}
		if nodes, err := store.List(ctx); err == nil || nodes != nil {
			t.Errorf("invalid namespace %q List() = %v, %v", namespace, nodes, err)
		}
		if err := store.Delete(ctx, fixtureNode(t).InstanceID); err == nil {
			t.Errorf("invalid namespace %q delete succeeded", namespace)
		}
	}
}

func TestMemoryCanceledContextDoesNotMutate(t *testing.T) {
	store := NewMemoryStore("default", time.Now)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	node := fixtureNode(t)
	if err := store.Upsert(ctx, node); !errors.Is(err, context.Canceled) {
		t.Fatalf("Upsert() = %v; want context.Canceled", err)
	}
	if nodes, err := store.List(ctx); !errors.Is(err, context.Canceled) || nodes != nil {
		t.Fatalf("List() = %v, %v; want context.Canceled", nodes, err)
	}
	if err := store.Upsert(context.Background(), node); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, node.InstanceID); !errors.Is(err, context.Canceled) {
		t.Fatalf("Delete() = %v; want context.Canceled", err)
	}
	nodes, err := store.List(context.Background())
	if err != nil || !reflect.DeepEqual(nodes, []Node{node}) {
		t.Fatalf("canceled Delete mutated records: %v, %v", nodes, err)
	}
}

func TestMemoryConcurrentOperations(t *testing.T) {
	store := NewMemoryStore("default", time.Now)
	base := fixtureNode(t)
	var wg sync.WaitGroup
	for i := 1; i <= 32; i++ {
		wg.Go(func() {
			node := base
			node.InstanceID = fmt.Sprintf("00000000-0000-4000-8000-%012d", i)
			ctx := context.Background()
			if err := store.Upsert(ctx, node); err != nil {
				t.Error(err)
			}
			if _, err := store.List(ctx); err != nil {
				t.Error(err)
			}
			if err := store.Delete(ctx, node.InstanceID); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if nodes, err := store.List(context.Background()); err != nil || len(nodes) != 0 {
		t.Fatalf("List() after concurrent deletes = %v, %v", nodes, err)
	}
}

func TestMemoryCanonicalUUIDIdentity(t *testing.T) {
	store := NewMemoryStore("default", time.Now)
	ctx := context.Background()
	node := fixtureNode(t)
	node.InstanceID = "abcdef00-1234-4000-8000-abcdef000001"
	if err := store.Upsert(ctx, node); err != nil {
		t.Fatal(err)
	}
	upper := node
	upper.InstanceID = strings.ToUpper(node.InstanceID)
	upper.Version = "v2.0.0"
	if err := store.Upsert(ctx, upper); err != nil {
		t.Fatal(err)
	}
	node.Version = "v2.0.0"
	nodes, err := store.List(ctx)
	if err != nil || !reflect.DeepEqual(nodes, []Node{node}) {
		t.Fatalf("UUID case variants did not deduplicate: %v, %v", nodes, err)
	}
	if err := store.Delete(ctx, upper.InstanceID); err != nil {
		t.Fatal(err)
	}
	if nodes, err := store.List(ctx); err != nil || len(nodes) != 0 {
		t.Fatalf("canonical Delete() left nodes: %v, %v", nodes, err)
	}
}

func TestGroupNodes(t *testing.T) {
	nodes := []Node{
		{Version: "1.2.3", BuildKind: "release"},
		{Version: "v1.2.3+build.7", BuildKind: "release"},
		{Version: "v1.2.3", BuildKind: "dev"},
		{Version: "", BuildKind: "dev"},
		{Version: "v1.2.3-rc.1", BuildKind: "release"},
		{Version: "1.2", BuildKind: "release"},
		{Version: "01.2.3", BuildKind: "release"},
		{Version: "local-vendor", BuildKind: "dev"},
		{Version: "v1.2.3", BuildKind: "release"},
	}
	want := []Group{
		{Version: "01.2.3", BuildKind: "release", Count: 1},
		{Version: "1.2", BuildKind: "release", Count: 1},
		{Version: "local-vendor", BuildKind: "dev", Count: 1},
		{Version: "unknown", BuildKind: "dev", Count: 1},
		{Version: "v1.2.3", BuildKind: "dev", Count: 1},
		{Version: "v1.2.3", BuildKind: "release", Count: 3},
		{Version: "v1.2.3-rc.1", BuildKind: "release", Count: 1},
	}
	if got := GroupNodes(nodes); !reflect.DeepEqual(got, want) {
		t.Fatalf("GroupNodes() = %#v; want %#v", got, want)
	}
	for i, j := 0, len(nodes)-1; i < j; i, j = i+1, j-1 {
		nodes[i], nodes[j] = nodes[j], nodes[i]
	}
	if got := GroupNodes(nodes); !reflect.DeepEqual(got, want) {
		t.Fatalf("groups depend on input order: %#v", got)
	}
	groups := GroupNodes([]Node{fixtureNode(t)})
	payload, err := json.Marshal(groups)
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != `[{"version":"v1.2.3","build_kind":"release","count":1}]` {
		t.Fatalf("public group leaks identity or changes wire fields: %s", payload)
	}
	if got := GroupNodes(nil); got == nil || len(got) != 0 {
		t.Fatalf("empty groups = %#v; want an empty JSON array", got)
	}
}

func redisTestStore(t *testing.T, namespace string) (*miniredis.Miniredis, *redis.Client, Store) {
	t.Helper()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr:       server.Addr(),
		MaxRetries: -1,
	})
	t.Cleanup(func() { _ = client.Close() })
	return server, client, NewRedisStore(client, namespace)
}

func TestRedisExpiryRefreshDedupAndExactDelete(t *testing.T) {
	server, client, store := redisTestStore(t, "")
	ctx := context.Background()
	node := fixtureNode(t)
	node.InstanceID = "abcdef00-1234-4000-8000-abcdef000001"
	const key = "tokenlive:gateway-versions:default:abcdef00-1234-4000-8000-abcdef000001"
	for i := 0; i < 2; i++ {
		if err := store.Upsert(ctx, node); err != nil {
			t.Fatal(err)
		}
	}
	if keys := server.Keys(); !reflect.DeepEqual(keys, []string{key}) {
		t.Fatalf("Upsert keys = %v; want fixed scoped key", keys)
	}
	if ttl := server.TTL(key); ttl != 3*time.Minute {
		t.Fatalf("report TTL = %v; want 3m", ttl)
	}
	server.FastForward(2 * time.Minute)
	upper := node
	upper.InstanceID = strings.ToUpper(node.InstanceID)
	upper.Version = ""
	if err := store.Upsert(ctx, upper); err != nil {
		t.Fatal(err)
	}
	server.FastForward(time.Minute)
	node.Version = ""
	nodes, err := store.List(ctx)
	if err != nil || !reflect.DeepEqual(nodes, []Node{node}) {
		t.Fatalf("List after refresh = %v, %v; want canonical unknown-version node", nodes, err)
	}
	other := node
	other.Namespace = "other"
	otherStore := NewRedisStore(client, "other")
	if err := otherStore.Upsert(ctx, other); err != nil {
		t.Fatal(err)
	}
	sibling := node
	sibling.InstanceID = "abcdef00-1234-4000-8000-abcdef000002"
	if err := store.Upsert(ctx, sibling); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, upper.InstanceID); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, upper.InstanceID); err != nil {
		t.Fatalf("repeated Delete: %v", err)
	}
	nodes, err = store.List(ctx)
	if err != nil || !reflect.DeepEqual(nodes, []Node{sibling}) {
		t.Fatalf("exact Delete removed sibling: %v, %v", nodes, err)
	}
	nodes, err = otherStore.List(ctx)
	if err != nil || !reflect.DeepEqual(nodes, []Node{other}) {
		t.Fatalf("exact Delete touched other namespace: %v, %v", nodes, err)
	}
	server.FastForward(TTL)
	for _, scoped := range []Store{store, otherStore} {
		if nodes, err := scoped.List(ctx); err != nil || len(nodes) != 0 {
			t.Fatalf("expired reports remain: %v, %v", nodes, err)
		}
	}
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("borrowed Redis client was closed: %v", err)
	}
}

func TestRedisInvalidInputNeverIssuesCommands(t *testing.T) {
	server, client, store := redisTestStore(t, "default")
	ctx := context.Background()
	node := fixtureNode(t)
	if err := store.Upsert(ctx, node); err != nil {
		t.Fatal(err)
	}
	before := server.CommandCount()
	for _, namespace := range []string{"*", "default:*", "[a-z]", strings.Repeat("x", 65)} {
		bad := NewRedisStore(client, namespace)
		if err := bad.Upsert(ctx, node); err == nil {
			t.Errorf("invalid namespace %q upsert succeeded", namespace)
		}
		if nodes, err := bad.List(ctx); err == nil || nodes != nil {
			t.Errorf("invalid namespace %q List = %v, %v", namespace, nodes, err)
		}
		if err := bad.Delete(ctx, node.InstanceID); err == nil {
			t.Errorf("invalid namespace %q delete succeeded", namespace)
		}
	}
	for _, id := range []string{"*", "../other", node.InstanceID + ":*", "00000000000040008000000000000001"} {
		if err := store.Delete(ctx, id); err == nil {
			t.Errorf("invalid UUID %q Delete succeeded", id)
		}
		invalid := node
		invalid.InstanceID = id
		if err := store.Upsert(ctx, invalid); err == nil {
			t.Errorf("invalid UUID %q Upsert succeeded", id)
		}
	}
	node.Namespace = "other"
	if err := store.Upsert(ctx, node); err == nil {
		t.Error("cross-namespace Upsert succeeded")
	}
	if got := server.CommandCount(); got != before {
		t.Fatalf("invalid requests reached Redis: command count %d -> %d", before, got)
	}
	node.Namespace = "default"
	nodes, err := store.List(ctx)
	if err != nil || !reflect.DeepEqual(nodes, []Node{node}) {
		t.Fatalf("invalid request mutated valid record: %v, %v", nodes, err)
	}
}

func TestRedisNilClientFailsClosed(t *testing.T) {
	store := NewRedisStore(nil, "")
	ctx := context.Background()
	node := fixtureNode(t)
	if err := store.Upsert(ctx, node); err == nil {
		t.Fatal("nil client Upsert succeeded")
	}
	if nodes, err := store.List(ctx); err == nil || nodes != nil {
		t.Fatalf("nil client List = %v, %v", nodes, err)
	}
	if err := store.Delete(ctx, node.InstanceID); err == nil {
		t.Fatal("nil client Delete succeeded")
	}
}

func TestRedisFailuresAreNotEmptyRegistry(t *testing.T) {
	server, _, store := redisTestStore(t, "default")
	ctx := context.Background()
	node := fixtureNode(t)
	server.SetError("ERR test Redis failure")
	if err := store.Upsert(ctx, node); err == nil || !strings.Contains(err.Error(), "test Redis failure") {
		t.Fatalf("Upsert failure lost: %v", err)
	}
	if nodes, err := store.List(ctx); err == nil || nodes != nil || !strings.Contains(err.Error(), "test Redis failure") {
		t.Fatalf("Redis failure looks like zero nodes: %v, %v", nodes, err)
	}
	if err := store.Delete(ctx, node.InstanceID); err == nil || !strings.Contains(err.Error(), "test Redis failure") {
		t.Fatalf("Delete failure lost: %v", err)
	}
}

func TestRedisCorruptRecordsAreErrors(t *testing.T) {
	node := fixtureNode(t)
	raw, err := json.Marshal(node)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		payload   string
		keySuffix string
		ttl       time.Duration
		wantError string
	}{
		{name: "no TTL", payload: string(raw), ttl: 0, wantError: "TTL"},
		{name: "invalid JSON", payload: "{", ttl: TTL, wantError: "decode"},
		{name: "invalid schema", payload: strings.Replace(string(raw), `"schema_version":1`, `"schema_version":2`, 1), ttl: TTL, wantError: "schema_version"},
		{name: "wrong namespace", payload: strings.Replace(string(raw), `"namespace":"default"`, `"namespace":"other"`, 1), ttl: TTL, wantError: "namespace"},
		{name: "wrong ID", payload: strings.Replace(string(raw), "000000000001", "000000000002", 1), ttl: TTL, wantError: "instance_id"},
		{name: "malformed key", payload: string(raw), keySuffix: "*", ttl: TTL, wantError: "instance_id"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, _, store := redisTestStore(t, "default")
			suffix := node.InstanceID
			if test.keySuffix != "" {
				suffix = test.keySuffix
			}
			key := "tokenlive:gateway-versions:default:" + suffix
			if err := server.Set(key, test.payload); err != nil {
				t.Fatal(err)
			}
			if test.ttl > 0 {
				server.SetTTL(key, test.ttl)
			}
			nodes, err := store.List(context.Background())
			if err == nil || nodes != nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("corrupt List = %v, %v; want %q error", nodes, err, test.wantError)
			}
			if !server.Exists(key) {
				t.Fatal("read path silently deleted a corrupt record")
			}
		})
	}
}

// commandHook changes only the command boundary under test; all other commands
// still exercise go-redis against the isolated miniredis server.
type commandHook struct {
	before func(redis.Cmder) error
	after  func(redis.Cmder)
}

func (h commandHook) DialHook(next redis.DialHook) redis.DialHook { return next }

func (h commandHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func (h commandHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if h.before != nil {
			if err := h.before(cmd); err != nil {
				return err
			}
		}
		if err := next(ctx, cmd); err != nil {
			return err
		}
		if h.after != nil {
			h.after(cmd)
		}
		return nil
	}
}

func TestRedisExpiryDuringRead(t *testing.T) {
	for _, stage := range []string{"get", "pttl"} {
		t.Run(stage, func(t *testing.T) {
			server, client, store := redisTestStore(t, "default")
			ctx := context.Background()
			if err := store.Upsert(ctx, fixtureNode(t)); err != nil {
				t.Fatal(err)
			}
			client.AddHook(commandHook{before: func(cmd redis.Cmder) error {
				if cmd.Name() == stage {
					server.FastForward(TTL)
				}
				return nil
			}})
			nodes, err := store.List(ctx)
			if err != nil || nodes == nil || len(nodes) != 0 {
				t.Fatalf("expiry before %s = %v, %v; want empty successful result", stage, nodes, err)
			}
		})
	}
}

func TestRedisReadFailuresReturnNoPartialNodes(t *testing.T) {
	for _, stage := range []string{"get", "pttl"} {
		t.Run(stage, func(t *testing.T) {
			_, client, store := redisTestStore(t, "default")
			ctx := context.Background()
			node := fixtureNode(t)
			if err := store.Upsert(ctx, node); err != nil {
				t.Fatal(err)
			}
			node.InstanceID = "00000000-0000-4000-8000-000000000002"
			if err := store.Upsert(ctx, node); err != nil {
				t.Fatal(err)
			}
			wantErr := errors.New("injected read failure")
			count := 0
			client.AddHook(commandHook{before: func(cmd redis.Cmder) error {
				if cmd.Name() == stage {
					count++
					if count == 2 {
						return wantErr
					}
				}
				return nil
			}})
			nodes, err := store.List(ctx)
			if !errors.Is(err, wantErr) || nodes != nil {
				t.Fatalf("%s error returned partial/empty success: %v, %v", stage, nodes, err)
			}
		})
	}
}

func TestRedisScanPaginationAndDuplicateKeys(t *testing.T) {
	server, client, store := redisTestStore(t, "default")
	ctx := context.Background()
	node := fixtureNode(t)
	for i := 1; i <= 205; i++ {
		node.InstanceID = fmt.Sprintf("00000000-0000-4000-8000-%012d", i)
		if err := store.Upsert(ctx, node); err != nil {
			t.Fatal(err)
		}
	}
	for _, key := range []string{
		"tokenlive:gateway-versions:default-sibling:bad-record",
		"tokenlive:gateway-versions:DEFAULT:bad-record",
		"request-metrics:default:bad-record",
	} {
		if err := server.Set(key, "not a version report"); err != nil {
			t.Fatal(err)
		}
	}
	scans := 0
	var scannedKeys []string
	client.AddHook(commandHook{after: func(cmd redis.Cmder) {
		if cmd.Name() == "scan" {
			scans++
			scan := cmd.(*redis.ScanCmd)
			// miniredis ignores COUNT. Split its first real response into
			// pages to exercise our iteration and Redis's allowed duplicates.
			if scans == 1 {
				scannedKeys, _ = scan.Val()
			}
			start := (scans - 1) * 100
			end := min(start+100, len(scannedKeys))
			page := append([]string(nil), scannedKeys[start:end]...)
			var cursor uint64
			if end < len(scannedKeys) {
				cursor = uint64(scans)
			}
			if len(page) > 0 {
				page = append(page, page[0])
			}
			scan.SetVal(page, cursor)
		}
	}})
	nodes, err := store.List(ctx)
	if err != nil || len(nodes) != 205 {
		t.Fatalf("paginated List() = %d nodes, %v; want 205 unique nodes", len(nodes), err)
	}
	if scans < 2 {
		t.Fatalf("pagination test exercised only %d SCAN command", scans)
	}
	seen := make(map[string]bool)
	for _, node := range nodes {
		if node.Namespace != "default" || seen[node.InstanceID] {
			t.Fatalf("unexpected/duplicate node: %#v", node)
		}
		seen[node.InstanceID] = true
	}
}

func TestRedisCanceledContextNeverIssuesCommands(t *testing.T) {
	server, _, store := redisTestStore(t, "default")
	base := context.Background()
	node := fixtureNode(t)
	if err := store.Upsert(base, node); err != nil {
		t.Fatal(err)
	}
	before := server.CommandCount()
	ctx, cancel := context.WithCancel(base)
	cancel()
	if err := store.Upsert(ctx, node); !errors.Is(err, context.Canceled) {
		t.Fatalf("Upsert() = %v; want context.Canceled", err)
	}
	if nodes, err := store.List(ctx); !errors.Is(err, context.Canceled) || nodes != nil {
		t.Fatalf("List() = %v, %v; want context.Canceled", nodes, err)
	}
	if err := store.Delete(ctx, node.InstanceID); !errors.Is(err, context.Canceled) {
		t.Fatalf("Delete() = %v; want context.Canceled", err)
	}
	if after := server.CommandCount(); after != before {
		t.Fatalf("canceled operations reached Redis: %d -> %d commands", before, after)
	}
	if nodes, err := store.List(base); err != nil || !reflect.DeepEqual(nodes, []Node{node}) {
		t.Fatalf("canceled request mutated registry: %v, %v", nodes, err)
	}
}
