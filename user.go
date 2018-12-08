package vbdb

import (
	"database/sql"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/vikebot/vbcore"
	"go.uber.org/zap"
)

// CountUsersCtx adds up the amount of all registered users
func CountUsersCtx(ctx *zap.Logger) int {
	count, err := s.UnsafeMysqlCount("user", "id")
	if err != nil {
		ctx.Error("vbdb.CountUsersCtx", zap.Error(err))
		return 0
	}
	ctx.Debug("resp: vbdb.CountUsersCtx", zap.Int("count", count))
	return count
}

// CountUsers is the same as `CountUsersCtx` but uses the `defaultCtx` as
// logger.
func CountUsers() int {
	return CountUsersCtx(defaultCtx)
}

// UserIDFromRegcodeCtx exchanges a registration code for a `userID` and a
// `finished` state (indicating the state of the registration).
func UserIDFromRegcodeCtx(code string, ctx *zap.Logger) (userID int, finished bool, success bool) {
	var done int
	err := s.Select(
		"SELECT user_id, done FROM user_register WHERE code=?",
		[]interface{}{code},
		[]interface{}{&userID, &done})
	if err != nil {
		ctx.Error("vbdb.UserIDFromRegCodeCtx",
			zap.String("code", code),
			zap.Error(err))
		return 0, false, false
	}

	ctx.Debug("resp: vbdb.UserIDFromRegCodeCtx",
		zap.String("code", code),
		zap.Int("user_id", userID),
		zap.Bool("finished", done != 0))
	return userID, done != 0, true
}

// UserIDFromRegcode is the same as `UserIDFromRegcodeCtx` but uses the
// `defaultCtx` as logger.
func UserIDFromRegcode(code string) (userID int, finished bool, success bool) {
	return UserIDFromRegcodeCtx(code, defaultCtx)
}

// UserIDFromUsernameCtx tries to find the user which currently uses th passed
// username. Comparisons are made with lower-case strings
func UserIDFromUsernameCtx(username string, ctx *zap.Logger) (userID int, exists bool, success bool) {
	username = strings.ToLower(username)

	exists, err := s.SelectExists("SELECT user_id FROM user_username WHERE username=? AND active=1",
		[]interface{}{username},
		[]interface{}{&userID})
	if err != nil {
		ctx.Error("vbdb.UserIDFromUsernameCtx",
			zap.String("username", username),
			zap.Error(err))
		return 0, false, false
	}

	ctx.Debug("resp: vbdb.UserIDFromUsernameCtx",
		zap.String("username", username),
		zap.Int("user_id", userID),
		zap.Bool("exists", exists))
	return userID, exists, true
}

// UserIDFromUsername is the same as `UserIDFromUsernameCtx` but uses the
// `defaultCtx` as logger.
func UserIDFromUsername(username string) (userID int, exists bool, success bool) {
	return UserIDFromUsernameCtx(username, defaultCtx)
}

// RegcodeFromUserIDCtx exchanged a `userID` to it's associated registration
// code and a `finished` state (indicating the state of the registration).
func RegcodeFromUserIDCtx(userID int, ctx *zap.Logger) (code string, finished bool, success bool) {
	var done int
	err := s.Select(
		"SELECT code, done FROM user_register WHERE user_id=?",
		[]interface{}{userID},
		[]interface{}{&code, &done})
	if err != nil {
		ctx.Error("vbdb.RegcodeFromUserIDCtx",
			zap.Int("userID", userID),
			zap.Error(err))
		return "", false, false
	}

	ctx.Debug("resp: vbdb.RegcodeFromUserIDCtx",
		zap.Int("user_id", userID),
		zap.String("code", code),
		zap.Bool("finished", done != 0))
	return code, done != 0, true
}

// RegcodeFromUserID is the same as `RegcodeFromUserIDCtx` but uses the
// `defaultCtx` as logger.
func RegcodeFromUserID(userID int) (code string, finished bool, success bool) {
	return RegcodeFromUserIDCtx(userID, defaultCtx)
}

