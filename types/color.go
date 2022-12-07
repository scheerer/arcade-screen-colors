package types

import (
	"image/color"
	"time"
)

type ScreenColor struct {
	Timestamp  time.Time
	Algorithms map[string]color.Color
}
