package main

import (
	"github.com/jmoiron/sqlx"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/spf13/viper"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/utils"

	"scroll-tech/database"
	"scroll-tech/database/migrate"
)

func initDB(file string) (*sqlx.DB, error) {
	viper.SetConfigFile(file)
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	viper.SetDefault("max_open_num", 200)
	viper.SetDefault("max_idle_num", 20)
	factory, err := database.NewOrmFactory(viper.GetViper())
	if err != nil {
		return nil, err
	}
	driverName := viper.GetString("driver_name")
	dsn := viper.GetString("dsn")
	log.Debug("Got db config from env", "driver name", driverName, "dsn", dsn)
	return factory.GetDB(), nil
}

// resetDB clean or reset database.
func resetDB(ctx *cli.Context) error {
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

// checkDBStatus check db status
func checkDBStatus(ctx *cli.Context) error {
	db, err := initDB(ctx.String(utils.ConfigFileFlag.Name))
	if err != nil {
		return err
	}

	return migrate.Status(db.DB)
}

// dbVersion return the latest version
func dbVersion(ctx *cli.Context) error {
	db, err := initDB(ctx.String(utils.ConfigFileFlag.Name))
	if err != nil {
		return err
	}

	version, err := migrate.Current(db.DB)
	log.Info("show database version", "db version", version)

	return err
}

// migrateDB migrate db
func migrateDB(ctx *cli.Context) error {
	db, err := initDB(ctx.String(utils.ConfigFileFlag.Name))
	if err != nil {
		return err
	}

	return migrate.Migrate(db.DB)
}

// rollbackDB rollback db by version
func rollbackDB(ctx *cli.Context) error {
	db, err := initDB(ctx.String(utils.ConfigFileFlag.Name))
	if err != nil {
		return err
	}
	version := ctx.Int64("version")
	return migrate.Rollback(db.DB, &version)
}
