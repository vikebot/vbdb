package vbdb

import (
	"go.uber.org/zap"
)

// CountGamesCtx adds up all existing roundentries. Example: There are three
// rounds in the database and all were joined by 20 players. Therefore `3 x
// 20 = 60` will be returned.
func CountGamesCtx(ctx *zap.Logger) int {
	count, err := s.UnsafeMysqlCount("roundentry", "id")
	if err != nil {
		ctx.Error("vbdb.CountGamesCtx", zap.Error(err))
		return 0
	}
	ctx.Debug("resp: vbdb.CountGamesCtx", zap.Int("count", count))
	return count
}

// CountGames is the same as `CountGamesCtx` but uses the `defaultCtx` as
// logger.
func CountGames() int {
	return CountGamesCtx(defaultCtx)
}
