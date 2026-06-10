package tgqueue

import (
	"sync"
	"time"
)

// DedupNotifier wraps a Notifier and collapses rapid-fire messages to the
// same user into one: the last message wins, delivered after a short delay.
// Useful for outbid notifications when a hot lot is bid on many times/second.
type DedupNotifier struct {
	inner   NotifierIface
	window  time.Duration

	mu      sync.Mutex
	pending map[int64]*dedupEntry
}

type dedupEntry struct {
	text  string
	timer *time.Timer
}

func NewDedupNotifier(inner NotifierIface, window time.Duration) *DedupNotifier {
	return &DedupNotifier{
		inner:   inner,
		window:  window,
		pending: make(map[int64]*dedupEntry),
	}
}

func (d *DedupNotifier) Send(tgID int64, text string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if entry, ok := d.pending[tgID]; ok {
		// Update text and reset the timer — latest message wins.
		entry.text = text
		entry.timer.Reset(d.window)
		return
	}

	entry := &dedupEntry{text: text}
	entry.timer = time.AfterFunc(d.window, func() {
		d.mu.Lock()
		e, ok := d.pending[tgID]
		if ok {
			delete(d.pending, tgID)
		}
		d.mu.Unlock()
		if ok {
			d.inner.Send(tgID, e.text)
		}
	})
	d.pending[tgID] = entry
}