// UpdateUserCtx compares a newUser with an oldUser and updates the differences
// into the database. The msg parameter is used to safe some kind of log
// message how this update come about. This func uses SQL-transactions, hence
// the complete update process only failes or succeeds - nothing in between.
func UpdateUserCtx(newUser *vbcore.User, oldUser *vbcore.SafeUser, msg string, ctx *zap.Logger) (success bool) {
	ctx.Debug("req: vbdb.UpdateUserCtx", zap.Int("user_id", oldUser.ID))

	// Open a new sql transaction
	tx, err := db.Begin()
	if err != nil {
		ctx.Error("vbdb.UpdateUserCtx - db.Begin", zap.Error(err))
		return false
	}

	// Create a local anonymous rollback function that roolbacks all changes if
	// possible and returns the default values to the calling method
	rollback := func(err error) bool {
		ctx.Error("vbdb.UpdateUserCtx", zap.Error(err))
		rlbErr := tx.Rollback()
		if rlbErr != nil {
			ctx.Error("vbdb.UpdateUserCtx - tx.Rollback", zap.Error(rlbErr))
		}
		return false
	}

	msgID, err := s.ExecTxID(tx, "INSERT INTO msg(message) VALUES(?)", msg)
	if err != nil {
		return rollback(err)
	}

	valueChanged := func(field *string, value string) bool {
		// Do checks in one line -> short-circuit evaluation
		return (field != nil && *field != value)
	}
	if valueChanged(newUser.Username, oldUser.Username) {
		ctx.Debug("user_username",
			zap.String("new", *newUser.Username),
			zap.String("old", oldUser.Username))

		// Set old username to inactive
		err = s.ExecTx(tx, "UPDATE user_username SET active=0 WHERE user_id=? AND username=?", oldUser.ID, oldUser.Username)
		if err != nil {
			return rollback(err)
		}
		// Insert new username
		err = s.ExecTx(tx, "INSERT INTO user_username(user_id, msg_id, username) VALUES(?, ?, ?)", oldUser.ID, msgID, *newUser.Username)
		if err != nil {
			return rollback(err)
		}
	}
	if valueChanged(newUser.Name, oldUser.Name) {
		ctx.Debug("user_name",
			zap.String("new", *newUser.Name),
			zap.String("old", oldUser.Name))
		err = s.ExecTx(tx, "INSERT INTO user_name(user_id, msg_id, name) VALUES(?, ?, ?)", oldUser.ID, msgID, *newUser.Name)
		if err != nil {
			return rollback(err)
		}
	}
	if valueChanged(newUser.Bio, oldUser.Bio) {
		ctx.Debug("user_bio",
			zap.String("new", *newUser.Bio),
			zap.String("old", oldUser.Bio))
		err = s.ExecTx(tx, "INSERT INTO user_bio(user_id, msg_id, bio) VALUES(?, ?, ?)", oldUser.ID, msgID, *newUser.Bio)
		if err != nil {
			return rollback(err)
		}
	}
	if valueChanged(newUser.Location, oldUser.Location) {
		ctx.Debug("user_location",
			zap.String("new", *newUser.Location),
			zap.String("old", oldUser.Location))
		err = s.ExecTx(tx, "INSERT INTO user_location(user_id, msg_id, location) VALUES(?, ?, ?)", oldUser.ID, msgID, *newUser.Location)
		if err != nil {
			return rollback(err)
		}
	}
	if valueChanged(newUser.Company, oldUser.Company) {
		ctx.Debug("user_company",
			zap.String("new", *newUser.Company),
			zap.String("old", oldUser.Company))
		err = s.ExecTx(tx, "INSERT INTO user_company(user_id, msg_id, company) VALUES(?, ?, ?)", oldUser.ID, msgID, *newUser.Company)
		if err != nil {
			return rollback(err)
		}
	}

	// Safe all changes -> if commit failes log error and return default values
	err = tx.Commit()
	if err != nil {
		return rollback(err)
	}

	return true
}

// UpdateUser is the same as `UpdateUserCtx` but uses the `defaultCtx` as
// logger.
func UpdateUser(newUser *vbcore.User, oldUser *vbcore.SafeUser, msg string) (success bool) {
	return UpdateUserCtx(newUser, oldUser, msg, defaultCtx)
}

