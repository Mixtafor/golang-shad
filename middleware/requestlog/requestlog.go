//go:build !solution

package requestlog

import (
	"crypto/rand"
	"math/big"
	"net/http"

	"github.com/felixge/httpsnoop"
	"go.uber.org/zap"
)

func Log(l *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lenMaxKey := 30
			maxInt := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(lenMaxKey)), nil)
			keyGen, err := rand.Int(rand.Reader, maxInt)

			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			key := keyGen.String()
			l.Info("request started", zap.String("path", r.URL.Path), zap.String("method", r.Method),
				zap.String("request_id", key))

			m := httpsnoop.Metrics{}

			defer func() {
				if rec := recover(); rec != nil {
					l.Info("request panicked", zap.String("path", r.URL.Path),
						zap.String("method", r.Method), zap.String("request_id", key))
					panic(rec)
				} else {
					l.Info("request finished", zap.String("path", r.URL.Path),
						zap.String("method", r.Method), zap.String("request_id", key),
						zap.Int("duration", int(m.Duration.Seconds()*1000)), zap.Int("status_code", m.Code))
				}
			}()

			m = httpsnoop.CaptureMetrics(next, w, r)
		})
	}
}
