package reperdb

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

// Repository provides access to known reper values.
type Repository interface {
	Contains(reper string) bool
	Count() int
}

// SQLiteRepository loads reper values from a SQLite database.
type SQLiteRepository struct {
	reper map[string]struct{}
}

// Open opens SQLite database and loads all known reper values.
func Open(path string) (*SQLiteRepository, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite database: %w", err)
	}

	reper, err := loadReper(db)
	if err != nil {
		return nil, err
	}

	return &SQLiteRepository{
		reper: reper,
	}, nil
}

func (r *SQLiteRepository) Contains(value string) bool {
	_, ok := r.reper[value]
	return ok
}

func (r *SQLiteRepository) Count() int {
	return len(r.reper)
}

func loadReper(db *sql.DB) (map[string]struct{}, error) {
	tables, err := findParamsTables(db)
	if err != nil {
		return nil, err
	}

	result := make(map[string]struct{})

	for _, table := range tables {
		values, err := loadTableReper(db, table)
		if err != nil {
			return nil, fmt.Errorf(
				"load reper from table %q: %w",
				table,
				err,
			)
		}

		for _, value := range values {
			result[value] = struct{}{}
		}
	}

	return result, nil
}

func findParamsTables(db *sql.DB) ([]string, error) {
	const query = `
		SELECT name
		FROM sqlite_master
		WHERE type = 'table'
		  AND name LIKE '%\_params' ESCAPE '\'
		ORDER BY name
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("find *_params tables: %w", err)
	}
	defer rows.Close()

	var result []string

	for rows.Next() {
		var name string

		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan table name: %w", err)
		}

		result = append(result, name)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate table names: %w", err)
	}

	return result, nil
}

func loadTableReper(db *sql.DB, table string) ([]string, error) {
	query := fmt.Sprintf(
		`SELECT raper FROM %s WHERE raper IS NOT NULL`,
		quoteIdentifier(table),
	)

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []string

	for rows.Next() {
		var value string

		if err := rows.Scan(&value); err != nil {
			return nil, fmt.Errorf("scan raper: %w", err)
		}

		value = strings.TrimSpace(value)

		if value != "" {
			result = append(result, value)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func quoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
