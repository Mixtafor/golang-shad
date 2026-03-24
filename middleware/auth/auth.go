//go:build !solution

package auth

import (
	"context"
	"errors"
	"net/http"
	"regexp"
)

type User struct {
	Name  string
	Email string
}

func ContextUser(ctx context.Context) (*User, bool) {
	usr := ctx.Value("user")
	if usr == nil {
		return nil, false
	}

	user, ok := usr.(*User)
	if !ok {
		return nil, false
	}
	return user, true
}

var ErrInvalidToken = errors.New("invalid token")

type TokenChecker interface {
	CheckToken(ctx context.Context, token string) (*User, error)
}

func CheckAuth(checker TokenChecker) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			re := regexp.MustCompile(`^Bearer ([\w]+)$`)
			matches := re.FindStringSubmatch(auth)
			var token string
			if len(matches) > 1 {
				token = matches[1]
			} else {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			user, err := checker.CheckToken(r.Context(), token)
			if err != nil {
				if errors.Is(err, ErrInvalidToken) {
					w.WriteHeader(http.StatusUnauthorized)
				} else {
					w.WriteHeader(http.StatusInternalServerError)
				}
				return
			}

			ctx := context.WithValue(r.Context(), "user", user)
			reqWithCtx := r.WithContext(ctx)

			next.ServeHTTP(w, reqWithCtx)
		})
	}
}
