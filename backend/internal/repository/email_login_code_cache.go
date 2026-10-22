package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/redis/go-redis/v9"
)

// Fork-local D85: isolated, hashed keys; atomic cooldown/hour quota and code TTL.
// Cooldown and hourly limits survive successful consumption and failed guesses.
var reserveEmailLoginCodeScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[2]) == 1 then return 0 end
local count = tonumber(redis.call('GET', KEYS[3]) or '0')
if count >= 5 then return 0 end
redis.call('INCR', KEYS[3])
if count == 0 then redis.call('EXPIRE', KEYS[3], 3600) end
redis.call('SET', KEYS[2], '1', 'EX', 60)
redis.call('SET', KEYS[1], ARGV[1], 'EX', 300)
redis.call('DEL', KEYS[4])
return 1
`)

// Compare, count every guess, and consume in one Redis operation. Only one
// concurrent correct caller can succeed; the fifth wrong guess burns the code.
var consumeEmailLoginCodeScript = redis.NewScript(`
local digest = redis.call('GET', KEYS[1])
if not digest then return 0 end
local count = redis.call('INCR', KEYS[2])
redis.call('PEXPIRE', KEYS[2], redis.call('PTTL', KEYS[1]))
if count <= 5 and digest == ARGV[1] then
 redis.call('DEL', KEYS[1], KEYS[2])
 return 1
end
if count >= 5 then redis.call('DEL', KEYS[1], KEYS[2]) end
return 0
`)

func emailLoginCodeKey(email string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return "email_login_code:{" + hex.EncodeToString(sum[:]) + "}"
}

func (c *emailCache) ReserveEmailLoginCode(ctx context.Context, email, digest string) (bool, error) {
	key := emailLoginCodeKey(email)
	n, err := reserveEmailLoginCodeScript.Run(ctx, c.rdb, []string{key, key + ":cooldown", key + ":hour", key + ":attempts"}, digest).Int()
	return n == 1, err
}

func (c *emailCache) ConsumeEmailLoginCode(ctx context.Context, email, digest string) (bool, error) {
	key := emailLoginCodeKey(email)
	n, err := consumeEmailLoginCodeScript.Run(ctx, c.rdb, []string{key, key + ":attempts"}, digest).Int()
	return n == 1, err
}
