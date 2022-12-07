package main

import "time"

type Configuration struct {
	Host            string
	Port            int
	LifxGroupName   string
	CaptureInterval time.Duration
	PixelDensity    int
}
