// Package productversion identifies the running product and compares stable releases.
package productversion

// Build describes a binary's version and whether it is a release or development build.
type Build struct {
	Version string `json:"version"`
	Kind    string `json:"kind"`
}

// Identity is supplied by the executable or embedding host, never by configuration.
type Identity struct {
	Edition        string `json:"edition"`
	InstallChannel string `json:"install_channel"`
	Build          Build  `json:"build"`
}

// ResolveIdentity preserves explicit host identity and safely defaults legacy callers.
// Unknown identity values never promote a development build to a release.
func ResolveIdentity(explicit *Identity, legacyVersion string) Identity {
	if explicit == nil {
		if legacyVersion == "" {
			legacyVersion = "dev"
		}
		return Identity{
			Edition:        "professional",
			InstallChannel: "release",
			Build:          Build{Version: legacyVersion, Kind: "dev"},
		}
	}

	identity := *explicit
	if identity.Edition != "standalone" && identity.Edition != "professional" {
		identity.Edition = "unknown"
		identity.InstallChannel = "unknown"
		identity.Build.Kind = "dev"
		return identity
	}
	switch identity.InstallChannel {
	case "homebrew", "release", "unknown":
	default:
		identity.InstallChannel = "unknown"
	}
	if identity.Build.Kind != "release" {
		identity.Build.Kind = "dev"
	}
	return identity
}
