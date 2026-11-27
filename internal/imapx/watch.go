// SPDX-License-Identifier: GPL-3.0-or-later

package imapx

import (
	"context"
	"errors"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// ErrNoIdle means the server cannot push changes; callers fall back to polling.
var ErrNoIdle = errors.New("imap: server does not support IDLE")

// RFC 2177 asks clients to re-issue IDLE at least every 29 minutes.
const idleRestart = 25 * time.Minute

// Watch keeps a dedicated connection in IDLE on folder and calls notify on every change
// until ctx is cancelled. ready fires once IDLE is running. Both callbacks must not block.
func (d *Dialer) Watch(ctx context.Context, user, pass, folder string, notify, ready func()) error {
	handler := &imapclient.UnilateralDataHandler{
		Expunge: func(uint32) { notify() },
		Mailbox: func(m *imapclient.UnilateralDataMailbox) {
			if m.NumMessages != nil {
				notify()
			}
		},
		Fetch: func(*imapclient.FetchMessageData) { notify() },
	}
	c, err := d.login(ctx, user, pass, handler)
	if err != nil {
		return err
	}
	defer c.Close()
	stop := context.AfterFunc(ctx, func() { _ = c.Close() })
	defer stop()

	if !c.Caps().Has(imap.CapIdle) && !c.Caps().Has(imap.CapIMAP4rev2) {
		return ErrNoIdle
	}
	if _, err := c.Select(folder, &imap.SelectOptions{ReadOnly: true}).Wait(); err != nil {
		return err
	}
	first := true
	for {
		idle, err := c.Idle()
		if err != nil {
			return ctxErr(ctx, err)
		}
		if first && ready != nil {
			first = false
			ready()
		}
		timer := time.NewTimer(idleRestart)
		ended := make(chan error, 1)
		go func() { ended <- idle.Wait() }()
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case err := <-ended:
			timer.Stop()
			return ctxErr(ctx, err)
		case <-timer.C:
			if err := idle.Close(); err != nil {
				return ctxErr(ctx, err)
			}
			if err := <-ended; err != nil {
				return ctxErr(ctx, err)
			}
		}
	}
}

func ctxErr(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err == nil {
		return ErrUnavailable
	}
	return err
}
