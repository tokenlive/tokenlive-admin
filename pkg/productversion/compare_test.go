package productversion

import "testing"

func TestCompare(t *testing.T) {
	cases := []struct{ current, kind, target, want string }{
		{"v1.2.3", "release", "1.2.4", "available"},
		{"v1.2.3", "release", "1.2.3", "current"},
		{"v2.0.0", "release", "1.9.0", "ahead"},
		{"v1.2.3", "dev", "1.2.4", "uncomparable"},
		{"git-abc", "dev", "1.2.4", "uncomparable"},
		{"1.2.3", "release", "1.3.0-rc.1", "no_candidate"},
		{"v01.2.3", "release", "1.2.4", "uncomparable"},
		{"1.2.3", "", "1.2.4", "uncomparable"},
		{"1.2.3", "unknown", "1.2.4", "uncomparable"},
		{"1.2.3", "Release", "1.2.4", "uncomparable"},
		{"v1.2.3-rc.1", "release", "1.2.4", "uncomparable"},
		{"v1.2", "release", "1.2.4", "uncomparable"},
		{"v1.2.3", "release", "", "no_candidate"},
		{"v1.2.3", "release", "1.3", "no_candidate"},
		{"v1.2.3", "release", "v01.3.0", "no_candidate"},
		{"v1.2.3+build.1", "release", "1.2.3+build.2", "current"},
		{"1.9.0", "release", "1.10.0", "available"},
		{"dev", "dev", "", "uncomparable"},
	}
	for _, tc := range cases {
		if got := Compare(Build{tc.current, tc.kind}, tc.target); got != tc.want {
			t.Errorf("%+v: got %s", tc, got)
		}
	}
}

func TestStableVersion(t *testing.T) {
	cases := []struct {
		raw, want string
		valid     bool
	}{
		{"1.2.3", "v1.2.3", true},
		{"v0.0.0", "v0.0.0", true},
		{"v1.2.3+build.001", "v1.2.3", true},
		{"1.2.3+linux-amd64.sha", "v1.2.3", true},
		{"v999999999999999999999.2.3", "v999999999999999999999.2.3", true},
		{"", "", false},
		{"dev", "", false},
		{"git-abc", "", false},
		{"v1", "", false},
		{"1.2", "", false},
		{"1.2.3.4", "", false},
		{"v01.2.3", "", false},
		{"1.02.3", "", false},
		{"1.2.03", "", false},
		{"1.2.3-rc.1", "", false},
		{"v1.2.3-rc.1+build.1", "", false},
		{"1.2.3+", "", false},
		{"1.2.3+bad..metadata", "", false},
		{"1.2.3+bad_metadata", "", false},
		{" v1.2.3", "", false},
		{"v1.2.3\n", "", false},
		{"V1.2.3", "", false},
		{"vv1.2.3", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			got, valid := StableVersion(tc.raw)
			if got != tc.want || valid != tc.valid {
				t.Errorf("StableVersion(%q) = (%q, %v), want (%q, %v)", tc.raw, got, valid, tc.want, tc.valid)
			}
		})
	}
}

func TestResolveIdentity(t *testing.T) {
	cases := []struct {
		name     string
		explicit *Identity
		legacy   string
		want     Identity
	}{
		{
			name: "legacy version remains a development build", legacy: "v1.2.3",
			want: Identity{"professional", "release", Build{"v1.2.3", "dev"}},
		},
		{
			name: "missing legacy version defaults to dev",
			want: Identity{"professional", "release", Build{"dev", "dev"}},
		},
		{
			name: "explicit standalone takes priority", legacy: "v9.9.9",
			explicit: &Identity{"standalone", "homebrew", Build{"v1.2.3", "release"}},
			want:     Identity{"standalone", "homebrew", Build{"v1.2.3", "release"}},
		},
		{
			name:     "explicit professional release is retained",
			explicit: &Identity{"professional", "release", Build{"v2.3.4+build.1", "release"}},
			want:     Identity{"professional", "release", Build{"v2.3.4+build.1", "release"}},
		},
		{
			name:     "invalid edition demotes all identity trust",
			explicit: &Identity{"enterprise", "release", Build{"raw-version", "release"}},
			want:     Identity{"unknown", "unknown", Build{"raw-version", "dev"}},
		},
		{
			name:     "empty edition does not guess professional",
			explicit: &Identity{"", "homebrew", Build{"v1.2.3", "release"}},
			want:     Identity{"unknown", "unknown", Build{"v1.2.3", "dev"}},
		},
		{
			name:     "invalid channel does not guess release",
			explicit: &Identity{"professional", "other", Build{"v1.2.3", "release"}},
			want:     Identity{"professional", "unknown", Build{"v1.2.3", "release"}},
		},
		{
			name:     "missing build kind is dev",
			explicit: &Identity{"standalone", "homebrew", Build{"v1.2.3", ""}},
			want:     Identity{"standalone", "homebrew", Build{"v1.2.3", "dev"}},
		},
		{
			name:     "unknown build kind is dev",
			explicit: &Identity{"professional", "release", Build{"v1.2.3", "nightly"}},
			want:     Identity{"professional", "release", Build{"v1.2.3", "dev"}},
		},
		{
			name: "explicit empty version does not borrow legacy", legacy: "v9.9.9",
			explicit: &Identity{"standalone", "unknown", Build{"", "dev"}},
			want:     Identity{"standalone", "unknown", Build{"", "dev"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var original Identity
			if tc.explicit != nil {
				original = *tc.explicit
			}
			if got := ResolveIdentity(tc.explicit, tc.legacy); got != tc.want {
				t.Errorf("ResolveIdentity() = %+v, want %+v", got, tc.want)
			}
			if tc.explicit != nil && *tc.explicit != original {
				t.Errorf("ResolveIdentity mutated caller identity: %+v", *tc.explicit)
			}
		})
	}
}
