package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/fadedpez/tucoramirez/pkg/db/migrations"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Define command-line flags
	createCmd := flag.NewFlagSet("create", flag.ExitOnError)
	migrateCmd := flag.NewFlagSet("migrate", flag.ExitOnError)

	// Create command options
	migrationsDir := createCmd.String("dir", "migrations", "Directory to store migrations")

	// Migrate command options
	dbPath := migrateCmd.String("db", "data/game.db", "Path to SQLite database")
	migrateDir := migrateCmd.String("dir", "migrations", "Directory containing migrations")

	// Show usage if no arguments provided
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Parse command
	switch os.Args[1] {
	case "create":
		createCmd.Parse(os.Args[2:])
		if createCmd.NArg() < 1 {
			fmt.Println("Error: Missing migration description")
			createCmd.Usage()
			os.Exit(1)
		}
		description := createCmd.Arg(0)
		createNewMigration(*migrationsDir, description)

	case "migrate":
		migrateCmd.Parse(os.Args[2:])
		applyMigrations(*dbPath, *migrateDir)

	case "help":
		printUsage()

	default:
		fmt.Printf("Error: Unknown command '%s'\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  go run cmd/migration/main.go create DESCRIPTION  - Create a new migration")
	fmt.Println("  go run cmd/migration/main.go migrate            - Apply pending migrations")
	fmt.Println("  go run cmd/migration/main.go help              - Show this help")
	fmt.Println("\nExamples:")
	fmt.Println("  go run cmd/migration/main.go create \"add wallet tables\"")
	fmt.Println("  go run cmd/migration/main.go migrate")
}

func createNewMigration(migrationsDir, description string) {
	// Create a temporary database connection to use the migrator
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Create migrator
	migrator := migrations.NewMigrator(db, migrationsDir)

	// Create migration file
	filePath, err := migrator.CreateMigration(description)
	if err != nil {
		log.Fatalf("Error creating migration: %v", err)
	}

	// Add helpful SQLite examples to the migration file
	addSQLiteExamples(filePath)

	fmt.Printf("Created migration file: %s\n", filePath)
	fmt.Println("Edit this file to add your database schema changes.")
}

func addSQLiteExamples(filePath string) {
	// Read existing content
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Error reading migration file: %v", err)
	}

	// Add SQLite examples
	examples := `
-- SQLite Examples:

-- Create a new table
-- CREATE TABLE IF NOT EXISTS table_name (
--   id INTEGER PRIMARY KEY AUTOINCREMENT,
--   name TEXT NOT NULL,
--   value INTEGER DEFAULT 0,
--   created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
-- );

-- Add a column to existing table
-- ALTER TABLE table_name ADD COLUMN new_column TEXT;

-- Create an index
-- CREATE INDEX IF NOT EXISTS idx_table_column ON table_name(column_name);

-- Your migration SQL goes below this line:

`

	// Combine existing content with examples
	newContent := string(content) + examples

	// Write back to file
	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		log.Fatalf("Error writing to migration file: %v", err)
	}
}

func applyMigrations(dbPath, migrationsDir string) {
	// Ensure database directory exists
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatalf("Error creating database directory: %v", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Create migrator
	migrator := migrations.NewMigrator(db, migrationsDir)

	// Apply migrations
	if err := migrator.MigrateUp(); err != nil {
		log.Fatalf("Error applying migrations: %v", err)
	}

	fmt.Println("Migrations applied successfully!")
}
