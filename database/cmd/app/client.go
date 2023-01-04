package app

import (
	"github.com/jmoiron/sqlx"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/utils"

	"scroll-tech/common/viper"

	"scroll-tech/database"
	"scroll-tech/database/migrate"
)

func initDB(ctx *cli.Context) (*sqlx.DB, error) {
	file := ctx.String(utils.ConfigFileFlag.Name)
	vp, err := viper.NewViper(file, "")
	if err != nil {
		return nil, err
	}
	factory, err := database.NewOrmFactory(vp)
	if err != nil {
		return nil, err
	}
	log.Debug("Got db config from env", "driver name", vp.GetString("driver_name"), "dsn", vp.GetString("dsn"))

	return factory.GetDB(), nil
}

// resetDB clean or reset database.
func resetDB(ctx *cli.Context) error {
	db, err := initDB(ctx)
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
	db, err := initDB(ctx)
	if err != nil {
		return err
	}

	return migrate.Status(db.DB)
}

// dbVersion return the latest version
func dbVersion(ctx *cli.Context) error {
	db, err := initDB(ctx)
	if err != nil {
		return err
	}

	version, err := migrate.Current(db.DB)
	log.Info("show database version", "db version", version)

	return err
}

// migrateDB migrate db
func migrateDB(ctx *cli.Context) error {
	db, err := initDB(ctx)
	if err != nil {
		return err
	}

	return migrate.Migrate(db.DB)
}

// rollbackDB rollback db by version
func rollbackDB(ctx *cli.Context) error {
	db, err := initDB(ctx)
	if err != nil {
		return err
	}
	version := ctx.Int64("version")
	return migrate.Rollback(db.DB, &version)
}
