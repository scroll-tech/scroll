package main

import (
	"github.com/jmoiron/sqlx"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/utils"
	"scroll-tech/database"
	"scroll-tech/database/migrate"

	"scroll-tech/bridge/config"
)

func initDB(file string) (*sqlx.DB, error) {
	cfg, err := config.NewConfig(file)
	if err != nil {
		return nil, err
	}
	dbCfg := cfg.DBConfig
	factory, err := database.NewOrmFactory(dbCfg)
	if err != nil {
		return nil, err
	}
	log.Debug("Got db config from env", "driver name", dbCfg.DriverName, "dsn", dbCfg.DSN)

	return factory.GetDB(), nil
}

// ResetDB clean or reset database.
func ResetDB(ctx *cli.Context) error {
	db, err := initDB(ctx.String(utils.ConfigFileFlag.Name))
	if err != nil {
		return err
	}

	var version int64
	err = migrate.Rollback(db.DB, &version)
	if err != nil {
		return err
	}
	log.Info("successful to reset", "init version", version)

	return nil
}

// CheckDBStatus check db status
func CheckDBStatus(ctx *cli.Context) error {
	db, err := initDB(ctx.String(utils.ConfigFileFlag.Name))
	if err != nil {
		return err
	}

	return migrate.Status(db.DB)
}

// DBVersion return the latest version
func DBVersion(ctx *cli.Context) error {
	db, err := initDB(ctx.String(utils.ConfigFileFlag.Name))
	if err != nil {
		return err
	}

	version, err := migrate.Current(db.DB)
	log.Info("show database version", "db version", version)

	return err
}

// MigrateDB migrate db
func MigrateDB(ctx *cli.Context) error {
	db, err := initDB(ctx.String(utils.ConfigFileFlag.Name))
	if err != nil {
		return err
	}

	return migrate.Migrate(db.DB)
}

// RollbackDB rollback db by version
func RollbackDB(ctx *cli.Context) error {
	db, err := initDB(ctx.String(utils.ConfigFileFlag.Name))
	if err != nil {
		return err
	}
	version := ctx.Int64("version")
	return migrate.Rollback(db.DB, &version)
}
