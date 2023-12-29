package migrate

import (
	"database/sql"
	"embed"
	"os"
	"strconv"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

// MigrationsDir migration dir
const MigrationsDir string = "migrations"

func init() {
	goose.SetBaseFS(embedMigrations)
	goose.SetSequential(true)
	goose.SetTableName("bridge_historyv2_migrations")

	verbose, _ := strconv.ParseBool(os.Getenv("LOG_SQL_MIGRATIONS"))
	goose.SetVerbose(verbose)
}

// Migrate migrate db
func Migrate(db *sql.DB) error {
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
