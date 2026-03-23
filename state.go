package main

import "sync"

var (
	mu            sync.Mutex
	completed     int32
	total         int
	globalResults []HostResult
)
