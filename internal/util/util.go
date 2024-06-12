package util

import (
	"os"
)

func Getenv[T StringParsable](key string, def T) T {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	return ParseStringAs(v, def)
}