// UsernameFromIDCtx loads the username associated with the given `userID`. Returns
// "" (empty string) if the user isn't found.
func UsernameFromIDCtx(userID int, ctx *zap.Logger) (username string, success bool) {
	exists, err := s.SelectExists(
		`SELECT username 
		FROM user_username
		WHERE user_id=?`,
		[]interface{}{userID},
		[]interface{}{&username},
	)
	if err != nil {
		ctx.Error("vbdb.UsernameFromIDCtx",
			zap.Int("user_id", userID),
			zap.Error(err))
		return "", false
	}

	if !exists {
		return "", false
	}

	return username, true
}

// UsernamesFromRoundIDCtx loads the usernames associated with the given `roundID`. Returns
// an empty slice if the roundID isn't valid.
func UsernamesFromRoundIDCtx(roundID int, ctx *zap.Logger) (usernames map[int]string, success bool) {
	usernames = make(map[int]string)

	var userID int
	var username string

	err := s.SelectRange(
		`SELECT user_username.user_id, user_username.username FROM user_username
		JOIN roundentry ON roundentry.user_id=user_username.user_id
		JOIN round r on roundentry.round_id=r.id
		WHERE r.id=?`,
		[]interface{}{roundID},
		[]interface{}{&userID, &username},
		func() {
			usernames[userID] = username
		})

	if err != nil {
		ctx.Error("vbdb.UsernamesFromRoundIDCtx",
			zap.Int("round_id", roundID),
			zap.Error(err))
		return usernames, false
	}

	return usernames, true
}

// UserFromIDCtx loads the user associated with the given `userID`. Returns
// `nil` if the user isn't found.
func UserFromIDCtx(userID int, ctx *zap.Logger) (user *vbcore.SafeUser, success bool) {
	var permission int
	var username, name, bio, location, company sql.NullString
	exists, err := s.SelectExists(
		`SELECT permission, username, name, bio, location, company
		FROM view_user
		WHERE id=?`,
		[]interface{}{userID},
		[]interface{}{&permission, &username, &name, &bio, &location, &company})
	if err != nil {
		ctx.Error("vbdb.UserFromIDCtx",
			zap.Int("user_id", userID),
			zap.Error(err))
		return nil, false
	}
	if !exists {
		return nil, true
	}

	email := []vbcore.Email{}
	var address string
	var status, public int
	err = s.SelectRange("SELECT email, status, public FROM user_email WHERE user_id=? AND deleted=0 ORDER BY id ASC",
		[]interface{}{userID},
		[]interface{}{&address, &status, &public},
		func() {
			email = append(email, vbcore.Email{
				Email:  address,
				Status: status,
				Public: public == 1,
			})
		})
	if err != nil {
		ctx.Error("vbdb.UserFromIDCtx",
			zap.Int("user_id", userID),
			zap.Error(err))
		return nil, false
	}

	web := []string{}
	var webrow string
	err = s.SelectRange("SELECT web FROM user_web WHERE user_id=? AND deleted=0 ORDER BY id ASC",
		[]interface{}{userID},
		[]interface{}{&webrow},
		func() {
			web = append(web, webrow)
		})
	if err != nil {
		ctx.Error("vbdb.UserFromIDCtx",
			zap.Int("user_id", userID),
			zap.Error(err))
		return nil, false
	}

	social := map[string]string{}
	var platform, link string
	err = s.SelectRange("SELECT platform, link FROM user_social WHERE user_id=? AND deleted=0 ORDER BY id ASC",
		[]interface{}{userID},
		[]interface{}{&platform, &link},
		func() {
			social[platform] = link
		})
	if err != nil {
		ctx.Error("vbdb.UserFromIDCtx",
			zap.Int("user_id", userID),
			zap.Error(err))
		return nil, false
	}

	return &vbcore.SafeUser{
		ID:               userID,
		Permission:       permission,
		PermissionString: vbcore.PermissionItoA(permission),
		Username:         vbcore.TernaryOperatorA(username.Valid, username.String, ""),
		Name:             vbcore.TernaryOperatorA(name.Valid, name.String, ""),
		Emails:           email,
		Bio:              vbcore.TernaryOperatorA(bio.Valid, bio.String, ""),
		Location:         vbcore.TernaryOperatorA(location.Valid, location.String, ""),
		Web:              web,
		Company:          vbcore.TernaryOperatorA(company.Valid, company.String, ""),
		Social:           social,
	}, true
}

