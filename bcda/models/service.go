package models

import (
	"fmt"
	"regexp"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	log "github.com/sirupsen/logrus"
)

// Ensure service satisfies the interface
var _ Service = &service{}

// Service contains all of the methods needed to interact with the data represented in the models package
type Service interface {
	cclfBeneficiaryService
}

type cclfBeneficiaryService interface {
	// GetNewAndExistingBeneficiaries, when supplied with the "since" parameter, returns two arrays
	// the first array contains all NEW beneficaries that were added to CCLF since the date supplied
	// the second array contains all EXISTING benficiaries that have existed in CCLF since prior to the date supplied
	GetNewAndExistingBeneficiaries(cmsID string, since time.Time) (newBeneficiaries, beneficiaries []*CCLFBeneficiary, err error)

	// GetBeneficiaries retrieves all beneficiaries associated with the ACO, contained in one array
	GetBeneficiaries(cmsID string) ([]*CCLFBeneficiary, error)
}

const (
	cclf8FileNum = int(8)
)

func newService(r Repository, cutoffDuration time.Duration, lookbackDays int) Service {
	serviceInstance = &service{
		repository:     r,
		logger:         log.StandardLogger(),
		cutoffDuration: cutoffDuration,
		sp: suppressionParameters{
			includeSuppressedBeneficiaries: false,
			lookbackDays:                   lookbackDays,
		},
	}

	return serviceInstance
}

type service struct {
	repository Repository

	logger *log.Logger

	cutoffDuration time.Duration
	sp             suppressionParameters
}

type suppressionParameters struct {
	includeSuppressedBeneficiaries bool
	lookbackDays                   int
}

func (s *service) GetNewAndExistingBeneficiaries(cmsID string, since time.Time) (newBeneficiaries, beneficiaries []*CCLFBeneficiary, err error) {

	var (
		cutoffTime time.Time
	)

	if s.cutoffDuration > 0 {
		cutoffTime = time.Now().Add(-1 * s.cutoffDuration)
	}

	cclfFileNew, err := s.repository.GetLatestCCLFFile(cmsID, cclf8FileNum, constants.ImportComplete, cutoffTime, time.Time{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get new CCLF file for cmsID %s %s", cmsID, err.Error())
	}
	if cclfFileNew == nil {
		return nil, nil, fmt.Errorf("no CCLF8 file found for cmsID %s cutoffTime %s", cmsID, cutoffTime.String())
	}

	cclfFileOld, err := s.repository.GetLatestCCLFFile(cmsID, cclf8FileNum, constants.ImportComplete, time.Time{}, since)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get old CCLF file for cmsID %s %s", cmsID, err.Error())
	}

	if cclfFileOld == nil {
		s.logger.Infof("Unable to find CCLF8 File for cmsID %s prior to date: %s; all beneficiaries will be considered NEW", cmsID, since)
		newBeneficiaries, err = s.getBenes(cclfFileNew.ID)
		if err != nil {
			return nil, nil, err
		}
		if len(newBeneficiaries) == 0 {
			return nil, nil, fmt.Errorf("Found 0 new beneficiaries from CCLF8 file for cmsID %s cclfFiledID %d", cmsID, cclfFileNew.ID)
		}
		return newBeneficiaries, nil, nil
	}

	oldMBIs, err := s.repository.GetCCLFBeneficiaryMBIs(cclfFileOld.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retreive MBIs for cmsID %s cclfFileID %d %s", cmsID, cclfFileOld.ID, err.Error())
	}

	// Retrieve all of the benes associated with this CCLF file.
	benes, err := s.getBenes(cclfFileNew.ID)
	if err != nil {
		return nil, nil, err
	}
	if len(benes) == 0 {
		return nil, nil, fmt.Errorf("Found 0 new or existing beneficiaries from CCLF8 file for cmsID %s cclfFiledID %d", cmsID, cclfFileNew.ID)
	}

	// Split the results beteween new and old benes based on the existence of the bene in the old map
	oldMBIMap := make(map[string]struct{}, len(oldMBIs))
	for _, oldMBI := range oldMBIs {
		oldMBIMap[oldMBI] = struct{}{}
	}
	for _, bene := range benes {
		if _, ok := oldMBIMap[bene.MBI]; ok {
			beneficiaries = append(beneficiaries, bene)
		} else {
			newBeneficiaries = append(newBeneficiaries, bene)
		}
	}

	return newBeneficiaries, beneficiaries, nil
}

func (s *service) GetBeneficiaries(cmsID string) ([]*CCLFBeneficiary, error) {
	var (
		cutoffTime time.Time
	)

	if s.cutoffDuration > 0 {
		cutoffTime = time.Now().Add(-1 * s.cutoffDuration)
	}

	cclfFile, err := s.repository.GetLatestCCLFFile(cmsID, cclf8FileNum, constants.ImportComplete, cutoffTime, time.Time{})
	if err != nil {
		return nil, fmt.Errorf("failed to get CCLF file for cmsID %s %s", cmsID, err.Error())
	}
	if cclfFile == nil {
		return nil, fmt.Errorf("no CCLF8 file found for cmsID %s cutoffTime %s", cmsID, cutoffTime.String())
	}

	benes, err := s.getBenes(cclfFile.ID)
	if err != nil {
		return nil, err
	}
	if len(benes) == 0 {
		return nil, fmt.Errorf("Found 0 beneficiaries from CCLF8 file for cmsID %s cclfFiledID %d", cmsID, cclfFile.ID)
	}

	return benes, nil
}

func (s *service) getBenes(cclfFileID uint) ([]*CCLFBeneficiary, error) {
	var (
		ignoredMBIs []string
		err         error
	)
	if !s.sp.includeSuppressedBeneficiaries {
		ignoredMBIs, err = s.repository.GetSuppressedMBIs(s.sp.lookbackDays)
		if err != nil {
			return nil, fmt.Errorf("failed to retreive suppressedMBIs %s", err.Error())
		}
	}

	benes, err := s.repository.GetCCLFBeneficiaries(cclfFileID, ignoredMBIs)
	if err != nil {
		return nil, fmt.Errorf("failed to get beneficiaries %s", err.Error())
	}

	return benes, nil
}

// IsSupportedACO determines if the particular ACO is supported by checking
// its CMS_ID against the supported formats.
func IsSupportedACO(cmsID string) bool {
	const (
		ssp     = `^A\d{4}$`
		ngaco   = `^V\d{3}$`
		cec     = `^E\d{4}$`
		pattern = `(` + ssp + `)|(` + ngaco + `)|(` + cec + `)`
	)

	return regexp.MustCompile(pattern).MatchString(cmsID)
}
