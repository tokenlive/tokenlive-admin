package util

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tokenlive/tokenlive-admin/pkg/errors"
)

// IsNilOrEmpty The function is used to check if a string pointer is nil or if the string it points to is empty.
func IsNilOrEmpty(value *string) bool {
	if value == nil {
		return true
	}
	return strings.TrimSpace(*value) == "" || len(*value) == 0
}

// IsEmptyOrBlank The function is used to check if a string is empty or if it contains only whitespace characters.
func IsEmptyOrBlank(value string) bool {
	return strings.TrimSpace(value) == "" || len(value) == 0
}

// StringPtr The function is used to create a string pointer from a string.
func StringPtr(s string) *string {
	return &s
}

// String The function is used to create a string from a string pointer.
func String(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ParseDurationString 将类似 "1s", "1m", "1h", "1d", "5w", "6y" 的字符串转换为 time.Duration
func ParseDurationString(durationStr string) (time.Duration, error) {
	// 正则表达式匹配数字和单位
	re := regexp.MustCompile(`^(\d+)([smhdwMyY])$`)
	matches := re.FindStringSubmatch(durationStr)

	if len(matches) != 3 {
		return 0, errors.BadRequest("", "无效的 duration 字符串: %s", durationStr)
	}

	// 提取数字部分
	value, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, errors.BadRequest("", "无效的数值: %s", matches[1])
	}

	// 提取单位部分
	unit := matches[2]

	// 根据单位计算 time.Duration
	switch unit {
	case "s":
		return time.Duration(value) * time.Second, nil
	case "m":
		return time.Duration(value) * time.Minute, nil
	case "h":
		return time.Duration(value) * time.Hour, nil
	case "d":
		return time.Duration(value) * 24 * time.Hour, nil
	case "w", "W":
		return time.Duration(value) * 7 * 24 * time.Hour, nil
	case "M": // 近似一个月，按照30天计算
		return time.Duration(value) * 30 * 24 * time.Hour, nil
	case "y", "Y": // 近似一年，按照365天计算
		return time.Duration(value) * 365 * 24 * time.Hour, nil
	default:
		return 0, errors.BadRequest("", "未知的单位: %s", unit)
	}
}

func TimeConvert(startTimeStr string, endTimeStr string) (time.Time, time.Time, error) {
	var startTime, endTime time.Time
	// Check if startTime and endTime are provided, otherwise set defaults
	if startTimeStr == "" || endTimeStr == "" {
		now := time.Now()
		endTime = now
		startTime = now.Add(-time.Hour) // 1 hour ago
	} else {
		// Parse startTime
		startTimeUnix, err := strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			// Handle startTime parsing error, e.g., return a 400 Bad Request
			return time.Time{}, time.Time{}, fmt.Errorf("Invalid startTime format. Use Unix timestamp (seconds)")
		}

		// Parse endTime
		endTimeUnix, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			// Handle endTime parsing error
			return time.Time{}, time.Time{}, fmt.Errorf("Invalid startTime format. Use Unix timestamp (seconds)")
		}
		switch len(startTimeStr) {
		case 10:
			startTimeUnix = startTimeUnix * 1000
			endTimeUnix = endTimeUnix * 1000
		case 13:
		default:
			return time.Time{}, time.Time{}, fmt.Errorf("Invalid startTime format. Use Unix timestamp in seconds or milliseconds")
		}
		startTime = time.UnixMilli(startTimeUnix)
		endTime = time.UnixMilli(endTimeUnix)
	}
	// You can also add a check to ensure startTime is not after endTime
	if startTime.After(endTime) {
		return startTime, endTime, fmt.Errorf("startTime cannot be after endTime")
	}
	return startTime, endTime, nil
}

// ConvertTimestampToTime converts a string timestamp in milliseconds to a time.Time object
func ConvertTimestampToTime(timestamp string) *time.Time {
	if IsEmptyOrBlank(timestamp) {
		return nil
	}
	// Convert the string to an int64
	milliseconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return nil
	}

	// Convert milliseconds to seconds and nanoseconds
	seconds := milliseconds / 1000
	nanoseconds := (milliseconds % 1000) * int64(time.Millisecond)

	// Create a time.Time object
	t := time.Unix(seconds, nanoseconds)
	return &t
}

func ConvertToFloat(value string) float64 {
	if IsEmptyOrBlank(value) {
		// If the value is empty or blank, return 100, such as in sidecar case
		return 100
	}
	// Convert the string to a float64
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return f
}