// UserFromID is the same as `UserFromIDCtx` but uses the `defaultCtx` as
// logger.
func UserFromID(userID int) (user *vbcore.SafeUser, success bool) {
	return UserFromIDCtx(userID, defaultCtx)
}

// UsernameFromID returns a username as string from an given ID
func UsernameFromID(userID int) (username string, success bool) {
	return UsernameFromIDCtx(userID, defaultCtx)
}

// UsernamesFromRoundID returns usernames as a slice of string from an given
// roundID
func UsernamesFromRoundID(roundID int) (usernames map[int]string, success bool) {
	return UsernamesFromRoundIDCtx(roundID, defaultCtx)
}

// UserFromUsernameCtx loads the user associated with the given `username`.
// Returns `nil` if the user isn't found.
func UserFromUsernameCtx(username string, ctx *zap.Logger) (user *vbcore.SafeUser, success bool) {
	userID, exists, success := UserIDFromUsernameCtx(username, ctx)
	if !success {
		return nil, false
	}
	if !exists {
		return nil, true
	}
	return UserFromIDCtx(userID, ctx)
}

// UserFromUsername is the same as `UserFromUsernameCtx` but uses the
// `defaultCtx` as logger.
func UserFromUsername(username string, ctx *zap.Logger) (user *vbcore.SafeUser, success bool) {
	return UserFromUsernameCtx(username, ctx)
}

