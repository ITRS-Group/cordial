/*
Copyright © 2022 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package instance

import (
	"errors"
	"log/slog"
	"os"
	"syscall"
	"time"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// Stop an instance. Options must be [StopOption] types but the
// signature will accept 'any' to avoid import loops from the geneos
// package.
//
// TODO: check for managed CAs and wait for those too.
func Stop(i geneos.Instance, options ...any) (err error) {
	var stoppedPID int

	stopOptions := []StopOption{}
	for _, o := range options {
		if option, ok := o.(StopOption); ok {
			stopOptions = append(stopOptions, option)
		}
	}
	opts := evalStopOptions(stopOptions...)

	if !opts.force && IsProtected(i) {
		return geneos.ErrProtected
	}

	wasRunning := IsRunning(i)
	if wasRunning {
		stoppedPID, _ = GetPID(i)
	}
	if err == nil && wasRunning && !opts.skipAudit {
		defer func() {
			args := []any{}
			if stoppedPID > 0 {
				args = append(args, slog.Int("pid", stoppedPID))
			}
			i.AuditEvent("stop", nil, args...)
		}()
	}

	if !wasRunning {
		i.Log().Debug("not running")
		return os.ErrProcessDone
	}

	if !opts.kill {
		if err = Signal(i, syscall.SIGTERM); err == os.ErrProcessDone {
			return nil
		}

		if errors.Is(err, syscall.EPERM) {
			return os.ErrPermission
		}

		for range 10 {
			time.Sleep(250 * time.Millisecond)
			if err = Signal(i, syscall.SIGTERM); err == os.ErrProcessDone {
				return nil
			}
		}

		if !IsRunning(i) {
			return nil
		}
	}

	if err = Signal(i, syscall.SIGKILL); err == os.ErrProcessDone {
		return nil
	}

	time.Sleep(250 * time.Millisecond)
	if !IsRunning(i) {
		return nil
	}
	return
}

type stopOptions struct {
	force     bool
	kill      bool
	skipAudit bool
}

type StopOption func(*stopOptions)

func evalStopOptions(options ...StopOption) *stopOptions {
	so := &stopOptions{}
	for _, opt := range options {
		opt(so)
	}
	return so
}

func WithStopForce(force bool) StopOption {
	return func(so *stopOptions) {
		so.force = force
	}
}

func WithStopKill(kill bool) StopOption {
	return func(so *stopOptions) {
		so.kill = kill
	}
}

// WithoutStopAudit suppresses the operational audit log entry for this stop.
func WithoutStopAudit() StopOption {
	return func(so *stopOptions) {
		so.skipAudit = true
	}
}
