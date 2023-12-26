package migrate

import (
    "database/sql"
    "embed"
    "log"
    "os"
    "strconv"

    "github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
// embedMigrations includes all SQL migration files.
var embedMigrations embed.FS

// MigrationsDir is the directory for database migration files.
const MigrationsDir = "migrations"

func init() {
    goose.SetBaseFS(embedMigrations)
    goose.SetSequential(true)
    goose.SetTableName("scroll_migrations")

    verbose, err := strconv.ParseBool(os.Getenv("LOG_SQL_MIGRATIONS"))
    if err != nil {
        log.Printf("Warning: Failed to parse LOG_SQL_MIGRATIONS: %v", err)
    }
    goose.SetVerbose(verbose)
}

// Migrate applies database migrations.
func Migrate(db *sql.DB) error {
    return goose.Up(db, MigrationsDir, goose.WithAllowMissing())
}

// Rollback reverts the database to the given version.
func Rollback(db *sql.DB, version *int64) error {
    if version != nil {
        return goose.DownTo(db, MigrationsDir, *version)
    }
    return goose.Down(db, MigrationsDir)
}

// ResetDB cleans and migrates the database.
func ResetDB(db *sql.DB) error {
    if err := Rollback(db, new(int64)); err != nil {
        return err
    }
    return Migrate(db)
}

// Current returns the current database version.
func Current(db *sql.DB) (int64, error) {
    return goose.GetDBVersion(db)
}

// Status checks if the database is up-to-date with migrations.
func Status(db *sql.DB) error {
    return goose.Version(db, MigrationsDir)
}

// Create generates a new migration file.
func Create(db *sql.DB, name, migrationType string) error {
    return goose.Create(db, MigrationsDir, name, migrationType)
}
