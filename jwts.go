package vbdb

import (
	"fmt"
	"time"

	"github.com/vikebot/vbcore"
	"go.uber.org/zap"
)

// JwtIsBlacklistedCtx checks whether the passed `jti` (JWT ID) is already
// blacklisted or not
func JwtIsBlacklistedCtx(jti string, ctx *zap.Logger) (blacklisted bool, success bool) {
	var valid int
	exists, err := s.SelectExists("SELECT id FROM jwts WHERE jti=? AND valid=1",
		[]interface{}{jti},
		[]interface{}{&valid})
	if err != nil {
		ctx.Error("vbdb.JwtIsBlacklistedCtx",
			zap.String("jti", jti),
			zap.Error(err))
		return false, false
	}

	// If the JWT exists ether increase iuc (invalid-usage-count) or vuc
	// (valid-usage-count)
	if exists {
		column := vbcore.TernaryOperatorA(valid == 0, "iuc", "vuc")
		err = s.Exec(fmt.Sprintf("UPDATE jwts SET %s = %s + 1 WHERE jti=?", column, column), jti)
		if err != nil {
			ctx.Error("jti exists but increasing iuc or vuc failed", zap.Error(err))
			return false, false
		}
	}

	ctx.Debug("vbdb.JwtIsBlacklistedCtx",
		zap.String("jti", jti),
		zap.Bool("blacklisted", valid == 0))
	return valid == 0, true
}

// JwtIsBlacklisted is the same as `JwtIsBlacklistedCtx` but uses the
// `defaultCtx` as logger.
func JwtIsBlacklisted(jti string) (blacklisted bool, success bool) {
	return JwtIsBlacklistedCtx(jti, defaultCtx)
}

// JwtAddCtx adds the passed `jti` (JWT ID), `exp` and `userID` into the
// `jwts` table as valid entries. Can later be modified to invalid
// (e.g. blacklisted)
func JwtAddCtx(jti string, exp time.Time, userID int, iat time.Time, ip string, ctx *zap.Logger) (success bool) {
	err := s.Exec("INSERT INTO jwts (jti, exp, user_id, iat, ip) VALUES(?, ?, ?, ?, ?)", jti, exp, userID, iat, ip)
	if err != nil {
		ctx.Error("vbdb.JwtAdd",
			zap.String("jti", jti),
			zap.Time("exp", exp),
			zap.Int("user_id", userID),
			zap.Time("iat", iat),
			zap.String("ip", ip),
			zap.Error(err))
		return false
	}
	return true
}

// JwtAdd is the same as `JwtAddCtx` but uses the `defaultCtx` as logger.
func JwtAdd(jti string, exp time.Time, userID int, iat time.Time, ip string) (success bool) {
	return JwtAddCtx(jti, exp, userID, iat, ip, defaultCtx)
}