// RegisterUserCtx registers a new user and returns the user's ID, the msgID
// used for all inserts and a registration code (which the user must use in
// order to register later).
func RegisterUserCtx(user vbcore.User, msg string, ctx *zap.Logger) (userID int, regCode string, success bool) {
	ctx.Debug("req: vbdb.RegisterUserCtx")

	// Open a new sql transaction
	tx, err := db.Begin()
	if err != nil {
		ctx.Error("vbdb.RegisterUserCtx", zap.Error(err))
		return
	}

	// Create a local anonymous rollback function that roolbacks all changes if
	// possible and returns the default values to the calling method
	rollback := func(err error) (int, string, bool) {
		ctx.Error("vbdb.RegisterUserCtx", zap.Error(err))
		rlbErr := tx.Rollback()
		if rlbErr != nil {
			ctx.Error("vbdb.RegisterUserCtx - tx.Rollback", zap.Error(rlbErr))
		}
		return 0, "", false
	}

	// Create return parameters: msgID, userID and generate a new regcode
	msgID, err := s.ExecTxID(tx, "INSERT INTO msg(message) VALUES(?)", msg)
	if err != nil {
		return rollback(err)
	}
	tmpID, err := s.ExecTxID(tx, "INSERT INTO user (msg_id) VALUES(?)", msgID)
	if err != nil {
		return rollback(err)
	}
	userID = int(tmpID)
	regCode, err = vbcore.CryptoGenString(32)
	if err != nil {
		return rollback(err)
	}

	// Create a list of inserts that must be handled
	inserts := make(map[string][]interface{})
	inserts["INSERT INTO user_permission (user_id, msg_id) VALUES(?, ?)"] = []interface{}{userID, msgID}
	inserts["INSERT INTO user_register (user_id, msg_id, code) VALUES(?, ?, ?)"] = []interface{}{userID, msgID, regCode}
	if user.Username != nil && len(*user.Username) > 0 {
		inserts["INSERT INTO user_username (user_id, msg_id, username) VALUES(?, ?, ?)"] = []interface{}{userID, msgID, user.Username}
	}
	if user.Name != nil && len(*user.Name) > 0 {
		inserts["INSERT INTO user_name (user_id, msg_id, name) VALUES(?, ?, ?)"] = []interface{}{userID, msgID, user.Name}
	}
	if user.Bio != nil && len(*user.Bio) > 0 {
		inserts["INSERT INTO user_bio (user_id, msg_id, bio) VALUES(?, ?, ?)"] = []interface{}{userID, msgID, user.Bio}
	}
	if user.Location != nil && len(*user.Location) > 0 {
		inserts["INSERT INTO user_location (user_id, msg_id, location) VALUES(?, ?, ?)"] = []interface{}{userID, msgID, user.Location}
	}
	if user.Company != nil && len(*user.Company) > 0 {
		inserts["INSERT INTO user_company (user_id, msg_id, company) VALUES(?, ?, ?)"] = []interface{}{userID, msgID, user.Company}
	}
	for k, v := range inserts {
		err = s.ExecTx(tx, k, v...)
		if err != nil {
			return rollback(err)
		}
	}

	for _, v := range user.Emails {
		if len(v.Email) == 0 {
			continue
		}
		err = s.ExecTx(tx, "INSERT INTO user_email (user_id, msg_id, email, status, public) VALUES(?, ?, ?, ?, ?)", userID, msgID, v.Email, v.Status, vbcore.TernaryOperatorI(v.Public, 1, 0))
		if err != nil {
			return rollback(err)
		}
	}
	for _, v := range user.Web {
		if len(v) == 0 {
			continue
		}
		err = s.ExecTx(tx, "INSERT INTO user_web (user_id, msg_id, web) VALUES(?, ?, ?)", userID, msgID, v)
		if err != nil {
			return rollback(err)
		}
	}
	for k, v := range user.Social {
		if len(v) == 0 {
			continue
		}
		err = s.ExecTx(tx, "INSERT INTO user_social (user_id, msg_id, platform, link) VALUES(?, ?, ?, ?)", userID, msgID, k, v)
		if err != nil {
			return rollback(err)
		}
	}
	for k, v := range user.OAuth {
		err = s.ExecTx(tx, "INSERT INTO oauth (user_id, msg_id, provider, id) VALUES(?, ?, ?, ?)", userID, msgID, k, v)
		if err != nil {
			return rollback(err)
		}
	}

	// Safe all changes till now. if commit failes log error and
	// return default values
	err = tx.Commit()
	if err != nil {
		return rollback(err)
	}

	return userID, regCode, true
}

// RegisterUser is the same as `RegisterUserCtx` but uses the `defaultCtx` as
// logger.
func RegisterUser(user vbcore.User, msg string) (userID int, regCode string, success bool) {
	return RegisterUserCtx(user, msg, defaultCtx)
}

// UserPermissionCtx loads the current permission of the specified user from
// the database. It is mandatory that the user corresponding to the passed ID
// does exists!
func UserPermissionCtx(userID int, ctx *zap.Logger) (permission int, success bool) {
	exists, err := s.SelectExists("SELECT permission FROM user_permission WHERE user_id=? ORDER BY id DESC LIMIT 1",
		[]interface{}{userID},
		[]interface{}{&permission})
	if err != nil || !exists {
		ctx.Error("vbdb.UserPermissionCtx",
			zap.Int("user_id", userID),
			zap.Error(err))
		return 0, false
	}

	ctx.Debug("resp: vbdb.UserPermissionCtx", zap.Int("permission", permission))
	return permission, true
}

// UserPermission is the same as `UserPermissionCtx` but uses the `defaultCtx`
// as logger.
func UserPermission(userID int) (permission int, success bool) {
	return UserPermissionCtx(userID, defaultCtx)
}

func UpdateUserEmailStatusCtx(userID int, email string, status int, ctx *zap.Logger) (success bool) {
	err := s.Exec("UPDATE user_email SET status=? WHERE user_id=? AND email=?", status, userID, email)
	if err != nil {
		ctx.Error("vbdb.UpdateUserEmailStatusCtx",
			zap.Int("user_id", userID),
			zap.String("email", email),
			zap.Int("status", status),
			zap.Error(err))
		return false
	}
	return true
}

