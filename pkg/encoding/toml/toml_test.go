package toml

import "testing"

func TestTomlDecode(t *testing.T) {
	var config struct {
		Middlewares []struct {
			Name    string `toml:"name"`
			Options Value  `toml:"options"`
		} `toml:"middlewares"`
	}

	md, err := Decode(`
	middlewares = [
  		{name = "ratelimit", options = {max = 10, period = 10}},
	]
	`, &config)
	if err != nil {
		t.Error(err)
		return
	}

	var rateLimitConfig struct {
		Max    int `toml:"max"`
		Period int `toml:"period"`
	}
	err = md.PrimitiveDecode(config.Middlewares[0].Options, &rateLimitConfig)
	if err != nil {
		t.Error(err)
		return
	}
	if rateLimitConfig.Max != 10 || rateLimitConfig.Period != 10 {
		t.Errorf("Expected {Max: 10, Period: 10}, got %v", rateLimitConfig)
	}
}

func TestUnmarshalEnv(t *testing.T) {
	t.Setenv("TEST_BOOL_TRUE", "true")
	t.Setenv("TEST_BOOL_FALSE", "false")
	t.Setenv("TEST_INT", "123")
	t.Setenv("TEST_STR", "hello")

	type Config struct {
		BoolTrue  bool   `toml:"bool_true"`
		BoolFalse bool   `toml:"bool_false"`
		IntVal    int    `toml:"int_val"`
		StrVal    string `toml:"str_val"`
		DefaultB  bool   `toml:"default_b"`
	}

	// 模拟带引号的环境变量占位符
	tomlStr := `
	bool_true = "${TEST_BOOL_TRUE:false}"
	bool_false = "${TEST_BOOL_FALSE:true}"
	int_val = "${TEST_INT:0}"
	str_val = "${TEST_STR:default}"
	default_b = "${TEST_NON_EXIST:false}"
	`

	var cfg Config
	err := Unmarshal([]byte(tomlStr), &cfg)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if !cfg.BoolTrue {
		t.Errorf("Expected BoolTrue to be true, got %v", cfg.BoolTrue)
	}
	if cfg.BoolFalse {
		t.Errorf("Expected BoolFalse to be false, got %v", cfg.BoolFalse)
	}
	if cfg.IntVal != 123 {
		t.Errorf("Expected IntVal to be 123, got %d", cfg.IntVal)
	}
	if cfg.StrVal != "hello" {
		t.Errorf("Expected StrVal to be 'hello', got '%s'", cfg.StrVal)
	}
	if cfg.DefaultB {
		t.Errorf("Expected DefaultB to be false, got %v", cfg.DefaultB)
	}
}

