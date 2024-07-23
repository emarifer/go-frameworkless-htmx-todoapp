package db

import (
	"database/sql"
	"fmt"
	"log"
	"log/slog"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func getConnection(logger *slog.Logger) (*sql.DB, error) {
	var err error

	// Init SQLite3 database
	db, err = sql.Open("sqlite3", "./app_data.db")
	if err != nil {
		return nil, fmt.Errorf("🔥 failed to connect to the database: %s", err)
	}

	logger.Info("💾 Database Info: connected successfully to DB")

	return db, nil
}

func createMigrations(db *sql.DB) error {
	stmt := `CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email VARCHAR(255) NOT NULL UNIQUE,
		password VARCHAR(255) NOT NULL,
		username VARCHAR(64) NOT NULL
	);`

	_, err := db.Exec(stmt)
	if err != nil {
		return err
	}

	stmt = `CREATE TABLE IF NOT EXISTS todos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_by INTEGER NOT NULL,
		title VARCHAR(64) NOT NULL,
		description VARCHAR(255) NULL,
		status BOOLEAN DEFAULT(FALSE),
		created_at DATETIME default CURRENT_TIMESTAMP,
		FOREIGN KEY(created_by) REFERENCES users(id)
	);`

	_, err = db.Exec(stmt)
	if err != nil {
		return err
	}

	return nil
}

func GetDB(l *slog.Logger) *sql.DB {
	var err error

	if db == nil {
		if db, err = getConnection(l); err != nil {
			log.Fatalf("🔥 failed to connect to the database: %s", err.Error())
		}

		if err = createMigrations(db); err != nil {
			log.Fatalf(
				"🔥 could not create migrations in database: %s", err.Error(),
			)
		}

		return db
	}

	return db
}
