package config

// ClickHouseConfig configures optional, read-only access to gateway usage logs.
type ClickHouseConfig struct {
	Enabled             bool
	Addr                []string
	Database            string
	Username            string
	Password            string
	TLS                 bool
	DialTimeoutSeconds  int `default:"3"`
	QueryTimeoutSeconds int `default:"5"`
}
