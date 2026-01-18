package rate

import (
	"golang.org/x/time/rate"
)

type Limiter interface {
	Allow() bool
}

type TokenLimiter struct {
	limiter *rate.Limiter
}

func NewTokenLimiter(rps float64, burst int) *TokenLimiter {
	return &TokenLimiter{
		limiter: rate.NewLimiter(rate.Limit(rps), burst),
	}
}

func (l *TokenLimiter) Allow() bool {
	return l.limiter.Allow()
}
