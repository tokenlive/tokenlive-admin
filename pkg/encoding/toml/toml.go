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
var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

type Value = toml.Primitive

func Unmarshal(buf []byte, v interface{}) error {
	// 替换环境变量占位符
	expanded := expandEnvVars(string(buf))
	return toml.Unmarshal([]byte(expanded), v)
}

// expandEnvVars 替换字符串中的环境变量占位符
// 支持格式: ${VAR} 或 ${VAR:default}
func expandEnvVars(s string) string {
	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		// 移除 ${ 和 }
		inner := match[2 : len(match)-1]

		// 检查是否有默认值
		parts := strings.SplitN(inner, ":", 2)
		envName := parts[0]

		// 获取环境变量
		envValue := os.Getenv(envName)
		if envValue == "" && len(parts) > 1 {
			envValue = parts[1]
		}

		return envValue
	})
}

func Marshal(v interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := toml.NewEncoder(buf).Encode(v)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
