package app

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
	"gorm.io/gorm"

	"bridge-history-api/config"
	"bridge-history-api/orm/migrate"
	"bridge-history-api/utils"
)

func getConfig(ctx *cli.Context) (*config.Config, error) {
	file := ctx.String(utils.ConfigFileFlag.Name)
	dbCfg, err := config.NewConfig(file)
	if err != nil {
		return nil, err
	}
	return dbCfg, nil
}

func initDB(dbCfg *config.DBConfig) (*gorm.DB, error) {
	return utils.InitDB(dbCfg)
}

// resetDB clean or reset database.
func resetDB(ctx *cli.Context) error {
	cfg, err := getConfig(ctx)
	if err != nil {
		return err
	}
	gormDB, err := initDB(cfg.DB)
	if err != nil {
		return err
	}
	db, err := gormDB.DB()
	if err != nil {
		return err
	}
	err = migrate.ResetDB(db)
	if err != nil {
		return err
	}
	log.Info("successful to reset")
	return nil
}

// checkDBStatus check db status
func checkDBStatus(ctx *cli.Context) error {
	cfg, err := getConfig(ctx)
	if err != nil {
		return err
	}
	gormDB, err := initDB(cfg.DB)
	if err != nil {
		return err
	}
	db, err := gormDB.DB()
	if err != nil {
		return err
	}
	return migrate.Status(db)
}

// dbVersion return the latest version
func dbVersion(ctx *cli.Context) error {
	cfg, err := getConfig(ctx)
	if err != nil {
		return err
	}
	gormDB, err := initDB(cfg.DB)
	if err != nil {
		return err
	}
	db, err := gormDB.DB()
	if err != nil {
		return err
	}
	version, err := migrate.Current(db)
	log.Info("show database version", "db version", version)

	return err
}

// migrateDB migrate db
func migrateDB(ctx *cli.Context) error {
	cfg, err := getConfig(ctx)
	if err != nil {
		return err
	}
	gormDB, err := initDB(cfg.DB)
	if err != nil {
		return err
	}
	db, err := gormDB.DB()
	if err != nil {
		return err
	}
	return migrate.Migrate(db)
}

// rollbackDB rollback db by version
func rollbackDB(ctx *cli.Context) error {
	cfg, err := getConfig(ctx)
	if err != nil {
		return err
	}
	gormDB, err := initDB(cfg.DB)
	if err != nil {
		return err
	}
	db, err := gormDB.DB()
	if err != nil {
		return err
	}
	version := ctx.Int64("version")
	return migrate.Rollback(db, &version)
}
