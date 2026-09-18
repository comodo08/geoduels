package main

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const guestSignupRateLimitKeyPrefix = "api:ratelimit:guest_signup:ip:"
const guestSignupDailyRateLimitKeyPrefix = "api:ratelimit:guest_signup:daily:ip:"

var guestSignupRateLimitScript = redis.NewScript(`
local function bump(key, ttl_ms, limit)
  if limit <= 0 then
    return {1, 0}
  end
  local count = redis.call("INCR", key)
  if count == 1 then
    redis.call("PEXPIRE", key, ttl_ms)
  end
  local ttl = redis.call("PTTL", key)
  if count <= limit then
    return {1, 0}
  end
  return {0, ttl}
end

local short = bump(KEYS[1], tonumber(ARGV[1]), tonumber(ARGV[2]))
if short[1] == 0 then
  return short
end
return bump(KEYS[2], tonumber(ARGV[3]), tonumber(ARGV[4]))
`)

func (a *api) checkGuestSignupRateLimit(r *http.Request) (bool, time.Duration, error) {
	if a.guestSignupIPLimit <= 0 && a.guestSignupDailyLimit <= 0 {
		return true, 0, nil
	}
	if a.redis == nil {
		return false, 0, errors.New("guest signup rate limit requires redis")
	}
	window := a.guestSignupIPWindow
	if window <= 0 {
		window = 10 * time.Minute
	}
	ip := a.clientIP(r)
	if ip == "" {
		ip = "unknown"
	}
	dailyWindow := a.guestSignupDailyWindow
	if dailyWindow <= 0 {
		dailyWindow = 24 * time.Hour
	}
	ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
	defer cancel()

	key := guestSignupRateLimitKeyPrefix + ip
	dailyKey := guestSignupDailyRateLimitKeyPrefix + ip
	result, err := guestSignupRateLimitScript.Run(
		ctx,
		a.redis,
		[]string{key, dailyKey},
		window.Milliseconds(),
		a.guestSignupIPLimit,
		dailyWindow.Milliseconds(),
		a.guestSignupDailyLimit,
	).Slice()
	if err != nil {
		return false, 0, err
	}
	if len(result) != 2 {
		return false, 0, errors.New("unexpected guest signup rate limit response")
	}
	allowed, err := redisInt64(result[0])
	if err != nil {
		return false, 0, err
	}
	ttlMillis, err := redisInt64(result[1])
	if err != nil {
		return false, 0, err
	}
	if allowed == 1 {
		return true, 0, nil
	}
	retryAfter := time.Duration(ttlMillis) * time.Millisecond
	if retryAfter <= 0 {
		retryAfter = dailyWindow
	}
	return false, retryAfter, nil
}

func writeRateLimited(w http.ResponseWriter, retryAfter time.Duration) {
	if retryAfter > 0 {
		seconds := int(retryAfter.Round(time.Second).Seconds())
		if seconds < 1 {
			seconds = 1
		}
		w.Header().Set("Retry-After", strconv.Itoa(seconds))
	}
	http.Error(w, "too many guest signups", http.StatusTooManyRequests)
}

func redisInt64(v any) (int64, error) {
	switch n := v.(type) {
	case int64:
		return n, nil
	case int:
		return int64(n), nil
	case string:
		return strconv.ParseInt(n, 10, 64)
	default:
		return 0, errors.New("unexpected redis integer type")
	}
}
