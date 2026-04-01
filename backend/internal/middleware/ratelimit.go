package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/y2k-pixel-blog/backend/internal/model"
)

// LoginAttempt tracks login attempts per IP
type LoginAttempt struct {
	FailCount    int
	LastFail     time.Time
	BlockedUntil time.Time
}

// CaptchaEntry stores a pending captcha challenge
type CaptchaEntry struct {
	Answer    int
	ExpiresAt time.Time
}

// RateLimiter provides IP-based rate limiting and captcha for login
type RateLimiter struct {
	mu            sync.RWMutex
	attempts      map[string]*LoginAttempt // IP -> attempt info
	captchas      map[string]*CaptchaEntry // token -> answer
	captchaThreshold int  // require captcha after this many fails
	blockThreshold   int  // block IP after this many fails
	blockDuration    time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		attempts:         make(map[string]*LoginAttempt),
		captchas:         make(map[string]*CaptchaEntry),
		captchaThreshold: 3,
		blockThreshold:   10,
		blockDuration:    15 * time.Minute,
	}
	// Clean up expired entries every 5 minutes
	go rl.cleanup()
	return rl
}

// cleanup removes expired entries periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, attempt := range rl.attempts {
			if !attempt.BlockedUntil.IsZero() && now.After(attempt.BlockedUntil) {
				delete(rl.attempts, ip)
			} else if now.Sub(attempt.LastFail) > 30*time.Minute {
				delete(rl.attempts, ip)
			}
		}
		for token, captcha := range rl.captchas {
			if now.After(captcha.ExpiresAt) {
				delete(rl.captchas, token)
			}
		}
		rl.mu.Unlock()
	}
}

// GetIPStatus returns the current status for an IP
func (rl *RateLimiter) GetIPStatus(ip string) (failCount int, blocked bool, needsCaptcha bool) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	attempt, exists := rl.attempts[ip]
	if !exists {
		return 0, false, false
	}

	now := time.Now()
	if !attempt.BlockedUntil.IsZero() && now.Before(attempt.BlockedUntil) {
		return attempt.FailCount, true, true
	}

	return attempt.FailCount, false, attempt.FailCount >= rl.captchaThreshold
}

// RecordFail records a failed login attempt
func (rl *RateLimiter) RecordFail(ip string) (failCount int, blocked bool) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	attempt, exists := rl.attempts[ip]
	if !exists {
		attempt = &LoginAttempt{}
		rl.attempts[ip] = attempt
	}

	attempt.FailCount++
	attempt.LastFail = time.Now()

	if attempt.FailCount >= rl.blockThreshold {
		attempt.BlockedUntil = time.Now().Add(rl.blockDuration)
		return attempt.FailCount, true
	}

	return attempt.FailCount, false
}

// RecordSuccess resets the counter for an IP on successful login
func (rl *RateLimiter) RecordSuccess(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.attempts, ip)
}

// GenerateCaptcha creates a new math captcha challenge
func (rl *RateLimiter) GenerateCaptcha() (token string, question string) {
	// Generate random math problem
	a, _ := rand.Int(rand.Reader, big.NewInt(20))
	b, _ := rand.Int(rand.Reader, big.NewInt(20))
	numA := int(a.Int64()) + 1
	numB := int(b.Int64()) + 1

	// Randomly choose operation
	opRand, _ := rand.Int(rand.Reader, big.NewInt(3))
	var answer int
	var op string

	switch opRand.Int64() {
	case 0:
		op = "+"
		answer = numA + numB
	case 1:
		op = "-"
		if numA < numB {
			numA, numB = numB, numA // ensure positive result
		}
		answer = numA - numB
	case 2:
		numB = int(b.Int64())%9 + 1 // keep multiplier small
		op = "×"
		answer = numA * numB
	}

	question = fmt.Sprintf("%d %s %d = ?", numA, op, numB)

	// Generate token
	tokenBytes := make([]byte, 16)
	rand.Read(tokenBytes)
	token = hex.EncodeToString(tokenBytes)

	// Store with 5 minute expiry
	rl.mu.Lock()
	rl.captchas[token] = &CaptchaEntry{
		Answer:    answer,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	rl.mu.Unlock()

	return token, question
}

// VerifyCaptcha checks if the answer matches the token's challenge
func (rl *RateLimiter) VerifyCaptcha(token string, answer int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.captchas[token]
	if !exists {
		return false
	}

	// Always delete after use (one-time)
	delete(rl.captchas, token)

	if time.Now().After(entry.ExpiresAt) {
		return false
	}

	return entry.Answer == answer
}

// LoginRateLimit returns a gin handler that checks IP rate limits
func (rl *RateLimiter) LoginRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		_, blocked, _ := rl.GetIPStatus(ip)

		if blocked {
			c.JSON(http.StatusTooManyRequests, model.ErrorResponse{
				Error: "Too many failed attempts. IP temporarily blocked. Try again later.",
				Code:  "IP_BLOCKED",
			})
			c.Abort()
			return
		}

		// Store rate limiter in context for handler use
		c.Set("rateLimiter", rl)
		c.Next()
	}
}

// CaptchaHandler handles GET /api/admin/captcha
func (rl *RateLimiter) CaptchaHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, question := rl.GenerateCaptcha()
		c.JSON(http.StatusOK, gin.H{
			"token":    token,
			"question": question,
		})
	}
}

// LoginStatusHandler handles GET /api/admin/login/status
func (rl *RateLimiter) LoginStatusHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		failCount, blocked, needsCaptcha := rl.GetIPStatus(ip)

		resp := gin.H{
			"needsCaptcha": needsCaptcha,
			"blocked":      blocked,
			"failCount":    failCount,
		}
		if blocked {
			rl.mu.RLock()
			if attempt, ok := rl.attempts[ip]; ok {
				resp["blockedUntil"] = attempt.BlockedUntil.Format(time.RFC3339)
				resp["retryAfterSeconds"] = int(time.Until(attempt.BlockedUntil).Seconds())
			}
			rl.mu.RUnlock()
		}
		c.JSON(http.StatusOK, resp)
	}
}
