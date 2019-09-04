package testutils

import (
	"archive/zip"
	"errors"
	"fmt"
	"github.com/CMSgov/bcda-app/bcda/cclf"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DestDir = "tempCCLFDir/"

// ImportCCLFPackage will copy the appropriate synthetic CCLF files, rename them,
// begin the import of those files and delete them from the place they were copied to after successful import.

func ImportCCLFPackage(acoSize, environment string) (err error) {
	acoSize = strings.ToLower(acoSize)
	// validation is here because this will get called from other locations than the CLI.
	switch acoSize {
	case
		"dev",
		"dev-auth",
		"small",
		"medium",
		"large",
		"extra-large":

	default:
		return errors.New("invalid argument for ACO size")
	}

	switch environment {
	case
		"test",
		"unit-test":
	default:
		return errors.New("invalid argument for environment")
	}

	sourcedir := filepath.Join("shared_files/syntheticCCLFFiles", environment, acoSize)
	sourcedir, err = utils.GetDirPath(sourcedir)
	if err != nil {
		return err
	}

	if _, err := os.Stat(DestDir); os.IsNotExist(err) {
		err = os.Mkdir(DestDir, os.ModePerm)
		if err != nil {
			return err
		}
	}

	dateString := fmt.Sprintf("%s.D%s.T%s", time.Now().Format("06"), time.Now().Format("060102"), time.Now().Format("1504059"))

	files, err := ioutil.ReadDir(sourcedir)
	if err != nil {
		return err
	}
	for _, file := range files {
		err = copyFiles(fmt.Sprintf("%s/%s", sourcedir, file.Name()), fmt.Sprintf("%s/%s%s", DestDir, file.Name(), dateString))
		if err != nil {
			return err
		}
	}

	success, failure, skipped, err := cclf.ImportCCLFDirectory(DestDir)
	if err != nil {
		return err
	}
	fmt.Printf("Completed CCLF import.  Successfully imported %d files.  Failed to import %d files.  Skipped %d files.  See logs for more details.\n", success, failure, skipped)
	if success == 2 {
		_, err = utils.DeleteDirectoryContents(DestDir)
		return err
	} else {
		err = errors.New("did not import 2 files")
		return err
	}
}

func copyFiles(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(filepath.Clean(src))
	if err != nil {
		return err
	}
	defer source.Close()

	newZipFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer newZipFile.Close()

	zipWriter := zip.NewWriter(newZipFile)
	defer zipWriter.Close()

	header, err := zip.FileInfoHeader(sourceFileStat)
	if err != nil {
		return err
	}

	// Using FileInfoHeader() above only uses the basename of the file. If we want
	// to preserve the folder structure we can overwrite this with the full path.
	header.Name = src

	// Change to deflate to gain better compression
	// see http://golang.org/pkg/archive/zip/#pkg-constants
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, source)
	return err
}
