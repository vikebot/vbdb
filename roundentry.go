package vbdb

import (
	"github.com/go-sql-driver/mysql"
	"github.com/vikebot/vbcore"
	"go.uber.org/zap"
)

// RoundentryFromWatchtokenCtx verifies that a user passed `watchtoken` exists
// in the database. If so informations about the specific `roundentry` are
// returned so further verifications can be performed.
func RoundentryFromWatchtokenCtx(watchtoken string, ctx *zap.Logger) (verification *vbcore.RoundentryVerification, exists bool, success bool) {
	var userID, roundID int
	exists, err := s.SelectExists(
		"SELECT user_id, round_id FROM roundentry WHERE watchtoken=?",
		[]interface{}{watchtoken},
		[]interface{}{&userID, &roundID})
	if err != nil {
		ctx.Error("vbdb.RoundentryFromWatchtokenCtx",
			zap.String("watchtoken", watchtoken),
			zap.Error(err))
		return nil, false, false
	}
	if !exists {
		return nil, false, true
	}
	return &vbcore.RoundentryVerification{
		UserID:  userID,
		RoundID: roundID,
	}, true, true
}

// RoundentryFromWatchtoken is the same as `RoundentryFromWatchtokenCtx` but
// uses the `defaultCtx` as logger.
func RoundentryFromWatchtoken(watchtoken string) (verification *vbcore.RoundentryVerification, exists bool, success bool) {
	return RoundentryFromWatchtokenCtx(watchtoken, defaultCtx)
}

// RoundentryFromRoundticketCtx verifies that a user passed `roundticket`
// exists in the database. If so informations about the specific `roundentry`
// are returned so further verifications can be performed.
func RoundentryFromRoundticketCtx(roundticket string, ctx *zap.Logger) (verification *vbcore.RoundentryVerification, exists bool, success bool) {
	var userID, roundID int
	var aeskey string
	exists, err := s.SelectExists(
		"SELECT user_id, round_id, aeskey FROM roundentry WHERE roundticket=?",
		[]interface{}{roundticket},
		[]interface{}{&userID, &roundID, &aeskey})
	if err != nil {
		ctx.Error("vbdb.RoundentryFromRoundticketCtx",
			zap.String("roundticket", roundticket),
			zap.Error(err))
		return nil, false, false
	}
	if !exists {
		return nil, false, true
	}
	return &vbcore.RoundentryVerification{
		UserID:  userID,
		RoundID: roundID,
		AESKey:  &aeskey,
	}, true, true
}

// RoundentryFromRoundticket is the same as `RoundentryFromRoundticketCtx` but
// uses the `defaultCtx` as logger.
func RoundentryFromRoundticket(roundticket string) (verification *vbcore.RoundentryVerification, exists bool, success bool) {
	return RoundentryFromRoundticketCtx(roundticket, defaultCtx)
}

// RoundentryConnectinfoCtx returns everything needed to connect to the `vbgs`
// instance hosting the game associated with the `authtoken`.
func RoundentryConnectinfoCtx(authtoken string, ctx *zap.Logger) (connectinfo *vbcore.RoundentryConnectinfo, exists bool, success bool) {
	var roundticket, aeskey, ipv4, ipv6 string
	var port int
	exists, err := s.SelectExists(`
		SELECT roundticket, aeskey, server.ipv4, server.ipv6, server.port
		FROM roundentry
			JOIN round ON roundentry.round_id = round.id
			JOIN server ON round.server_id = server.id
		WHERE authtoken=?`,
		[]interface{}{authtoken}, []interface{}{&roundticket, &aeskey, &ipv4, &ipv6, &port})
	if err != nil {
		ctx.Error("vbdb.RoundentryConnectinfoCtx",
			zap.Error(err),
			zap.String("authtoken", authtoken))
		return nil, false, false
	}
	if !exists {
		return nil, false, true
	}
	return &vbcore.RoundentryConnectinfo{
		Roundticket: roundticket,
		AESKey:      aeskey,
		IPv4:        ipv4,
		IPv6:        ipv6,
		Port:        port,
	}, true, true
}

