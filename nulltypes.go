package gorp

import (
	"database/sql/driver"
	"time"
)

// A nullable Time value
type NullTime struct {
	Time  time.Time
	Valid bool // Valid is true if Time is not NULL
}

// Scan implements the Scanner interface.
func (nt *NullTime) Scan(value interface{}) error {
	switch t := value.(type) {
	case time.Time:
		nt.Time, nt.Valid = t, true
	case []byte:
		nt.Valid = false
		for _, dtfmt := range []string{
			"2006-01-02 15:04:05.999999999",
			"2006-01-02T15:04:05.999999999",
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05",
			"2006-01-02 15:04",
			"2006-01-02T15:04",
			"2006-01-02",
			"2006-01-02 15:04:05-07:00",
		} {
			var err error
			if nt.Time, err = time.Parse(dtfmt, string(t)); err == nil {
				nt.Valid = true
				break
			}
		}
	}
	return nil
}

// Value implements the driver Valuer interface.
func (nt NullTime) Value() (driver.Value, error) {
	if !nt.Valid {
		return nil, nil
	}
	return nt.Time, nil
}
