package vbdb

import (
	"github.com/go-sql-driver/mysql"
	"github.com/vikebot/vbcore"
	"go.uber.org/zap"
)

// ActiveRoundsCtx loads all rounds which have not status `vbcore.RoundStatusFinished`.
func ActiveRoundsCtx(ctx *zap.Logger) (rounds []vbcore.Round, success bool) {
	rounds = []vbcore.Round{}

	var id, joined, min, max, roundstatus int
	var name, wallpaper string
	var starttime mysql.NullTime
	err := s.SelectRange(`
		SELECT round.id,
			round.name,
			round.wallpaper,
			(SELECT COUNT(id) FROM roundentry WHERE roundentry.round_id = round.id) AS "joined",
			roundsize.min,
			roundsize.max,
			round.starttime,
			round.roundstatus_id
		FROM round, roundsize
		WHERE round.roundstatus_id IN (?, ?, ?)
		ORDER BY round.id ASC`,
		[]interface{}{vbcore.RoundStatusOpen, vbcore.RoundStatusClosed, vbcore.RoundStatusRunning},
		[]interface{}{&id, &name, &wallpaper, &joined, &min, &max, &starttime, &roundstatus},
		func() {
			r := vbcore.Round{
				ID:          id,
				Name:        name,
				Wallpaper:   wallpaper,
				Joined:      joined,
				Min:         min,
				Max:         max,
				Starttime:   starttime.Time,
				RoundStatus: roundstatus,
			}
			rounds = append(rounds, r)
		})
	if err != nil {
		ctx.Error("vbdb.ActiveRoundsCtx", zap.Error(err))
		return nil, false
	}

	return rounds, true
}

// ActiveRounds is the same as `ActiveRoundsCtx` but uses the `defaultCtx` as
// logger.
func ActiveRounds() (lobbies []vbcore.Round, success bool) {
	return ActiveRoundsCtx(defaultCtx)
}

// JoinedUsersCtx returns the `userID`s of all users which joined the round
// specified through the roundID.
func JoinedUsersCtx(roundID int, ctx *zap.Logger) (joined []int, success bool) {
	users := []int{}

	var userID int
	err := s.SelectRange("SELECT user_id FROM roundentry WHERE round_id=?",
		[]interface{}{roundID},
		[]interface{}{&userID},
		func() {
			users = append(users, userID)
		})
	if err != nil {
		ctx.Error("vbdb.JoinedUsersCtx",
			zap.Int("roundID", roundID),
			zap.Error(err))
		return nil, false
	}

	return users, true
}

// JoinedUsers is the same as `JoinedUsersCtx` but uses the `defaultCtx` as
// logger.
func JoinedUsers(roundID int) (joined []int, success bool) {
	return JoinedUsersCtx(roundID, defaultCtx)
}

// JoinRoundCtx lets the user join the specified round. The first return value
// `alreadyJoined` indicates if the user has already joined the team and the
// call is redundent.
func JoinRoundCtx(userID, roundID int, ctx *zap.Logger) (alreadyJoined, success bool) {
	exists, err := s.MysqlExists("SELECT id FROM roundentry WHERE user_id=? AND round_id=?", userID, roundID)
	if err != nil {
		ctx.Error("vbdb.JoinRoundCtx - authtoken gen failed",
			zap.Int("user_id", userID),
			zap.Int("round_id", roundID),
			zap.Error(err))
		return false, false
	}
	if exists {
		return true, true
	}

	authtoken, err := vbcore.CryptoGenString(18)
	if err != nil {
		ctx.Error("vbdb.JoinRoundCtx - authtoken gen failed",
			zap.Int("user_id", userID),
			zap.Int("round_id", roundID),
			zap.Error(err))
		return false, false
	}

	roundticket, err := vbcore.CryptoGenString(16)
	if err != nil {
		ctx.Error("vbdb.JoinRoundCtx - roundticket gen failed",
			zap.Int("user_id", userID),
			zap.Int("round_id", roundID),
			zap.Error(err))
		return false, false
	}

	watchtoken, err := vbcore.CryptoGenString(12)
	if err != nil {
		ctx.Error("vbdb.JoinRoundCtx - watchtoken gen failed",
			zap.Int("user_id", userID),
			zap.Int("round_id", roundID),
			zap.Error(err))
		return false, false
	}

	key, err := vbcore.CryptoGen()
	if err != nil {
		ctx.Error("vbdb.JoinRoundCtx - aeskey gen failed",
			zap.Int("user_id", userID),
			zap.Int("round_id", roundID),
			zap.Error(err))
		return false, false
	}

	err = s.Exec("INSERT INTO roundentry (authtoken, roundticket, watchtoken, user_id, round_id, aeskey) VALUES(?, ?, ?, ?, ?, ?)", authtoken, roundticket, watchtoken, userID, roundID, key)
	if err != nil {
		ctx.Error("vbdb.JoinRoundCtx - roundentry insert failed",
			zap.Int("user_id", userID),
			zap.Int("round_id", roundID),
			zap.Error(err))
		return false, false
	}

	return false, true
}

// JoinRound is the same as `JoinRoundCtx` but uses the `defaultCtx` as logger.
func JoinRound(userID, roundID int) (alreadyJoined, success bool) {
	return JoinRoundCtx(userID, roundID, defaultCtx)
}

// RoundExistsCtx checks whether a round specified by it's ID exists or not
func RoundExistsCtx(roundID int, ctx *zap.Logger) (exists, success bool) {
	exists, err := s.MysqlExists("SELECT id FROM round WHERE id=?", roundID)
	if err != nil {
		ctx.Error("vbdb.RoundExistsCtx",
			zap.Int("round_id", roundID),
			zap.Error(err))
		return false, false
	}
	return exists, true
}

// RoundExists is the same as `RoundExistsCtx` but uses the `defaultCtx` as
// logger.
func RoundExists(roundID int) (exists, success bool) {
	return RoundExistsCtx(roundID, defaultCtx)
}