// RoundentryConnectinfo is the same as `RoundentryConnectinfoCtx` but uses
// the `defaultCtx` as logger.
func RoundentryConnectinfo(authtoken string) (connectinfo *vbcore.RoundentryConnectinfo, exists bool, success bool) {
	return RoundentryConnectinfoCtx(authtoken, defaultCtx)
}

// ActiveRoundentriesCtx returns all Roundentries from a user, for games which
// are currently of status `vbcore.RoundStatusOpen`, `vbcore.RoundStatusClosed`
// or `vbcore.RoundStatusRunning`.
// Doesn't returns Roundentries for already finished games (e.g.
// `vbcore.RoundStatusFinished`).
func ActiveRoundentriesCtx(userID int, ctx *zap.Logger) (roundentries []vbcore.Roundentry, success bool) {
	roundentries = []vbcore.Roundentry{}

	var id, joined, min, max, roundstatus int
	var name, wallpaper, authtoken, watchtoken string
	var starttime mysql.NullTime
	err := s.SelectRange(`
		SELECT r.id,
			r.name,
			r.wallpaper,
			(SELECT COUNT(id) FROM roundentry sqre WHERE sqre.round_id = r.id) AS "joined",
			rs.min,
			rs.max,
			r.starttime,
			r.roundstatus_id,
			re.authtoken,
			re.watchtoken
		FROM roundentry re
			JOIN round r ON re.round_id=r.id
			JOIN roundsize rs ON r.roundsize_id=rs.id
		WHERE re.user_id = ? AND
			r.roundstatus_id IN (?, ?, ?)
		ORDER BY r.id ASC`,
		[]interface{}{userID, vbcore.RoundStatusOpen, vbcore.RoundStatusClosed, vbcore.RoundStatusRunning},
		[]interface{}{&id, &name, &wallpaper, &joined, &min, &max, &starttime, &roundstatus, &authtoken, &watchtoken},
		func() {
			r := vbcore.Roundentry{
				Round: vbcore.Round{
					ID:          id,
					Name:        name,
					Wallpaper:   wallpaper,
					Joined:      joined,
					Min:         min,
					Max:         max,
					Starttime:   starttime.Time,
					RoundStatus: roundstatus,
				},
				Authtoken:  authtoken,
				Watchtoken: watchtoken,
			}
			roundentries = append(roundentries, r)
		})
	if err != nil {
		ctx.Error("vbdb.ActiveRoundentriesCtx", zap.Error(err))
		return nil, false
	}

	return roundentries, true
}

// ActiveRoundentries is the same as `ActiveRoundentriesCtx` but uses
// the `defaultCtx` as logger.
func ActiveRoundentries(userID int) (roundentries []vbcore.Roundentry, success bool) {
	return ActiveRoundentriesCtx(userID, defaultCtx)
}

// WebsocketAddressFromWatchtokenCtx resolves a watchtoken to it's associated
// gameserver's websocket address.
func WebsocketAddressFromWatchtokenCtx(watchtoken string, ctx *zap.Logger) (websocket string, exists, success bool) {
	exists, err := s.SelectExists(`
		SELECT CONCAT(s.ipv4, ':', s.wsproxy) AS websocket
		FROM roundentry re
			JOIN round r ON re.round_id=r.id
			JOIN server s ON r.server_id=s.id
		WHERE re.watchtoken=?
		`,
		[]interface{}{watchtoken},
		[]interface{}{&websocket})
	if err != nil {
		ctx.Error("vbdb.WebsocketAddressFromWatchtokenCtx",
			zap.String("watchtoken", watchtoken),
			zap.Error(err))
		return "", false, false
	}
	if !exists {
		return "", false, true
	}

	return websocket, true, true
}
