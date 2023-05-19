package main

import (
	"fmt"
	"sync/atomic"
	"time"
)

type metric struct {
	success  uint64
	fail     uint64
	duration uint64
	start    time.Time
}

func (m *metric) Measure(f func() bool) {
	start := time.Now()
	if m.start.IsZero() {
		m.start = start
	}
	defer func() {
		atomic.AddUint64(&m.duration, uint64(time.Since(start).Microseconds()))
	}()
	if f() {
		atomic.AddUint64(&m.success, 1)
		return
	}
	atomic.AddUint64(&m.fail, 1)
}
func (m *metric) Rps() string {
	sec := (float64(m.duration) / 1000_1000) // microsecond to second
	return fmt.Sprintf("%.1f est rps %.1f real rps",
		float64(m.success)/sec,                           // without delay
		float64(m.success)/time.Since(m.start).Seconds(), // with delay
	)

}

func (m *metric) RealStat() string {
	return fmt.Sprintf("ok:%d err:%d", m.success, m.fail)
}
