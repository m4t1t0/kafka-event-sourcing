package database

import (
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var fs embed.FS

// RunMigrations applies all pending migrations using the given PostgreSQL DSN.
// The dsn should be a GORM-style DSN; it is converted to a postgres:// URL internally.
func RunMigrations(dsn string) error {
	source, err := iofs.New(fs, "migrations")
	if err != nil {
		return fmt.Errorf("migration source: %w", err)
	}

	pgURL, err := dsnToURL(dsn)
	if err != nil {
		return fmt.Errorf("parse dsn: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, pgURL)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}

	return nil
}

// dsnToURL converts a GORM-style key=value DSN to a postgres:// URL.
func dsnToURL(dsn string) (string, error) {
	params := make(map[string]string)
	for _, part := range splitDSN(dsn) {
		if len(part) == 0 {
			continue
		}
		kv := splitKV(part)
		if len(kv) == 2 {
			params[kv[0]] = kv[1]
		}
	}

	host := params["host"]
	port := params["port"]
	user := params["user"]
	password := params["password"]
	dbname := params["dbname"]
	sslmode := params["sslmode"]

	if host == "" || user == "" || dbname == "" {
		return "", fmt.Errorf("dsn must contain host, user, and dbname")
	}
	if port == "" {
		port = "5432"
	}
	if sslmode == "" {
		sslmode = "disable"
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, password, host, port, dbname, sslmode), nil
}

func splitDSN(dsn string) []string {
	var parts []string
	current := ""
	for _, c := range dsn {
		if c == ' ' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func splitKV(s string) []string {
	for i, c := range s {
		if c == '=' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}
