package ui

import "sync/atomic"

var wrapWidth atomic.Int64

func setWrapWidth(width int) {
	if width <= 0 {
		return
	}
	wrapWidth.Store(int64(width))
}

func currentWrapWidth() int {
	return int(wrapWidth.Load())
}
