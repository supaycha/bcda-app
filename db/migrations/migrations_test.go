package migrations

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jinzhu/gorm"

	"github.com/stretchr/testify/assert"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/stretchr/testify/suite"
)

// These tests relies on migrate tool being installed
// See: https://github.com/golang-migrate/migrate/tree/v4.13.0/cmd/migrate
type MigrationTestSuite struct {
	suite.Suite

	db *sql.DB

	bcdaDB    string
	bcdaDBURL string

	bcdaQueueDB    string
	bcdaQueueDBURL string
}

func (s *MigrationTestSuite) SetupSuite() {
	// We expect that the DB URL follows
	// postgres://<USER_NAME>:<PASSWORD>@<HOST>:<PORT>/<DB_NAME>
	re := regexp.MustCompile(`(postgresql\:\/\/\S+\:\S+\@\S+\:\d+\/)(.*)(\?.*)`)

	s.db = database.GetDbConnection()

	databaseURL := os.Getenv("DATABASE_URL")
	s.bcdaDB = fmt.Sprintf("migrate_test_bcda_%d", time.Now().Nanosecond())
	s.bcdaQueueDB = fmt.Sprintf("migrate_test_bcda_queue_%d", time.Now().Nanosecond())
	fmt.Printf("'%s'\n", databaseURL)
	s.bcdaDBURL = re.ReplaceAllString(databaseURL, fmt.Sprintf("${1}%s${3}", s.bcdaDB))
	s.bcdaQueueDBURL = re.ReplaceAllString(databaseURL, fmt.Sprintf("${1}%s${3}", s.bcdaQueueDB))

	if _, err := s.db.Exec("CREATE DATABASE " + s.bcdaDB); err != nil {
		assert.FailNowf(s.T(), "Could not create bcda db", err.Error())
	}

	if _, err := s.db.Exec("CREATE DATABASE " + s.bcdaQueueDB); err != nil {
		assert.FailNowf(s.T(), "Could not create bcda_queue db", err.Error())
	}
}

func (s *MigrationTestSuite) TearDownSuite() {
	if _, err := s.db.Exec("DROP DATABASE " + s.bcdaDB); err != nil {
		assert.FailNowf(s.T(), "Could not drop bcda db", err.Error())
	}

	if _, err := s.db.Exec("DROP DATABASE " + s.bcdaQueueDB); err != nil {
		assert.FailNowf(s.T(), "Could not drop bcda_queue db", err.Error())
	}
}

func TestMigrationTestSuite(t *testing.T) {
	suite.Run(t, new(MigrationTestSuite))
}

func (s *MigrationTestSuite) TestBCDAMigration() {
	migrator := migrator{
		migrationPath: "./bcda/",
		dbURL:         s.bcdaDBURL,
	}
	fmt.Println(s.bcdaDBURL)
	db, err := gorm.Open("postgres", s.bcdaDBURL)
	if err != nil {
		assert.FailNowf(s.T(), "Failed to open postgres connection", err.Error())
	}
	defer db.Close()

	migration1Tables := []string{"acos", "cclf_beneficiaries", "cclf_beneficiary_xrefs",
		"cclf_files", "job_keys", "jobs", "suppression_files", "suppressions"}

	// Tests should begin with "up" migrations, in order, followed by "down" migrations in reverse order
	tests := []struct {
		name  string
		tFunc func(t *testing.T)
	}{
		{
			"Apply initial schema",
			func(t *testing.T) {
				migrator.runMigration(t, "1")
				for _, table := range migration1Tables {
					assert.True(t, db.HasTable(table), fmt.Sprintf("Table %s should exist", table))
				}
			},
		},
		{
			"Revert initial schema",
			func(t *testing.T) {
				migrator.runMigration(t, "0")
				for _, table := range migration1Tables {
					assert.False(t, db.HasTable(table), fmt.Sprintf("Table %s should not exist", table))
				}
			},
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, tt.tFunc)
	}
}

func (s *MigrationTestSuite) TestBCDAQueueMigration() {
	migrator := migrator{
		migrationPath: "./bcda_queue/",
		dbURL:         s.bcdaQueueDBURL,
	}
	fmt.Println(s.bcdaDBURL)
	db, err := gorm.Open("postgres", s.bcdaQueueDBURL)
	if err != nil {
		assert.FailNowf(s.T(), "Failed to open postgres connection", err.Error())
	}
	defer db.Close()

	migration1Tables := []string{"que_jobs"}

	// Tests should begin with "up" migrations, in order, followed by "down" migrations in reverse order
	tests := []struct {
		name  string
		tFunc func(t *testing.T)
	}{
		{
			"Apply initial schema",
			func(t *testing.T) {
				migrator.runMigration(t, "1")
				for _, table := range migration1Tables {
					assert.True(t, db.HasTable(table), fmt.Sprintf("Table %s should exist", table))
				}
			},
		},
		{
			"Revert initial schema",
			func(t *testing.T) {
				migrator.runMigration(t, "0")
				for _, table := range migration1Tables {
					assert.False(t, db.HasTable(table), fmt.Sprintf("Table %s should not exist", table))
				}
			},
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, tt.tFunc)
	}
}

type migrator struct {
	migrationPath string
	dbURL         string
}

func (m migrator) runMigration(t *testing.T, idx string) {
	args := []string{"goto", idx}
	expVersion := idx
	// Since we do not have a 0 index, this is interpreted
	// as revert the last migration (1)
	if idx == "0" {
		args = []string{"down", "1"}
	}

	args = append([]string{"-database", m.dbURL, "-path",
		m.migrationPath}, args...)

	_, err := exec.Command("migrate", args...).CombinedOutput()
	if err != nil {
		t.Errorf("Failed to run migration %s", err.Error())
	}

	// If we're going down past the first schema, we won't be able
	// to check the version since there's no active schema version
	if idx == "0" {
		return
	}

	// Expected output:
	// <VERSION>
	// If there's a failure (i.e. dirty migration)
	// <VERSION> (dirty)
	out, err := exec.Command("migrate", "-database", m.dbURL, "-path",
		m.migrationPath, "version").CombinedOutput()
	if err != nil {
		t.Errorf("Failed to retrieve version information %s", err.Error())
	}
	str := strings.TrimSpace(string(out))

	assert.Contains(t, expVersion, str)
	assert.NotContains(t, str, "dirty")
}