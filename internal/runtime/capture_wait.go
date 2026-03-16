package runtime

import (
	"time"

	"github.com/salman-frs/meridian/internal/model"
)

func (r *Runner) waitForCapture(snapshot func() []model.SignalCapture, signals []model.SignalType) []model.SignalCapture {
	deadline := r.now().Add(r.options.CaptureTimeout)
	for {
		captures := snapshot()
		if allSignalsCaptured(captures, signals) || r.now().After(deadline) {
			return captures
		}
		r.sleep(200 * time.Millisecond)
	}
}

func allSignalsCaptured(captures []model.SignalCapture, signals []model.SignalType) bool {
	for _, signal := range signals {
		capture := model.SignalCapture{Signal: signal}
		for _, item := range captures {
			if item.Signal == signal {
				capture = item
				break
			}
		}
		if capture.Count < 1 {
			return false
		}
	}
	return true
}
