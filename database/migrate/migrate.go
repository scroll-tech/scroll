package migrate

import (
	"database/sql"
	"embed"
	"os"
	"strconv"

	"github.com/pressly/goose/v3"
)

//go:embed migrations
var embedMigrations embed.FS

// MigrationsDir migration dir
const MigrationsDir string = "migrations"

func init() {
	// note goose ignore ono-sql files by default so we do not need to specify *.sql
	goose.SetBaseFS(embedMigrations)
	goose.SetSequential(true)
	goose.SetTableName("scroll_migrations")

	verbose, _ := strconv.ParseBool(os.Getenv("LOG_SQL_MIGRATIONS"))
	goose.SetVerbose(verbose)
}

// MigrateModule migrate db used by other module with specified goose TableName
// sql file for that module must be put as a sub-directory under `MigrationsDir`
func MigrateModule(db *sql.DB, moduleName string) error {

	goose.SetTableName(moduleName + "_migrations")
	defer func() {
		goose.SetTableName("scroll_migrations")
	}()

	return goose.Up(db, MigrationsDir+"/"+moduleName, goose.WithAllowMissing())
}

// RollbackModule rollback the specified module to the given version
func RollbackModule(db *sql.DB, moduleName string, version *int64) error {

	goose.SetTableName(moduleName + "_migrations")
	defer func() {
		goose.SetTableName("scroll_migrations")
	}()
	moduleDir := MigrationsDir + "/" + moduleName

	if version != nil {
		return goose.DownTo(db, moduleDir, *version)
	}
	return goose.Down(db, moduleDir)
}

// ResetModuleDB clean and migrate db for a module.
func ResetModuleDB(db *sql.DB, moduleName string) error {
	if err := RollbackModule(db, moduleName, new(int64)); err != nil {
		return err
	}
	return MigrateModule(db, moduleName)
}

// Migrate migrate db
func Migrate(db *sql.DB) error {
	//return goose.Up(db, MIGRATIONS_DIR, goose.WithAllowMissing())
	return goose.Up(db, MigrationsDir, goose.WithAllowMissing())
}

// Rollback rollback to the given version
func Rollback(db *sql.DB, version *int64) error {
	if version != nil {
		return goose.DownTo(db, MigrationsDir, *version)
	}
	return goose.Down(db, MigrationsDir)
}

// ResetDB clean and migrate db.
func ResetDB(db *sql.DB) error {
	if err := Rollback(db, new(int64)); err != nil {
		return err
	}
	return Migrate(db)
}

// Current get current version
func Current(db *sql.DB) (int64, error) {
	return goose.GetDBVersion(db)
}

// Status is normal or not
func Status(db *sql.DB) error {
	return goose.Version(db, MigrationsDir)
}

// Create a new migration folder
func Create(db *sql.DB, name, migrationType string) error {
	return goose.Create(db, MigrationsDir, name, migrationType)
}
