package app

import (
	"github.com/jmoiron/sqlx"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/utils"

	"scroll-tech/database"
	"scroll-tech/database/migrate"
)

func getConfig(ctx *cli.Context) (*database.DBConfig, error) {
	file := ctx.String(utils.ConfigFileFlag.Name)
	dbCfg, err := database.NewConfig(file)
	if err != nil {
		return nil, err
	}
	return dbCfg, nil
}

func getMigrationConfig(ctx *cli.Context) string {
	path := ctx.String(utils.MigrationDirFlag.Name)
	return path
}

func initDB(dbCfg *database.DBConfig) (*sqlx.DB, error) {
	factory, err := database.NewOrmFactory(dbCfg)
	if err != nil {
		return nil, err
	}
	log.Debug("Got db config from env", "driver name", dbCfg.DriverName, "dsn", dbCfg.DSN)

	return factory.GetDB(), nil
}

// resetDB clean or reset database.
func resetDB(ctx *cli.Context) error {
	cfg, err := getConfig(ctx)
	if err != nil {
		return err
	}
	db, err := initDB(cfg)
	if err != nil {
		return err
	}
	var version int64
	err = migrate.Rollback(db.DB, &version, getMigrationConfig(ctx))
	if err != nil {
		return err
	}
	log.Info("successful to reset", "init version", version)

	return nil
}

// checkDBStatus check db status
func checkDBStatus(ctx *cli.Context) error {
	cfg, err := getConfig(ctx)
	if err != nil {
		return err
	}
	db, err := initDB(cfg)
	if err != nil {
		return err
	}

	return migrate.Status(db.DB, getMigrationConfig(ctx))
}

// dbVersion return the latest version
func dbVersion(ctx *cli.Context) error {
	cfg, err := getConfig(ctx)
	if err != nil {
		return err
	}
	db, err := initDB(cfg)
	if err != nil {
		return err
	}

	version, err := migrate.Current(db.DB)
	log.Info("show database version", "db version", version)

	return err
}

// migrateDB migrate db
func migrateDB(ctx *cli.Context) error {
	cfg, err := getConfig(ctx)
	if err != nil {
		return err
	}
	db, err := initDB(cfg)
	if err != nil {
		return err
	}
	return migrate.Migrate(db.DB, getMigrationConfig(ctx))
}

// rollbackDB rollback db by version
func rollbackDB(ctx *cli.Context) error {
	cfg, err := getConfig(ctx)
	if err != nil {
		return err
	}
	db, err := initDB(cfg)
	if err != nil {
		return err
	}
	version := ctx.Int64("version")
	return migrate.Rollback(db.DB, &version, getMigrationConfig(ctx))
}
