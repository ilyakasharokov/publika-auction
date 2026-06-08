package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"

	bidsvc "publika-auction/internal/service/bid"
	"publika-auction/internal/metrics"
)

type SSEHub struct {
	mu   sync.RWMutex
	subs map[string]chan bidsvc.Event
}

func NewSSEHub() *SSEHub {
	return &SSEHub{subs: make(map[string]chan bidsvc.Event)}
}

func (h *SSEHub) Publish(e bidsvc.Event) {
	metrics.SSEEventsPublishedTotal.WithLabelValues(e.Type).Inc()
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.subs {
		select {
		case ch <- e:
		default:
		}
	}
}

func (h *SSEHub) subscribe() (string, <-chan bidsvc.Event) {
	id := uuid.New().String()
	ch := make(chan bidsvc.Event, 32)
	h.mu.Lock()
	h.subs[id] = ch
	h.mu.Unlock()
	metrics.SSESubscribers.Inc()
	return id, ch
}

func (h *SSEHub) unsubscribe(id string) {
	h.mu.Lock()
	if ch, ok := h.subs[id]; ok {
		close(ch)
		delete(h.subs, id)
	}
	h.mu.Unlock()
	metrics.SSESubscribers.Dec()
}

func (h *SSEHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	id, ch := h.subscribe()
	defer h.unsubscribe(id)

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case ev, open := <-ch:
			if !open {
				return
			}
			data, _ := json.Marshal(ev)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type, data)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
