//go:build !solution

package httpgauge

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
)

type Gauge struct {
	mu      sync.RWMutex
	metrics map[string]int
}

func New() *Gauge {
	return &Gauge{
		metrics: make(map[string]int),
	}
}

func (g *Gauge) Snapshot() map[string]int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return maps.Clone(g.metrics)
}

// ServeHTTP returns accumulated statistics in text format ordered by pattern.
//
// For example:
//
//	/a 10
//	/b 5
//	/c/{id} 7
func (g *Gauge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sb := strings.Builder{}
	g.mu.RLock()

	keys := make([]string, 0, len(g.metrics))
	for k := range g.metrics {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteByte(' ')
		sb.WriteString(fmt.Sprintf("%d", g.metrics[k]))
		sb.WriteByte('\n')
	}

	w.Write([]byte(sb.String()))
	g.mu.RUnlock()
}

func (g *Gauge) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rctx := chi.RouteContext(r.Context())
			if rctx == nil {
				http.Error(w, "no route context", http.StatusInternalServerError)
				return
			}

			g.mu.Lock()
			g.metrics[rctx.RoutePattern()]++
			g.mu.Unlock()
		}()

		next.ServeHTTP(w, r)
	})
}
