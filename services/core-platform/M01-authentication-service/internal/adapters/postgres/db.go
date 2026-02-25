package postgres

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Connect opens and validates a Postgres-backed GORM connection pool.
// Pool parameters are set centrally to keep service behavior predictable across runtimes.
func Connect(ctx context.Context, databaseURL string, maxConns int32) (*gorm.DB, error) {
	slog.Default().InfoContext(ctx, "postgres connect started",
		"module", "postgres",
		"layer", "adapter",
		"operation", "connect",
		"outcome", "start",
	)
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		PrepareStmt:    true,
		TranslateError: true,
	})
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("gorm sql db: %w", err)
	}
	if maxConns > 0 {
		sqlDB.SetMaxOpenConns(int(maxConns))
		sqlDB.SetMaxIdleConns(int(maxConns) / 2)
	}
	sqlDB.SetConnMaxIdleTime(15 * time.Minute)
	sqlDB.SetConnMaxLifetime(time.Hour)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	slog.Default().InfoContext(ctx, "postgres connect completed",
		"module", "postgres",
		"layer", "adapter",
		"operation", "connect",
		"outcome", "success",
	)
	return db, nil
}

// RunMigrations applies embedded SQL migrations in lexical order.
// Embedding migrations with the binary avoids drift between code and schema at startup.
func RunMigrations(ctx context.Context, db *gorm.DB) error {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	slog.Default().InfoContext(ctx, "postgres migrations started",
		"module", "postgres",
		"layer", "adapter",
		"operation", "run_migrations",
		"outcome", "start",
		"migration_count", len(names),
	)

	for _, name := range names {
		raw, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if err := db.WithContext(ctx).Exec(string(raw)).Error; err != nil {
			return fmt.Errorf("exec migration %s: %w", name, err)
		}
		slog.Default().InfoContext(ctx, "migration applied",
			"module", "postgres",
			"layer", "adapter",
			"operation", "apply_migration",
			"outcome", "success",
			"migration", name,
		)
	}
	slog.Default().InfoContext(ctx, "postgres migrations completed",
		"module", "postgres",
		"layer", "adapter",
		"operation", "run_migrations",
		"outcome", "success",
		"migration_count", len(names),
	)
	return nil
}
