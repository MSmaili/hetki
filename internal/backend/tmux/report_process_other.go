//go:build !darwin && !linux

package tmux

import (
	"context"
	"errors"
)

func readReportProcess(context.Context, int) (reportProcess, error) {
	return reportProcess{}, errors.New("automatic report attachment is supported on macOS and Linux")
}
