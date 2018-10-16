package vbdb

import (
	"strconv"
	"strings"

	"github.com/google/go-github/github"
	"github.com/vikebot/vbcore"
	"go.uber.org/zap"
)

// RegisterOauthGoogleCtx maps the `*vbcore.GoogleUser` instance to a
// `vbcore.User` and passes it to `RegisterUserCtx`.
func RegisterOauthGoogleCtx(u *vbcore.GoogleUser, ctx *zap.Logger) (userID int, code string, success bool) {
	username := strings.ToLower(u.GivenName) + "_" + u.Sub[len(u.Sub)/2:]

	user := vbcore.User{
		Username: &username,
		Name:     &u.Name,
		Emails: []vbcore.Email{
			vbcore.Email{
				Email:  u.Email,
				Status: vbcore.TernaryOperatorI(u.EmailVerified, vbcore.EmailVerified, vbcore.EmailLinked),
				Public: false,
			},
		},
		Social: map[string]string{
			"google": u.Profile,
		},
		OAuth: map[string]string{
			vbcore.OAuthProviderGoogle: u.Sub,
		},
	}

	return RegisterUserCtx(user, "CopyGoogleData", ctx)
}

// RegisterOauthGoogle is the same as `RegisterOauthGoogleCtx` but uses the
// `defaultCtx` as logger.
func RegisterOauthGoogle(user *vbcore.GoogleUser) (userID int, code string, success bool) {
	return RegisterOauthGoogleCtx(user, defaultCtx)
}

// RegisterOauthGithubCtx maps the `*github.User` instance to a `vbcore.User`
// and passes it to `RegisterUserCtx`.
func RegisterOauthGithubCtx(u *github.User, ctx *zap.Logger) (userID int, code string, success bool) {
	web := []string{}
	if u.Blog != nil {
		web = append(web, *u.Blog)
	}

	email := []vbcore.Email{}
	if u.Email != nil {
		email = append(email, vbcore.Email{
			Email:  *u.Email,
			Status: vbcore.EmailLinked,
			Public: false,
		})
	}

	user := vbcore.User{
		Username: u.Login,
		Name:     u.Name,
		Emails:   email,
		Bio:      u.Bio,
		Location: u.Location,
		Web:      web,
		Company:  u.Company,
		Social: map[string]string{
			vbcore.OAuthProviderGithub: *u.HTMLURL,
		},
		OAuth: map[string]string{
			vbcore.OAuthProviderGithub: strconv.FormatInt(*u.ID, 10),
		},
	}

	return RegisterUserCtx(user, "CopyGithubData", ctx)
}

// RegisterOauthGithub is the same as `RegisterOauthGithubCtx` but uses the
// `defaultCtx` as logger.
func RegisterOauthGithub(user *github.User) (userID int, code string, success bool) {
	return RegisterOauthGithubCtx(user, defaultCtx)
}
