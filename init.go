package vbdb

import (
	"database/sql"
	"fmt"

	"github.com/harwoeck/sqle"

	"github.com/vikebot/vbcore"
	"go.uber.org/zap"

	// Driver import for MariaDB
	_ "github.com/go-sql-driver/mysql"
)

var (
	db         *sql.DB
	s          *sqle.Sqle
	defaultCtx *zap.Logger
	conf       *Config
)

// Config collects all relevant informations/credentials to access a vikebot
// database
type Config struct {
	DbAddr *vbcore.Endpoint
	DbUser string
	DbPass string
	DbName string
}

// Init initializes all internally used structures of vbdb. E.g. a database
// connection pool is created and connections get established. If any errors
// occur durring initialization the error will be returned. The passed zap
// instance is saved and used as default throughout the package if no explicit
// logging-context is provided
func Init(config *Config, logCtx *zap.Logger) (err error) {
	conf = config
	defaultCtx = logCtx

	// Open sql connection
	db, err = sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&collation=utf8mb4_unicode_ci", conf.DbUser, conf.DbPass, conf.DbAddr, conf.DbName))
	if err != nil {
		return err
	}
	err = db.Ping()
	if err != nil {
		return err
	}
	defaultCtx.Info("db connection established and ready to use")

	// create sqle
	s = sqle.New(db)
	defaultCtx.Info("sqle instance created and ready to use")

	return nil
}
