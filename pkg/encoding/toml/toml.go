package toml

import (
	"bytes"
	"os"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

var (
	DecodeFile = toml.DecodeFile
	Decode     = toml.Decode
)

// 匹配 ${VAR} 或 ${VAR:default} 格式的环境变量占位符
var (
	// 匹配被双引号包裹的整个环境变量占位符，例如 "${VAR}" 或 "${VAR:default}"
	quotedEnvVarPattern = regexp.MustCompile(`"(\$\{([^}]+)\})"`)
	// 匹配未被双引号包裹的环境变量占位符
	envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)
	// 匹配布尔值或数字
	boolOrNumPattern = regexp.MustCompile(`^(true|false|[0-9]+)$`)
)

type Value = toml.Primitive

func Unmarshal(buf []byte, v interface{}) error {
	// 替换环境变量占位符
	expanded := expandEnvVars(string(buf))
	return toml.Unmarshal([]byte(expanded), v)
}

// expandEnvVars 替换字符串中的环境变量占位符
// 支持格式: ${VAR} 或 ${VAR:default}
func expandEnvVars(s string) string {
	// 1. 先替换被双引号包裹的占位符
	s = quotedEnvVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		// match 格式为 "${VAR:default}"
		// 移除前后的双引号，得到 ${VAR:default}
		innerMatch := match[1 : len(match)-1]
		// 移除 ${ 和 } 得到 VAR:default
		inner := innerMatch[2 : len(innerMatch)-1]

		parts := strings.SplitN(inner, ":", 2)
		envName := parts[0]
		envValue := os.Getenv(envName)
		if envValue == "" && len(parts) > 1 {
			envValue = parts[1]
		}

		// 如果替换后的值是布尔或数字，则不加双引号返回
		if boolOrNumPattern.MatchString(envValue) {
			return envValue
		}
		// 否则，重新用双引号包裹返回
		return `"` + envValue + `"`
	})

	// 2. 再替换其余未被双引号包裹的占位符
	s = envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		inner := match[2 : len(match)-1]
		parts := strings.SplitN(inner, ":", 2)
		envName := parts[0]
		envValue := os.Getenv(envName)
		if envValue == "" && len(parts) > 1 {
			envValue = parts[1]
		}
		return envValue
	})

	return s
}

func Marshal(v interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := toml.NewEncoder(buf).Encode(v)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
