package time_ptypes

import (
	"database/sql/driver"
	"fmt"
	"time"
)

type NullTime struct {
	Time  time.Time
	Valid bool
}

// Value implements the driver.Valuer interface.
func (u NullTime) Value() (driver.Value, error) {
	if !u.Valid {
		return nil, nil
	}
	return u.Time, nil
}

// Scan implements the sql.Scanner interface.
func (u *NullTime) Scan(src interface{}) error {
	if src == nil {
		u.Valid = false
		return nil
	}

	u.Valid = true
	switch src := src.(type) {
	case []byte:
		if len(src) == 16 {
			return u.Time.UnmarshalBinary(src)
		}
		return u.Time.UnmarshalText(src)

	case string:
		return u.Time.UnmarshalText([]byte(src))
	}

	return fmt.Errorf("NullTime: cannot convert %T to Time", src)
}

// MarshalBinary implements the encoding.BinaryMarshaler interface.
func (t NullTime) MarshalBinary() ([]byte, error) {
	if !t.Valid {
		return nil, nil
	}

	return t.Time.MarshalBinary()
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface.
func (t *NullTime) UnmarshalBinary(data []byte) error {
	if len(data) == 0 {
		t.Valid = false
		return nil
	}

	return t.Time.UnmarshalBinary(data)
}

// MarshalJSON implements the json.Marshaler interface.
// The time is a quoted string in RFC 3339 format, with sub-second precision added if present.
func (t NullTime) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return nil, nil
	}

	return t.Time.MarshalJSON()
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// The time is expected to be a quoted string in RFC 3339 format.
func (t *NullTime) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		t.Valid = false
		return nil
	}

	return t.Time.UnmarshalJSON(data)
}

// MarshalText implements the encoding.TextMarshaler interface.
// The time is formatted in RFC 3339 format, with sub-second precision added if present.
func (t NullTime) MarshalText() ([]byte, error) {
	if !t.Valid {
		return nil, nil
	}

	return t.Time.MarshalText()
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
// The time is expected to be in RFC 3339 format.
func (t *NullTime) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		t.Valid = false
		return nil
	}

	return t.Time.UnmarshalText(data)
}