func UpdateUserEmailStatus(userID int, email string, status int) (success bool) {
	return UpdateUserEmailStatusCtx(userID, email, status, defaultCtx)
}

func UserEmailVerificationLoadCtx(userID int, email string, ctx *zap.Logger) (lastSent *time.Time, valid bool, succes bool) {
	var last mysql.NullTime

	err := s.Select("SELECT verification_last FROM user_email WHERE user_id=? AND email=?", []interface{}{userID, email}, []interface{}{&last})
	if err != nil {
		ctx.Error("vbdb.UserEmailVerificationLoadCtx",
			zap.Int("user_id", userID),
			zap.String("email", email),
			zap.Error(err))
		return nil, false, false
	}

	return &last.Time, last.Valid, true
}

func UserEmailVerificationSetCtx(userID int, email string, verificationCode string, ctx *zap.Logger) (success bool) {
	err := s.Exec("UPDATE user_email SET verification_code=?, verification_last=? WHERE user_id=? AND email=?", verificationCode, time.Now().UTC(), userID, email)
	if err != nil {
		ctx.Error("vbdb.UserEmailVerificationSetCtx",
			zap.Int("user_id", userID),
			zap.String("email", email),
			zap.String("verification_code", vbcore.StrMask(verificationCode)),
			zap.Error(err))
		return false
	}

	return true
}

func UserEmailVerificationIsCtx(userID int, email string, verificationCode string, ctx *zap.Logger) (verified bool, success bool) {
	exists, err := s.MysqlExists("SELECT id FROM user_email WHERE user_id=? AND email=? AND verification_code=?", userID, email, verificationCode)
	if err != nil {
		ctx.Error("vbdb.UserEmailVerificationIsCtx",
			zap.Int("user_id", userID),
			zap.String("email", email),
			zap.String("verification_code", vbcore.StrMask(verificationCode)),
			zap.Error(err))
		return false, false
	}

	return exists, true
}

func UserDeleteWebExpectCtx(userID int, web []string, ctx *zap.Logger) (success bool) {
	if len(web) == 0 {
		return true
	}

	// Generate a string with questionsmarks, so that the amount of question=
	// marks equals the amount of entries in our web list
	var questionMarks string
	for i := 0; i < len(web); i++ {
		questionMarks += "?,"
	}
	questionMarks = questionMarks[:len(questionMarks)-1]

	params := []interface{}{userID}
	for _, i := range web {
		params = append(params, i)
	}
	err := s.Exec("UPDATE user_web SET deleted=1 WHERE user_id=? AND web NOT IN("+questionMarks+")", params...)
	if err != nil {
		ctx.Error("vbdb.UserDeleteWebExpect",
			zap.Int("user_id", userID),
			zap.Strings("web", web),
			zap.Error(err))
		return false
	}

	return true
}

func UserDeleteSocialExpectCtx(userID int, social []string, ctx *zap.Logger) (success bool) {
	if len(social) == 0 {
		return true
	}

	// Generate a string with questionsmarks, so that the amount of question=
	// marks equals the amount of entries in our web list
	var questionMarks string
	for i := 0; i < len(social); i++ {
		questionMarks += "?,"
	}
	questionMarks = questionMarks[:len(questionMarks)-1]

	params := []interface{}{userID}
	for _, i := range social {
		params = append(params, i)
	}
	err := s.Exec("UPDATE user_social SET deleted=1 WHERE user_id=? AND platform NOT IN("+questionMarks+")", params...)
	if err != nil {
		ctx.Error("vbdb.UserDeleteSocialExpectCtx",
			zap.Int("user_id", userID),
			zap.Strings("social_platforms", social),
			zap.Error(err))
		return false
	}

	return true
}

func UserSetRegistrationDoneCtx(userID int, ctx *zap.Logger) (success bool) {
	err := s.Exec("UPDATE user_register SET done=1 WHERE user_id=?", userID)
	if err != nil {
		ctx.Error("vbdb.UserSetRegistrationDoneCtx",
			zap.Int("user_id", userID),
			zap.Error(err))
		return false
	}

	return true
}
