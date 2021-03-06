package postgres

import (
	"database/sql/driver"
	"fmt"
	"math/rand"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/CMSgov/bcda-app/bcda/constants"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/jinzhu/gorm"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/suite"
)

type RepositoryTestSuite struct {
	suite.Suite
}

func TestRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryTestSuite))
}

func (r *RepositoryTestSuite) TestGetLatestCCLFFile() {
	cmsID := "cmsID"
	cclfNum := int(8)
	importStatus := constants.ImportComplete

	tests := []struct {
		name          string
		lowerBound    time.Time
		upperBound    time.Time
		expQueryRegex string
		result        *models.CCLFFile
	}{
		{
			"NoTime",
			time.Time{},
			time.Time{},
			`SELECT * FROM "cclf_files" WHERE "cclf_files"."deleted_at" IS NULL AND ((aco_cms_id = $1 AND cclf_num = $2 AND import_status = $3)) ORDER BY timestamp DESC`,
			getCCLFFile(cclfNum, cmsID, importStatus),
		},
		{
			"LowerBoundTime",
			time.Now(),
			time.Time{},
			`SELECT * FROM "cclf_files" WHERE "cclf_files"."deleted_at" IS NULL AND ((aco_cms_id = $1 AND cclf_num = $2 AND import_status = $3 AND timestamp >= $4)) ORDER BY timestamp DESC`,
			getCCLFFile(cclfNum, cmsID, importStatus),
		},
		{
			"UpperBoundTime",
			time.Time{},
			time.Now(),
			`SELECT * FROM "cclf_files" WHERE "cclf_files"."deleted_at" IS NULL AND ((aco_cms_id = $1 AND cclf_num = $2 AND import_status = $3 AND timestamp <= $4)) ORDER BY timestamp DESC`,
			getCCLFFile(cclfNum, cmsID, importStatus),
		},
		{
			"LowerAndUpperBoundTime",
			time.Now(),
			time.Now(),
			`SELECT * FROM "cclf_files" WHERE "cclf_files"."deleted_at" IS NULL AND ((aco_cms_id = $1 AND cclf_num = $2 AND import_status = $3 AND timestamp >= $4 AND timestamp <= $5)) ORDER BY timestamp DESC`,
			getCCLFFile(cclfNum, cmsID, importStatus),
		},
		{
			"NoResult",
			time.Time{},
			time.Time{},
			`SELECT * FROM "cclf_files" WHERE "cclf_files"."deleted_at" IS NULL AND ((aco_cms_id = $1 AND cclf_num = $2 AND import_status = $3)) ORDER BY timestamp DESC`,
			nil,
		},
	}

	for _, tt := range tests {
		r.T().Run(tt.name, func(t *testing.T) {

			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			gdb, err := gorm.Open("postgres", db)
			if err != nil {
				t.Fatalf("Failed to instantiate gorm db %s", err.Error())
			}

			defer func() {
				err = mock.ExpectationsWereMet()
				assert.NoError(t, err)
				gdb.Close()
				db.Close()
			}()
			repository := NewRepository(gdb)

			args := []driver.Value{cmsID, cclfNum, importStatus}
			if !tt.lowerBound.IsZero() {
				args = append(args, tt.lowerBound)
			}
			if !tt.upperBound.IsZero() {
				args = append(args, tt.upperBound)
			}

			query := mock.ExpectQuery(regexp.QuoteMeta(tt.expQueryRegex)).
				WithArgs(args...)
			if tt.result == nil {
				query.WillReturnError(gorm.ErrRecordNotFound)
			} else {
				query.WillReturnRows(sqlmock.
					NewRows([]string{"id", "cclf_num", "name", "aco_cms_id", "timestamp", "performance_year", "import_status"}).
					AddRow(tt.result.ID, tt.result.CCLFNum, tt.result.Name, tt.result.ACOCMSID, tt.result.Timestamp, tt.result.PerformanceYear, tt.result.ImportStatus))
			}
			cclfFile, err := repository.GetLatestCCLFFile(cmsID, cclfNum, importStatus, tt.lowerBound, tt.upperBound)
			assert.NoError(t, err)

			if tt.result == nil {
				assert.Nil(t, cclfFile)
			} else {
				assert.Equal(t, tt.result, cclfFile)
			}
		})
	}
}

