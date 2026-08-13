package reperdb_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/rusl222/scada/reperdb"
)

func TestOpen_LoadsReperFromAllParamsTables(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "sqlite.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}

	createTestDatabase(t, db)

	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	repo, err := reperdb.Open(dbPath)
	if err != nil {
		t.Fatalf("reperdb.Open() error = %v", err)
	}

	tests := []struct {
		name  string
		reper string
		want  bool
	}{
		{
			name:  "first table",
			reper: "ABC",
			want:  true,
		},
		{
			name:  "second table",
			reper: "DEF",
			want:  true,
		},
		{
			name:  "unicode",
			reper: "1TF QСТ КУШ",
			want:  true,
		},
		{
			name:  "unknown",
			reper: "UNKNOWN",
			want:  false,
		},
		{
			name:  "non params table",
			reper: "SHOULD_NOT_BE_LOADED",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := repo.Contains(tt.reper)

			if got != tt.want {
				t.Errorf(
					"Contains(%q) = %v, want %v",
					tt.reper,
					got,
					tt.want,
				)
			}
		})
	}

	if got, want := repo.Count(), 3; got != want {
		t.Errorf("Count() = %d, want %d", got, want)
	}
}

func createTestDatabase(t *testing.T, db *sql.DB) {
	t.Helper()

	statements := []string{
		`
		CREATE TABLE foo_params (
			id INTEGER PRIMARY KEY,
			raper TEXT
		)
		`,
		`
		CREATE TABLE bar_params (
			id INTEGER PRIMARY KEY,
			raper TEXT
		)
		`,
		`
		CREATE TABLE notparams (
			id INTEGER PRIMARY KEY,
			raper TEXT
		)
		`,
		`
		INSERT INTO foo_params (raper)
		VALUES
			('ABC'),
			('1TF QСТ КУШ'),
			(NULL)
		`,
		`
		INSERT INTO bar_params (raper)
		VALUES
			('DEF')
		`,
		`
		INSERT INTO notparams (raper)
		VALUES
			('SHOULD_NOT_BE_LOADED')
		`,
	}

	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("exec SQL: %v", err)
		}
	}
}
