package migrations

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Migration represents a database migration
type Migration struct {
	Version     string
	Description string
	SQL         string
}

// Migrator handles database migrations
type Migrator struct {
	db            *sql.DB
	migrationsDir string
}

// NewMigrator creates a new migrator
func NewMigrator(db *sql.DB, migrationsDir string) *Migrator {
	return &Migrator{
		db:            db,
		migrationsDir: migrationsDir,
	}
}

// Initialize creates the migrations table if it doesn't exist
func (m *Migrator) Initialize() error {
	_, err := m.db.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			id INTEGER PRIMARY KEY,
			version TEXT NOT NULL,
			description TEXT NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}

// GetAppliedMigrations returns a map of already applied migrations
func (m *Migrator) GetAppliedMigrations() (map[string]bool, error) {
	rows, err := m.db.Query("SELECT version FROM migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, rows.Err()
}

// LoadMigrations loads all migration files from the migrations directory
func (m *Migrator) LoadMigrations() ([]Migration, error) {
	files, err := ioutil.ReadDir(m.migrationsDir)
	if err != nil {
		return nil, err
	}

	var migrations []Migration
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		filePath := filepath.Join(m.migrationsDir, file.Name())
		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			return nil, err
		}

		// Parse version and description from filename (e.g., "001_initial_schema.sql")
		parts := strings.SplitN(strings.TrimSuffix(file.Name(), ".sql"), "_", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid migration filename: %s", file.Name())
		}

		migrations = append(migrations, Migration{
			Version:     parts[0],
			Description: strings.ReplaceAll(parts[1], "_", " "),
			SQL:         string(content),
		})
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// ApplyMigration applies a single migration
func (m *Migrator) ApplyMigration(migration Migration) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}

	// Apply the migration
	_, err = tx.Exec(migration.SQL)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error applying migration %s: %w", migration.Version, err)
	}

	// Record the migration
	_, err = tx.Exec(
		"INSERT INTO migrations (version, description) VALUES (?, ?)",
		migration.Version,
		migration.Description,
	)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error recording migration %s: %w", migration.Version, err)
	}

	return tx.Commit()
}

// MigrateUp applies all pending migrations
func (m *Migrator) MigrateUp() error {
	// Initialize migrations table
	if err := m.Initialize(); err != nil {
		return err
	}

	// Get applied migrations
	applied, err := m.GetAppliedMigrations()
	if err != nil {
		return err
	}

	// Load migrations
	migrations, err := m.LoadMigrations()
	if err != nil {
		return err
	}

	// Apply pending migrations
	for _, migration := range migrations {
		if applied[migration.Version] {
			log.Printf("Migration %s already applied, skipping", migration.Version)
			continue
		}

		log.Printf("Applying migration %s: %s", migration.Version, migration.Description)
		if err := m.ApplyMigration(migration); err != nil {
			return err
		}
		log.Printf("Migration %s applied successfully", migration.Version)
	}

	return nil
}

// CreateMigration creates a new migration file
func (m *Migrator) CreateMigration(description string) (string, error) {
	// Get the next version number
	migrations, err := m.LoadMigrations()
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	nextVersion := "001"
	if len(migrations) > 0 {
		version := fmt.Sprintf("%03d", len(migrations)+1)
		nextVersion = version
	}

	// Create migrations directory if it doesn't exist
	if err := os.MkdirAll(m.migrationsDir, 0755); err != nil {
		return "", err
	}

	// Create migration file
	fileName := fmt.Sprintf("%s_%s.sql", nextVersion, strings.ReplaceAll(description, " ", "_"))
	filePath := filepath.Join(m.migrationsDir, fileName)

	// Create empty file with a comment
	content := fmt.Sprintf("-- Migration: %s\n-- Created: %s\n\n", description, time.Now().Format(time.RFC3339))
	if err := ioutil.WriteFile(filePath, []byte(content), 0644); err != nil {
		return "", err
	}

	return filePath, nil
}
