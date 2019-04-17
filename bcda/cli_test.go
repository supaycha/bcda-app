package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/urfave/cli"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
)

type CLITestSuite struct {
	suite.Suite
	testApp *cli.App
}

func (s *CLITestSuite) SetupTest() {
	s.testApp = setUpApp()
}

func TestCLITestSuite(t *testing.T) {
	suite.Run(t, new(CLITestSuite))
}

func (s *CLITestSuite) TestCreateACO() {

	// init
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	// set up the test app writer (to redirect CLI responses from stdout to a byte buffer)
	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	assert := assert.New(s.T())

	// Successful ACO creation
	ACOName := "Unit Test ACO 1"
	args := []string{"bcda", "create-aco", "--name", ACOName}
	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.NotNil(buf)
	acoUUID := strings.TrimSpace(buf.String())
	var testACO models.ACO
	db.First(&testACO, "Name=?", ACOName)
	assert.Equal(testACO.UUID.String(), acoUUID)
	buf.Reset()

	ACO2Name := "Unit Test ACO 2"
	aco2ID := "A9999"
	args = []string{"bcda", "create-aco", "--name", ACO2Name, "--cms-id", aco2ID}
	err = s.testApp.Run(args)
	assert.Nil(err)
	assert.NotNil(buf)
	acoUUID = strings.TrimSpace(buf.String())
	var testACO2 models.ACO
	db.First(&testACO2, "Name=?", ACO2Name)
	assert.Equal(testACO2.UUID.String(), acoUUID)
	assert.Equal(*testACO2.CMSID, aco2ID)
	buf.Reset()

	// Negative tests

	// No parameters
	args = []string{"bcda", "create-aco"}
	err = s.testApp.Run(args)
	assert.Equal("ACO name (--name) must be provided", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// No ACO Name
	badACO := ""
	args = []string{"bcda", "create-aco", "--name", badACO}
	err = s.testApp.Run(args)
	assert.Equal("ACO name (--name) must be provided", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// ACO name without flag
	args = []string{"bcda", "create-aco", ACOName}
	err = s.testApp.Run(args)
	assert.Equal("ACO name (--name) must be provided", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// Unexpected flag
	args = []string{"bcda", "create-aco", "--abcd", "efg"}
	err = s.testApp.Run(args)
	assert.Equal("flag provided but not defined: -abcd", err.Error())
	assert.Contains(buf.String(), "Incorrect Usage: flag provided but not defined")
	buf.Reset()

	// Invalid CMS ID
	args = []string{"bcda", "create-aco", "--name", ACOName, "--cms-id", "ABCDE"}
	err = s.testApp.Run(args)
	assert.Equal("ACO CMS ID (--cms-id) is invalid", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()
}

func (s *CLITestSuite) TestImportCCLF8() {
	assert := assert.New(s.T())

	filePath := "../shared_files/cclf/T.A0001.ACO.ZC8Y18.D181120.T1000009"
	metadata, err := getCCLFFileMetadata(filePath)
	metadata.filePath = filePath
	assert.Nil(err)
	err = importCCLF8(metadata)
	assert.Nil(err)

	metadata.cclfNum = 9
	err = importCCLF8(metadata)
	assert.NotNil(err)

	metadata.cclfNum = 8
	metadata.filePath = ""
	err = importCCLF8(metadata)
	assert.NotNil(err)

}

func (s *CLITestSuite) TestImportCCLF9() {
	assert := assert.New(s.T())

	filePath := "../shared_files/cclf/T.A0001.ACO.ZC9Y18.D181120.T1000010"
	metadata, err := getCCLFFileMetadata(filePath)
	metadata.filePath = filePath
	assert.Nil(err)
	err = importCCLF9(metadata)
	assert.Nil(err)

	metadata.cclfNum = 8
	err = importCCLF9(metadata)
	assert.NotNil(err)

	metadata.cclfNum = 9
	metadata.filePath = ""
	err = importCCLF9(metadata)
	assert.NotNil(err)
}

func (s *CLITestSuite) TestGetCCLFFileMetadata() {
	assert := assert.New(s.T())

	_, err := getCCLFFileMetadata("/path/to/file")
	assert.EqualError(err, "invalid filename")

	metadata, err := getCCLFFileMetadata("/path/T.A0000.ACO.ZC8Y18.D190117.T9909420")
	assert.EqualError(err, "failed to parse date 'D190117.T990942' from filename")

	expTime, _ := time.Parse(time.RFC3339, "2019-01-17T21:09:42Z")
	metadata, err = getCCLFFileMetadata("/path/T.A0000.ACO.ZC8Y18.D190117.T2109420")
	assert.Equal("test", metadata.env)
	assert.Equal("A0000", metadata.acoID)
	assert.Equal(8, metadata.cclfNum)
	assert.Equal(expTime, metadata.timestamp)
	assert.Nil(err)

	expTime, _ = time.Parse(time.RFC3339, "2019-01-08T23:55:00Z")
	metadata, err = getCCLFFileMetadata("/path/P.A0001.ACO.ZC9Y18.D190108.T2355000")
	assert.Equal("production", metadata.env)
	assert.Equal("A0001", metadata.acoID)
	assert.Equal(9, metadata.cclfNum)
	assert.Equal(expTime, metadata.timestamp)
	assert.Nil(err)

	metadata, err = getCCLFFileMetadata("/cclf/T.A0001.ACO.ZC8Y18.D18NOV20.T1000010")
	assert.NotNil(err)

	metadata, err = getCCLFFileMetadata("/cclf/T.ABCDE.ACO.ZC8Y18.D181120.T1000010")
	assert.NotNil(err)

}
func (s *CLITestSuite) TestSortCCLFFiles() {
	assert := assert.New(s.T())
	var cclf0, cclf8, cclf9 []cclfFileMetadata
	var skipped int

	filePath := "../shared_files/cclf/"
	err := filepath.Walk(filePath, sortCCLFFiles(&cclf0, &cclf8, &cclf9, &skipped))
	assert.Nil(err)
	assert.Equal(1, len(cclf8))
	assert.Equal(1, len(cclf9))
	assert.Equal(0, len(cclf0))
	assert.Equal(0, skipped)

	cclf0, cclf8, cclf9 = nil, nil, nil
	skipped = 0

	filePath = "../shared_files/cclf_BadFileNames/"
	err = filepath.Walk(filePath, sortCCLFFiles(&cclf0, &cclf8, &cclf9, &skipped))
	assert.Nil(err)
	assert.Equal(1, len(cclf8))
	assert.Equal(0, len(cclf9))
	assert.Equal(0, len(cclf0))
	assert.Equal(2, skipped)

	cclf0, cclf8, cclf9 = nil, nil, nil
	skipped = 0

	filePath = "../shared_files/cclf0/"
	err = filepath.Walk(filePath, sortCCLFFiles(&cclf0, &cclf8, &cclf9, &skipped))
	assert.Nil(err)
	assert.Equal(0, len(cclf8))
	assert.Equal(0, len(cclf9))
	assert.Equal(3, len(cclf0))
	assert.Equal(0, skipped)

	cclf0, cclf8, cclf9 = nil, nil, nil
	skipped = 0

	filePath = "../shared_files/cclf8/"
	err = filepath.Walk(filePath, sortCCLFFiles(&cclf0, &cclf8, &cclf9, &skipped))
	assert.Nil(err)
	assert.Equal(5, len(cclf8))
	assert.Equal(0, len(cclf9))
	assert.Equal(0, len(cclf0))
	assert.Equal(0, skipped)

	cclf0, cclf8, cclf9 = nil, nil, nil
	skipped = 0

	filePath = "../shared_files/cclf9/"
	err = filepath.Walk(filePath, sortCCLFFiles(&cclf0, &cclf8, &cclf9, &skipped))
	assert.Nil(err)
	assert.Equal(0, len(cclf8))
	assert.Equal(4, len(cclf9))
	assert.Equal(0, len(cclf0))
	assert.Equal(0, skipped)

	cclf0, cclf8, cclf9 = nil, nil, nil
	skipped = 0

	filePath = "../shared_files/cclf_All/"
	err = filepath.Walk(filePath, sortCCLFFiles(&cclf0, &cclf8, &cclf9, &skipped))
	assert.Nil(err)
	assert.Equal(5, len(cclf8))
	assert.Equal(4, len(cclf9))
	assert.Equal(3, len(cclf0))
	for _, metadata := range cclf8 {
		assert.Equal(8, metadata.cclfNum)
	}
	for _, metadata := range cclf9 {
		assert.Equal(9, metadata.cclfNum)
	}
	for _, metadata := range cclf0 {
		assert.Equal(0, metadata.cclfNum)
	}
}

func (s *CLITestSuite) TestImportCCLFDirectory() {
	assert := assert.New(s.T())

	// set up the test app writer (to redirect CLI responses from stdout to a byte buffer)
	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	args := []string{"bcda", "import-cclf-directory", "--directory", "../shared_files/cclf/"}
	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.Contains(buf.String(), "Completed CCLF import.")
	assert.Contains(buf.String(), "Successfully imported 2 files.")
	assert.Contains(buf.String(), "Failed to import 0 files.")
	assert.Contains(buf.String(), "Skipped 0 files.")

	buf.Reset()

	// dir has 3 files, but 2 will be ignored because of bad file names.
	args = []string{"bcda", "import-cclf-directory", "--directory", "../shared_files/cclf_BadFileNames/"}
	err = s.testApp.Run(args)
	assert.Nil(err)
	assert.Contains(buf.String(), "Completed CCLF import.")
	assert.Contains(buf.String(), "Successfully imported 1 files.")
	assert.Contains(buf.String(), "Failed to import 0 files.")
	assert.Contains(buf.String(), "Skipped 2 files.")
	buf.Reset()

}
func (s *CLITestSuite) TestDeleteDirectory() {
	assert := assert.New(s.T())
	dirToDelete := "../shared_files/doomedDirectory"
	s.makeDirToDelete(dirToDelete)
	defer os.Remove(dirToDelete)

	f, err := os.Open(dirToDelete)
	assert.Nil(err)
	files, err := f.Readdir(-1)
	assert.Nil(err)
	assert.Equal(4, len(files))

	filesDeleted, err := deleteDirectoryContents(dirToDelete)
	assert.Equal(4, filesDeleted)
	assert.Nil(err)

	f, err = os.Open(dirToDelete)
	assert.Nil(err)
	files, err = f.Readdir(-1)
	assert.Nil(err)
	assert.Equal(0, len(files))

	filesDeleted, err = deleteDirectoryContents("This/Does/not/Exist")
	assert.Equal(0, filesDeleted)
	assert.NotNil(err)
}

func (s *CLITestSuite) TestDeleteDirectoryContents() {
	assert := assert.New(s.T())
	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	dirToDelete := "../shared_files/doomedDirectory"
	s.makeDirToDelete(dirToDelete)
	defer os.Remove(dirToDelete)

	args := []string{"bcda", "delete-dir-contents", "--dirToDelete", dirToDelete}
	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.Contains(buf.String(), fmt.Sprintf("Successfully Deleted 4 files from %v", dirToDelete))
	buf.Reset()

	// File, not a directory
	args = []string{"bcda", "delete-dir-contents", "--dirToDelete", "../shared_files/cclf/T.A0001.ACO.ZC8Y18.D181120.T1000009"}
	err = s.testApp.Run(args)
	assert.NotNil(err)
	assert.NotContains(buf.String(), "Successfully Deleted")
	buf.Reset()

	os.Setenv("TESTDELETEDIRECTORY", "NOT/A/REAL/DIRECTORY")
	args = []string{"bcda", "delete-dir-contents", "--envvar", "TESTDELETEDIRECTORY"}
	err = s.testApp.Run(args)
	assert.NotNil(err)
	assert.NotContains(buf.String(), "Successfully Deleted")
	buf.Reset()

}

func (s *CLITestSuite) makeDirToDelete(filePath string) {
	assert := assert.New(s.T())
	dirToDelete := filePath
	err := os.Mkdir(dirToDelete, os.ModePerm)
	assert.Nil(err)

	_, err = os.Create(filepath.Join(dirToDelete, "deleteMe1.txt"))
	assert.Nil(err)
	_, err = os.Create(filepath.Join(dirToDelete, "deleteMe2.txt"))
	assert.Nil(err)
	_, err = os.Create(filepath.Join(dirToDelete, "deleteMe3.txt"))
	assert.Nil(err)
	_, err = os.Create(filepath.Join(dirToDelete, "deleteMe4.txt"))
	assert.Nil(err)
}
