package util

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

// RawJSON represents raw encoded JSON bytes that implement sql.Scanner and driver.Valuer.
// It safely handles both []byte (BLOB/JSON) and string (TEXT) from various SQL drivers,
// preventing scan errors across database drivers and Go runtime versions.
type RawJSON []byte

// Value implements driver.Valuer.
func (j RawJSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return []byte(j), nil
}

// Scan implements sql.Scanner.
func (j *RawJSON) Scan(value any) error {
	if value == nil {
		*j = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*j = append((*j)[0:0], v...)
		return nil
	case string:
		*j = append((*j)[0:0], v...)
		return nil
	default:
		return fmt.Errorf("unsupported Scan, storing driver.Value type %T into type *RawJSON", value)
	}
}

// MarshalJSON returns j as the JSON encoding of j.
func (j RawJSON) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return j, nil
}

// UnmarshalJSON sets *j to a copy of data.
func (j *RawJSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("RawJSON: UnmarshalJSON on nil pointer")
	}
	*j = append((*j)[0:0], data...)
	return nil
}
