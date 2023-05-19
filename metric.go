package main

import (
	"fmt"
	"sync/atomic"
	"time"
)

type metric struct {
	success  uint64
	fail     uint64
	duration time.Duration
	start    time.Time
}

func (m *metric) Measure(f func() bool) {
	start := time.Now()
	if m.start.IsZero() {
		m.start = start
	}
	defer func() {
		m.duration += time.Since(start)
	}()
	if f() {
		atomic.AddUint64(&m.success, 1)
		return
	}
	atomic.AddUint64(&m.fail, 1)
}
func (m *metric) Rps() string {
	return fmt.Sprintf("%.1f est rps %.1f real rps",
		float64(m.success)/m.duration.Seconds(), // without overhead
		float64(m.success)/time.Since(m.start).Seconds(),
	)

}

func (m *metric) RealStat() string {
	return fmt.Sprintf("ok:%d err:%d", m.success, m.fail)
}
