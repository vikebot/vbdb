package vbdb

import (
	"go.uber.org/zap"
)

// OAuthExistsCtx checks if the `providerID` is already registered with the
// specified `provider`. If so the associated `userID` (from vikebot) is returned.
func OAuthExistsCtx(providerID string, provider string, ctx *zap.Logger) (userID int, exists bool, success bool) {
	exists, err := s.SelectExists("SELECT user_id FROM oauth WHERE id=? AND provider=?",
		[]interface{}{providerID, provider},
		[]interface{}{&userID})
	if err != nil {
		ctx.Error("vbdb.OAuthExistsCtx",
			zap.String("provider_id", providerID),
			zap.String("provider", provider),
			zap.Error(err))
		return 0, false, false
	}
	ctx.Debug("resp: vbdb.OAuthExistsCtx",
		zap.Int("user_id", userID),
		zap.String("provider_id", providerID),
		zap.String("provider", provider),
		zap.Bool("exists", exists))
	return userID, exists, true
}

// OAuthExists is the same as `OAuthExistsCtx` but uses the `defaultCtx` as
// logger.
func OAuthExists(providerID string, provider string) (userID int, exists bool, success bool) {
	return OAuthExistsCtx(providerID, provider, defaultCtx)
}