func (r *RepositoryTestSuite) TestGetCCLFBeneficiaryMBIs() {
	tests := []struct {
		name          string
		expQueryRegex string
		errToReturn   error
	}{
		{
			"HappyPath",
			`SELECT mbi FROM "cclf_beneficiaries" WHERE (file_id = $1)`,
			nil,
		},
		{
			"ErrorOnQuery",
			`SELECT mbi FROM "cclf_beneficiaries" WHERE (file_id = $1)`,
			fmt.Errorf("Some SQL error"),
		},
	}

	for _, tt := range tests {
		r.T().Run(tt.name, func(t *testing.T) {
			mbis := []string{"0", "1", "2"}
			cclfFileID := uint(rand.Int63())

			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			gdb, err := gorm.Open("postgres", db)
			if err != nil {
				t.Fatalf("Failed to instantiate gorm db %s", err.Error())
			}

			defer func() {
				err = mock.ExpectationsWereMet()
				assert.NoError(t, err)
				gdb.Close()
				db.Close()
			}()

			repository := NewRepository(gdb)

			query := mock.ExpectQuery(regexp.QuoteMeta(tt.expQueryRegex)).
				WithArgs(cclfFileID)
			if tt.errToReturn == nil {
				rows := sqlmock.NewRows([]string{"mbi"})
				for _, mbi := range mbis {
					rows.AddRow(mbi)
				}
				query.WillReturnRows(rows)
			} else {
				query.WillReturnError(tt.errToReturn)
			}

			result, err := repository.GetCCLFBeneficiaryMBIs(cclfFileID)
			if tt.errToReturn == nil {
				assert.NoError(t, err)
				assert.Equal(t, mbis, result)
			} else {
				assert.Error(t, err)
				assert.Nil(t, result)
			}
		})
	}
}

func (r *RepositoryTestSuite) TestGetCCLFBeneficiaries() {
	tests := []struct {
		name            string
		expQueryRegex   string
		ignoredMBIs     []string
		expectedResults []*models.CCLFBeneficiary
		errToReturn     error
	}{
		{
			"NoIgnoreMBIs",
			`SELECT * FROM "cclf_beneficiaries" WHERE "cclf_beneficiaries"."deleted_at" IS NULL AND ((id in (( SELECT id FROM ( SELECT max(id) as id, mbi FROM cclf_beneficiaries WHERE file_id = $1 GROUP BY mbi ) as id))))`,
			nil,
			[]*models.CCLFBeneficiary{
				getCCLFBeneficiary(),
				getCCLFBeneficiary(),
				getCCLFBeneficiary(),
				getCCLFBeneficiary(),
			},
			nil,
		},
		{
			"IgnoredMBIs",
			`SELECT * FROM "cclf_beneficiaries" WHERE "cclf_beneficiaries"."deleted_at" IS NULL AND ((id in (( SELECT id FROM ( SELECT max(id) as id, mbi FROM cclf_beneficiaries WHERE file_id = $1 GROUP BY mbi ) as id))) AND ("cclf_beneficiaries"."mbi" NOT IN ($2,$3)))`,
			[]string{"123", "456"},
			[]*models.CCLFBeneficiary{
				getCCLFBeneficiary(),
			},
			nil,
		},
		{
			"ErrorOnQuery",
			`SELECT * FROM "cclf_beneficiaries" WHERE "cclf_beneficiaries"."deleted_at" IS NULL AND ((id in (( SELECT id FROM ( SELECT max(id) as id, mbi FROM cclf_beneficiaries WHERE file_id = $1 GROUP BY mbi ) as id))))`,
			nil,
			nil,
			fmt.Errorf("Some SQL error"),
		},
	}

	for _, tt := range tests {
		r.T().Run(tt.name, func(t *testing.T) {
			cclfFileID := uint(rand.Int63())

			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			gdb, err := gorm.Open("postgres", db)
			if err != nil {
				t.Fatalf("Failed to instantiate gorm db %s", err.Error())
			}

			defer func() {
				err = mock.ExpectationsWereMet()
				assert.NoError(t, err)
				gdb.Close()
				db.Close()
			}()

			repository := NewRepository(gdb)

			var query *sqlmock.ExpectedQuery
			if tt.ignoredMBIs == nil {
				query = mock.ExpectQuery(regexp.QuoteMeta(tt.expQueryRegex)).
					WithArgs(cclfFileID)
			} else {
				args := []driver.Value{cclfFileID}
				for _, ignoredMBI := range tt.ignoredMBIs {
					args = append(args, ignoredMBI)
				}
				query = mock.ExpectQuery(regexp.QuoteMeta(tt.expQueryRegex)).
					WithArgs(args...)
			}
			if tt.errToReturn == nil {
				rows := sqlmock.NewRows([]string{"id", "file_id", "hicn", "mbi", "blue_button_id"})
				for _, bene := range tt.expectedResults {
					rows.AddRow(bene.ID, bene.FileID, bene.HICN, bene.MBI, bene.BlueButtonID)
				}
				query.WillReturnRows(rows)
			} else {
				query.WillReturnError(tt.errToReturn)
			}

			result, err := repository.GetCCLFBeneficiaries(cclfFileID, tt.ignoredMBIs)
			if tt.errToReturn == nil {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResults, result)
			} else {
				assert.Error(t, err)
				assert.Nil(t, result)
			}
		})
	}
}

