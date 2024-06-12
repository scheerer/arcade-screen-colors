package util

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const defaultTimestampLayout = time.RFC3339

type StringParsable interface {
	string | []string | int | []int | int64 | []int64 | float64 | bool | time.Duration | []time.Duration | map[string]time.Duration | decimal.Decimal | []decimal.Decimal | time.Time
}

func envVarStringSplitter(s string) []string {
	parts := strings.Split(s, ",")
	v := make([]string, 0, len(parts))
	for _, p := range parts {
		v = append(v, strings.TrimSpace(p))
	}
	return v
}

func envSliceTypeParser[T StringParsable](s string, f func(string) (T, error)) ([]T, error) {
	parts := envVarStringSplitter(s)
	v := make([]T, 0, len(parts))
	for _, p := range parts {
		v2, err := f(p)
		if err != nil {
			return v, err
		}
		v = append(v, v2)
	}
	return v, nil
}

// ParseStringAs parses the input string as a StringParsable type, returning the default
// if an error occurs. It will panic if the type from StringParsable is not implemented.
func ParseStringAs[T StringParsable](v string, def T) T {
	v = strings.Trim(v, `"`) // in case something comes in as if it were a json string

	var parser func(string) (any, error)
	switch any(def).(type) {
	case string:
		parser = func(s string) (any, error) { return s, nil }
	case []string:
		parser = func(s string) (any, error) {
			return envSliceTypeParser(s, func(s string) (string, error) { return s, nil })
		}
	case int:
		parser = func(s string) (any, error) { return strconv.Atoi(s) }
	case []int:
		parser = func(s string) (any, error) {
			return envSliceTypeParser(s, strconv.Atoi)
		}
	case int64:
		parser = func(s string) (any, error) { return strconv.ParseInt(s, 0, 64) }
	case []int64:
		parser = func(s string) (any, error) {
			return envSliceTypeParser(s, func(s2 string) (int64, error) {
				return strconv.ParseInt(s2, 0, 64)
			})
		}
	case time.Duration:
		parser = func(s string) (any, error) { return time.ParseDuration(s) }
	case []time.Duration:
		parser = func(s string) (any, error) {
			return envSliceTypeParser(s, time.ParseDuration)
		}
	case bool:
		parser = func(s string) (any, error) { return strconv.ParseBool(s) }
	case float64:
		parser = func(s string) (any, error) { return strconv.ParseFloat(s, 64) }
	case decimal.Decimal:
		parser = func(s string) (any, error) { return decimal.NewFromString(s) }
	case []decimal.Decimal:
		parser = func(s string) (any, error) {
			return envSliceTypeParser(s, decimal.NewFromString)
		}
	case time.Time:
		parser = func(s string) (any, error) {
			return time.Parse(defaultTimestampLayout, s)
		}
	default:
		panic("ParseStringAs got a type we can't handle")
	}

	val, err := parser(v)
	if err != nil {
		return def
	}
	return val.(T)
}

func ParseString(src any) string {
	if src == nil {
		return ""
	}
	switch v := src.(type) {
	case string:
		return v
	case []uint8:
		return string(v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		return ""
	}
}

func RandomString(l int) string {
	id := uuid.NewString()
	s := strings.ReplaceAll(id, "-", "")
	for len(s) < l {
		id = uuid.NewString()
		t := strings.ReplaceAll(id, "-", "")
		s = s + t
	}
	return s[:l]
}

func ParseInt64(m any) int64 {
	switch val := m.(type) {
	case int:
		return int64(val)
	case int64:
		return val
	case []uint8:
		id64, parseErr := strconv.ParseInt(string(val), 10, 64)
		if parseErr != nil {
			return 0
		}
		return id64
	case string:
		conv, _ := strconv.ParseInt(m.(string), 10, 64)
		return conv
	case float32:
		return int64(val)
	case float64:
		return int64(val)
	default:
		return 0
	}
}

func ParseTimestamp(b any) time.Time {
	t, _ := time.Parse(defaultTimestampLayout, string(b.([]uint8)))
	return t
}

func ParseDecimal(b any) decimal.Decimal {
	d, _ := decimal.NewFromString(ParseString(b))
	return d
}
