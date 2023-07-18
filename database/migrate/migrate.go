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

func init() {
	goose.SetBaseFS(embedMigrations)
	goose.SetSequential(true)
	goose.SetTableName("scroll_migrations")

	verbose, _ := strconv.ParseBool(os.Getenv("LOG_SQL_MIGRATIONS"))
	goose.SetVerbose(verbose)
}

// Migrate migrate db
func Migrate(db *sql.DB, path string) error {
	//return goose.Up(db, MIGRATIONS_DIR, goose.WithAllowMissing())
	return goose.Up(db, path, goose.WithAllowMissing())
}

// Rollback rollback to the given version
func Rollback(db *sql.DB, version *int64, path string) error {
	if version != nil {
		return goose.DownTo(db, path, *version)
	}
	return goose.Down(db, path)
}

// ResetDB clean and migrate db.
func ResetDB(db *sql.DB, path string) error {
	if err := Rollback(db, new(int64), path); err != nil {
		return err
	}
	return Migrate(db, path)
}

// Current get current version
func Current(db *sql.DB) (int64, error) {
	return goose.GetDBVersion(db)
}

// Status is normal or not
func Status(db *sql.DB, path string) error {
	return goose.Version(db, path)
}

// Create a new migration folder
func Create(db *sql.DB, name, migrationType string, path string) error {
	return goose.Create(db, path, name, migrationType)
}