func (r *RepositoryTestSuite) TestGetSuppressedMBIs() {
	lookbackDays := 10
	tests := []struct {
		name          string
		expQueryRegex string
		errToReturn   error
	}{
		{
			"HappyPath",
			`SELECT DISTINCT s.mbi FROM ( SELECT mbi, MAX(effective_date) max_date FROM suppressions WHERE (NOW() - interval '10 days') < effective_date AND effective_date <= NOW() AND preference_indicator != '' GROUP BY mbi ) h JOIN suppressions s ON s.mbi = h.mbi and s.effective_date = h.max_date WHERE preference_indicator = 'N'`,
			nil,
		},
		{
			"ErrorOnQuery",
			`SELECT DISTINCT s.mbi FROM ( SELECT mbi, MAX(effective_date) max_date FROM suppressions WHERE (NOW() - interval '10 days') < effective_date AND effective_date <= NOW() AND preference_indicator != '' GROUP BY mbi ) h JOIN suppressions s ON s.mbi = h.mbi and s.effective_date = h.max_date WHERE preference_indicator = 'N'`,
			fmt.Errorf("Some SQL error"),
		},
	}

	for _, tt := range tests {
		r.T().Run(tt.name, func(t *testing.T) {
			suppressedMBIs := []string{"0", "1", "2"}

			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			gdb, err := gorm.Open("postgres", db)
			if err != nil {
				t.Fatalf("Failed to instantiate gorm db %s", err.Error())
			}

			defer func() {
				err = mock.ExpectationsWereMet()
				assert.NoError(t, err)
				gdb.Close()
				db.Close()
			}()

			repository := NewRepository(gdb)

			// No arguments because the lookback days is embedded in the query
			query := mock.ExpectQuery(regexp.QuoteMeta(tt.expQueryRegex))
			if tt.errToReturn == nil {
				rows := sqlmock.NewRows([]string{"mbi"})
				for _, mbi := range suppressedMBIs {
					rows.AddRow(mbi)
				}
				query.WillReturnRows(rows)
			} else {
				query.WillReturnError(tt.errToReturn)
			}

			result, err := repository.GetSuppressedMBIs(lookbackDays)
			if tt.errToReturn == nil {
				assert.NoError(t, err)
				assert.Equal(t, suppressedMBIs, result)
			} else {
				assert.Error(t, err)
				assert.Nil(t, result)
			}
		})
	}
}

func getCCLFFile(cclfNum int, cmsID, importStatus string) *models.CCLFFile {
	createTime := time.Now()
	return &models.CCLFFile{
		Model: gorm.Model{
			ID: uint(rand.Int63()),
		},
		CCLFNum:         cclfNum,
		Name:            fmt.Sprintf("CCLFFile%d", rand.Uint64()),
		ACOCMSID:        cmsID,
		Timestamp:       createTime,
		PerformanceYear: 2020,
		ImportStatus:    importStatus,
	}
}

func getCCLFBeneficiary() *models.CCLFBeneficiary {
	return &models.CCLFBeneficiary{
		Model: gorm.Model{
			ID: uint(rand.Int63()),
		},
		FileID:       uint(rand.Uint32()),
		HICN:         fmt.Sprintf("HICN%d", rand.Uint32()),
		MBI:          fmt.Sprintf("MBI%d", rand.Uint32()),
		BlueButtonID: fmt.Sprintf("BlueButton%d", rand.Uint32()),
	}
}
