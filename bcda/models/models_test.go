package models

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"log"
	random "math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/go-chi/chi"
	"github.com/jinzhu/gorm"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ModelsTestSuite struct {
	suite.Suite

	// Re-initialized for every test
	db      *gorm.DB
	service *MockService
}

func (s *ModelsTestSuite) SetupTest() {
	InitializeGormModels()
	s.db = database.GetGORMDbConnection()
	s.service = &MockService{}

	serviceInstance = s.service
}

func (s *ModelsTestSuite) TearDownTest() {
	database.Close(s.db)
}

func (s *ModelsTestSuite) TestCreateACO() {
	assert := s.Assert()

	const ACOName = "ACO Name"
	cmsID := "A0000"
	acoUUID, err := CreateACO(ACOName, &cmsID)

	assert.Nil(err)
	assert.NotNil(acoUUID)

	var aco ACO
	err = s.db.Find(&aco, "UUID = ?", acoUUID).Error
	assert.Nil(err)
	assert.NotNil(aco)
	assert.Equal(ACOName, aco.Name)
	assert.Equal(acoUUID.String(), aco.ClientID)
	assert.Equal(cmsID, *aco.CMSID)
	pubKey, err := aco.GetPublicKey()
	assert.EqualError(err, "not able to decode PEM-formatted public key")
	assert.Nil(pubKey)
	assert.NotNil(GetATOPrivateKey())
	// should confirm the keys are a matched pair? i.e., encrypt something with one and decrypt with the other
	// the auth provider determines what the clientID contains (formatting, alphabet used, etc).
	// we require that it be representable in a string of less than 255 characters
	const ClientID = "Alpha client id"
	aco.ClientID = ClientID
	s.db.Save(aco)
	s.db.Find(&aco, "UUID = ?", acoUUID)
	assert.NotNil(aco)
	assert.Equal(ACOName, aco.Name)
	assert.NotNil(aco.ClientID)
	assert.Equal(ClientID, aco.ClientID)

	// make sure we can't duplicate the ACO UUID
	aco = ACO{
		UUID: acoUUID,
		Name: "Duplicate UUID Test",
	}
	err = s.db.Save(&aco).Error
	assert.EqualError(err, "pq: duplicate key value violates unique constraint \"acos_pkey\"")

	// Duplicate CMS ID
	aco = ACO{
		UUID:  uuid.NewRandom(),
		CMSID: &cmsID,
		Name:  "Duplicate CMS ID Test",
	}
	err = s.db.Save(&aco).Error
	assert.EqualError(err, "pq: duplicate key value violates unique constraint \"acos_cms_id_key\"")
}

func (s *ModelsTestSuite) TestACOPublicKeyColumn() {
	assert := s.Assert()

	// Setup ACO
	cmsID := "A4444"
	aco := ACO{Name: "Pub Key Test ACO", CMSID: &cmsID, UUID: uuid.NewRandom()}
	err := s.db.Create(&aco).Error
	assert.Nil(err)
	assert.NotEmpty(aco)
	defer s.db.Delete(&aco)

	// Setup key
	pubKey := GetATOPublicKey()
	publicKeyPKIX, err := x509.MarshalPKIXPublicKey(pubKey)
	assert.Nil(err, "unable to marshal public key")
	publicKeyBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyPKIX,
	})
	assert.NotNil(publicKeyBytes, "unexpectedly empty public key byte slice")

	// Save and verify
	aco.PublicKey = string(publicKeyBytes)
	err = s.db.Save(&aco).Error
	assert.Nil(err)
	err = s.db.First(&aco, "cms_id = ?", cmsID).Error
	assert.Nil(err)
	assert.NotEmpty(aco)
	assert.NotEmpty(aco.PublicKey)
	assert.Equal(publicKeyBytes, []byte(aco.PublicKey))
}

func (s *ModelsTestSuite) TestACOSavePublicKey() {
	assert := s.Assert()

	// Setup ACO
	cmsID := "A4445"
	aco := ACO{Name: "Pub Key Save Test ACO", CMSID: &cmsID, UUID: uuid.NewRandom()}
	err := s.db.Create(&aco).Error
	assert.Nil(err)
	defer s.db.Delete(&aco)

	// Setup key
	keyPair, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.Nil(err, "error creating random test keypair")
	publicKeyPKIX, err := x509.MarshalPKIXPublicKey(&keyPair.PublicKey)
	assert.Nil(err, "unable to marshal public key")
	publicKeyBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyPKIX,
	})
	assert.NotNil(publicKeyBytes, "unexpectedly empty public key byte slice")

	// Save key
	err = aco.SavePublicKey(bytes.NewReader(publicKeyBytes))
	if err != nil {
		assert.FailNow("error saving key: " + err.Error())
	}

	// Retrieve and verify
	err = s.db.Find(&aco, "cms_id = ?", cmsID).Error
	assert.Nil(err, "unable to retrieve ACO from database")
	assert.NotNil(aco)
	assert.NotNil(aco.PublicKey)

	// Retrieve and verify
	storedKey, err := aco.GetPublicKey()
	assert.Nil(err)
	assert.NotNil(storedKey)
	storedPublicKeyPKIX, err := x509.MarshalPKIXPublicKey(storedKey)
	assert.Nil(err, "unable to marshal saved public key")
	storedPublicKeyBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: storedPublicKeyPKIX,
	})
	assert.NotNil(storedPublicKeyBytes, "unexpectedly empty stored public key byte slice")
	assert.Equal(storedPublicKeyBytes, publicKeyBytes)
}

func (s *ModelsTestSuite) TestACOSavePublicKeyInvalidKey() {
	assert := s.Assert()

	// Setup ACO
	cmsID := "A4447"
	aco := ACO{Name: "Pub Key Save Test ACO", CMSID: &cmsID, UUID: uuid.NewRandom()}
	err := s.db.Create(&aco).Error
	assert.Nil(err)
	defer s.db.Delete(&aco)

	emptyPEM := "-----BEGIN RSA PUBLIC KEY-----    -----END RSA PUBLIC KEY-----"
	invalidPEM :=
		`-----BEGIN RSA PUBLIC KEY-----
z2v9wLlK4zPAs3pLln3R/4NnGFKw2Eku2JVFTotQ03gSmSzesZixicw8LxgYKbNV
oyTpERFansw6BbCJe7AP90rmaxCx80NiewFq+7ncqMbCMcqeUuCwk8MjS6bjvpcC
htFCqeRi6AAUDRg0pcG8yoM+jo13Z5RJPOIf3ofohncfH5wr5Q7qiOCE5VH4I7cp
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAsZYpl2VjUja8VgkgoQ9K
lgjvcjwaQZ7pLGrIA/BQcm+KnCIYOHaDH15eVDKQ+M2qE4FHRwLec/DTqlwg8TkT
IYjBnXgN1Sg18y+SkSYYklO4cxlvMO3V8gaot9amPmt4YbpgG7CyZ+BOUHuoGBTh
OwIDAQAB
-----END RSA PUBLIC KEY-----`
	keyPair, err := rsa.GenerateKey(rand.Reader, 512)
	assert.Nil(err, "unable to generate key pair")
	publicKeyPKIX, err := x509.MarshalPKIXPublicKey(&keyPair.PublicKey)
	assert.Nil(err, "unable to marshal public key")
	lowBitPubKey := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyPKIX,
	})
	assert.NotNil(lowBitPubKey, "unexpectedly empty public key byte slice")

	err = aco.SavePublicKey(strings.NewReader(""))
	assert.NotNil(err, "empty string should not be saved")

	err = aco.SavePublicKey(strings.NewReader(emptyPEM))
	assert.NotNil(err, "empty PEM should not be saved")

	err = aco.SavePublicKey(strings.NewReader(invalidPEM))
	assert.NotNil(err, "invalid PEM should not be saved")

	err = aco.SavePublicKey(bytes.NewReader(lowBitPubKey))
	assert.NotNil(err, "insecure public key should not be saved")
}

func (s *ModelsTestSuite) TestACOPublicKeyEmpty() {
	assert := s.Assert()
	emptyPEM := "-----BEGIN RSA PUBLIC KEY-----    -----END RSA PUBLIC KEY-----"
	validPEM :=
		`-----BEGIN RSA PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAsZYpl2VjUja8VgkgoQ9K
lgjvcjwaQZ7pLGrIA/BQcm+KnCIYOHaDH15eVDKQ+M2qE4FHRwLec/DTqlwg8TkT
IYjBnXgN1Sg18y+SkSYYklO4cxlvMO3V8gaot9amPmt4YbpgG7CyZ+BOUHuoGBTh
z2v9wLlK4zPAs3pLln3R/4NnGFKw2Eku2JVFTotQ03gSmSzesZixicw8LxgYKbNV
oyTpERFansw6BbCJe7AP90rmaxCx80NiewFq+7ncqMbCMcqeUuCwk8MjS6bjvpcC
htFCqeRi6AAUDRg0pcG8yoM+jo13Z5RJPOIf3ofohncfH5wr5Q7qiOCE5VH4I7cp
OwIDAQAB
-----END RSA PUBLIC KEY-----`
	emptyPubKey := ACO{PublicKey: ""}
	emptyPubKey2 := ACO{PublicKey: emptyPEM}
	nonEmptyPEM := ACO{PublicKey: validPEM}

	k, err := emptyPubKey.GetPublicKey()
	assert.EqualError(err, "not able to decode PEM-formatted public key")
	assert.Nil(k, "Empty string does not yield nil public key!")
	k, err = emptyPubKey2.GetPublicKey()
	assert.EqualError(err, "not able to decode PEM-formatted public key")
	assert.Nil(k, "Empty PEM key does not yield nil public key!")
	k, err = nonEmptyPEM.GetPublicKey()
	assert.Nil(err)
	assert.NotNil(k, "Valid PEM key yields nil public key!")
}

func (s *ModelsTestSuite) TestACOPublicKeyFixtures() {
	assert := s.Assert()
	acoUUID1 := "DBBD1CE1-AE24-435C-807D-ED45953077D3"
	acoUUID2 := constants.DevACOUUID

	var aco1 ACO
	var aco2 ACO
	err := s.db.First(&aco1, "uuid = ?", acoUUID1).Error
	assert.Nil(err)
	assert.NotEmpty(aco1, "This ACO (DBBD1CE1-AE24-435C-807D-ED45953077D3) is in the fixtures; why is it not being found?")
	assert.NotEmpty(aco1.PublicKey, "The fixture (DBBD1CE1-AE24-435C-807D-ED45953077D3) has data in the public_key column; why is it not being returned?")
	pubKey, err := aco1.GetPublicKey()
	assert.Nil(err)
	assert.NotNil(pubKey, "Public key for DBBD1CE1-AE24-435C-807D-ED45953077D3 is unexpectedly nil.  Was there a parsing error in aco.GetPublicKey?")

	err = s.db.First(&aco2, "uuid = ?", acoUUID2).Error
	assert.Nil(err)
	assert.NotEmpty(aco2, "This ACO (0C527D2E-2E8A-4808-B11D-0FA06BAF8254) is in the fixtures; why is it not being found?")
	assert.NotEmpty(aco2.PublicKey, "The fixture (0C527D2E-2E8A-4808-B11D-0FA06BAF8254) has data in the public_key column; why is it not being returned?")
	pubKey, err = aco2.GetPublicKey()
	assert.Nil(err)
	assert.NotNil(pubKey, "Public key for 0C527D2E-2E8A-4808-B11D-0FA06BAF8254 is unexpectedly nil.  Was there a parsing error in aco.GetPublicKey?")
}

func (s *ModelsTestSuite) TestACOPublicKeyRetrieve() {
	assert := s.Assert()

	// Setup ACO
	cmsID := "A4446"
	aco := ACO{Name: "Pub Key Test ACO", CMSID: &cmsID, UUID: uuid.NewRandom()}
	err := s.db.Create(&aco).Error
	assert.Nil(err)
	assert.NotEmpty(aco)
	defer s.db.Delete(&aco)

	// Setup key
	keyPair, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.Nil(err, "error creating random test keypair")
	publicKeyPKIX, err := x509.MarshalPKIXPublicKey(&keyPair.PublicKey)
	assert.Nil(err, "unable to marshal public key")
	publicKeyBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyPKIX,
	})
	assert.NotNil(publicKeyBytes, "unexpectedly empty public key byte slice")

	// Save
	aco.PublicKey = string(publicKeyBytes)
	err = s.db.Save(&aco).Error
	assert.Nil(err)
	s.db.Find(&aco, "cms_id = ?", cmsID)
	assert.NotNil(aco)
	assert.NotNil(aco.PublicKey)

	// Retrieve and verify
	storedKey, err := aco.GetPublicKey()
	if err != nil {
		assert.FailNow("error getting stored key")
	}
	if storedKey == nil {
		assert.FailNow("no stored key was found")
	}
	storedPublicKeyPKIX, err := x509.MarshalPKIXPublicKey(storedKey)
	assert.Nil(err, "unable to marshal saved public key")
	storedPublicKeyBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: storedPublicKeyPKIX,
	})
	assert.NotNil(storedPublicKeyBytes, "unexpectedly empty stored public key byte slice")
	assert.Equal(storedPublicKeyBytes, publicKeyBytes)
}

func (s *ModelsTestSuite) TestACOGetPublicKey_SSAS() {
	router := chi.NewRouter()
	keyStr := `-----BEGIN RSA PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAsZYpl2VjUja8VgkgoQ9K
lgjvcjwaQZ7pLGrIA/BQcm+KnCIYOHaDH15eVDKQ+M2qE4FHRwLec/DTqlwg8TkT
IYjBnXgN1Sg18y+SkSYYklO4cxlvMO3V8gaot9amPmt4YbpgG7CyZ+BOUHuoGBTh
z2v9wLlK4zPAs3pLln3R/4NnGFKw2Eku2JVFTotQ03gSmSzesZixicw8LxgYKbNV
oyTpERFansw6BbCJe7AP90rmaxCx80NiewFq+7ncqMbCMcqeUuCwk8MjS6bjvpcC
htFCqeRi6AAUDRg0pcG8yoM+jo13Z5RJPOIf3ofohncfH5wr5Q7qiOCE5VH4I7cp
OwIDAQAB
-----END RSA PUBLIC KEY-----
`
	router.Get("/system/{systemID}/key", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(`{ "client_id": "123456", "public_key": "` + strings.Replace(keyStr, "\n", "\\n", -1) + `" }`))
		if err != nil {
			log.Fatal(err)
		}
	})
	server := httptest.NewServer(router)

	origAuthProvider := os.Getenv("BCDA_AUTH_PROVIDER")
	os.Setenv("BCDA_AUTH_PROVIDER", "ssas")
	defer os.Setenv("BCDA_AUTH_PROVIDER", origAuthProvider)

	origSSASURL := os.Getenv("SSAS_URL")
	os.Setenv("SSAS_URL", server.URL)
	defer os.Setenv("SSAS_URL", origSSASURL)

	origSSASUseTLS := os.Getenv("SSAS_USE_TLS")
	os.Setenv("SSAS_USE_TLS", "false")
	defer os.Setenv("SSAS_USE_TLS", origSSASUseTLS)

	cmsID := "A0001"
	aco := ACO{Name: "Public key from SSAS ACO", CMSID: &cmsID, UUID: uuid.NewRandom(), ClientID: "100"}

	key, err := aco.GetPublicKey()
	if err != nil {
		s.FailNow("Failed to get key", err.Error())
	}

	keyBytes, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		s.FailNow("Failed to marshal key", err.Error())
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: keyBytes,
	})

	assert.Equal(s.T(), keyStr, string(pemBytes))
}

func TestModelsTestSuite(t *testing.T) {
	suite.Run(t, new(ModelsTestSuite))
}

func (s *ModelsTestSuite) TestJobCompleted() {

	j := Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		RequestURL: "/api/v1/Patient/$export",
		Status:     "Pending",
		JobCount:   1,
	}
	s.db.Save(&j)
	completed, err := j.CheckCompletedAndCleanup(s.db)
	assert.Nil(s.T(), err)
	assert.False(s.T(), completed)

	err = s.db.Create(&JobKey{JobID: j.ID, FileName: "SOMETHING.ndjson"}).Error
	assert.Nil(s.T(), err)
	completed, err = j.CheckCompletedAndCleanup(s.db)
	assert.Nil(s.T(), err)
	assert.True(s.T(), completed)
	s.db.Delete(&j)
}
func (s *ModelsTestSuite) TestJobDefaultCompleted() {

	// Job is completed, but no keys exist.  This is fine, it is still complete
	j := Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		RequestURL: "/api/v1/Patient/$export",
		Status:     "Completed",
		JobCount:   10,
	}
	s.db.Save(&j)

	completed, err := j.CheckCompletedAndCleanup(s.db)
	assert.Nil(s.T(), err)
	assert.True(s.T(), completed)
	s.db.Delete(&j)

}
func (s *ModelsTestSuite) TestJobwithKeysCompleted() {

	j := Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		RequestURL: "/api/v1/Patient/$export",
		Status:     "Pending",
		JobCount:   10,
	}
	s.db.Save(&j)
	completed, err := j.CheckCompletedAndCleanup(s.db)
	assert.Nil(s.T(), err)
	assert.False(s.T(), completed)

	for i := 1; i <= 5; i++ {
		err = s.db.Create(&JobKey{JobID: j.ID, FileName: "SOMETHING.ndjson"}).Error
		assert.Nil(s.T(), err)
	}
	// JobKeys exist, but not enough to make the job complete
	completed, err = j.CheckCompletedAndCleanup(s.db)
	assert.Nil(s.T(), err)
	assert.False(s.T(), completed)

	for i := 1; i <= 5; i++ {
		err = s.db.Create(&JobKey{JobID: j.ID, FileName: "SOMETHING.ndjson"}).Error
		assert.Nil(s.T(), err)
	}
	completed, err = j.CheckCompletedAndCleanup(s.db)
	assert.Nil(s.T(), err)
	assert.True(s.T(), completed)
	s.db.Delete(&j)
}

func (s *ModelsTestSuite) TestGetEnqueueJobs() {
	type expectedJobArgs struct {
		resourceType string
		since        string
		priority     int16
		numBenes     int
	}
	tests := []struct {
		name             string
		j                Job
		cmsID            string
		priorityACOs     string
		resourceTypes    []string
		since            string
		retrieveNewBenes bool
		numOldBenes      int
		numNewBenes      int
		expectedJobArgs  []expectedJobArgs

		// Optional methods that can be defined to set/unset
		// extra settings (e.g. environment variables)
		setup   func(t *testing.T)
		cleanup func(t *testing.T)
	}{
		{
			"AllResourcesTypes_WithSince_Patient",
			Job{ACOID: uuid.Parse(constants.DevACOUUID), RequestURL: "/api/v1/Patient/$export", Status: "Pending"},
			"A9994",
			"",
			[]string{"Patient", "ExplanationOfBenefit", "Coverage"},
			"2020-02-13T08:00:00.000-05:00",
			true,
			50,
			0, // No new benes because of the since time. CCLFnew == CCLFold
			[]expectedJobArgs{
				expectedJobArgs{"Patient", "gt2020-02-13T08:00:00.000-05:00", 20, 50},
				expectedJobArgs{"ExplanationOfBenefit", "gt2020-02-13T08:00:00.000-05:00", 30, 50},
				expectedJobArgs{"Coverage", "gt2020-02-13T08:00:00.000-05:00", 20, 50},
			},
			nil,
			nil,
		},
		{
			"AllResourcesTypes_WithOldSince_Group",
			Job{ACOID: uuid.Parse(constants.DevACOUUID), RequestURL: "/api/v1/Group/$export", Status: "Pending"},
			"A9994",
			"",
			[]string{"Patient", "ExplanationOfBenefit", "Coverage"},
			"1900-02-13T08:00:00.000-05:00",
			true,
			0, // No old benes because of the since time causes no CCLFold to be found
			50,
			[]expectedJobArgs{
				expectedJobArgs{"Patient", "", 20, 50},
				expectedJobArgs{"ExplanationOfBenefit", "", 30, 50},
				expectedJobArgs{"Coverage", "", 20, 50},
			},
			nil,
			nil,
		},
		{
			"AllResourcesTypes_WithSince_Group",
			Job{ACOID: uuid.Parse(constants.DevACOUUID), RequestURL: "/api/v1/Group/$export", Status: "Pending"},
			"A9994",
			"",
			[]string{"Patient", "ExplanationOfBenefit", "Coverage"},
			"2020-02-13T08:00:00.000-05:00",
			true,
			40, // oldBenes
			10, // newBenes
			[]expectedJobArgs{
				expectedJobArgs{"Patient", "", 20, 10},
				expectedJobArgs{"ExplanationOfBenefit", "", 30, 10},
				expectedJobArgs{"Coverage", "", 20, 10},
				expectedJobArgs{"Patient", "gt2020-02-13T08:00:00.000-05:00", 20, 40},
				expectedJobArgs{"ExplanationOfBenefit", "gt2020-02-13T08:00:00.000-05:00", 30, 40},
				expectedJobArgs{"Coverage", "gt2020-02-13T08:00:00.000-05:00", 20, 40},
			},
			nil,
			nil,
		},
		{
			"Patient",
			Job{ACOID: uuid.Parse(constants.DevACOUUID), RequestURL: "/api/v1/Patient/$export?_type=Patient", Status: "Pending"},
			"A9994",
			"",
			[]string{"Patient"},
			"",
			false,
			50,
			0, // no new benes because we're retrieveNewBenes == false
			[]expectedJobArgs{
				expectedJobArgs{"Patient", "", 20, 50},
			},
			nil,
			nil,
		},
		{
			"ExplanationOfBenefit",
			Job{ACOID: uuid.Parse(constants.DevACOUUID), RequestURL: "/api/v1/Patient/$export?_type=ExplanationOfBenefit", Status: "Pending"},
			"A9994",
			"",
			[]string{"ExplanationOfBenefit"},
			"",
			false,
			50,
			0, // no new benes because we're retrieveNewBenes == false
			// Distribution based on BCDA_FHIR_MAX_RECORDS_EOB and numBenes
			[]expectedJobArgs{
				expectedJobArgs{"ExplanationOfBenefit", "", 100, 15},
				expectedJobArgs{"ExplanationOfBenefit", "", 100, 15},
				expectedJobArgs{"ExplanationOfBenefit", "", 100, 15},
				expectedJobArgs{"ExplanationOfBenefit", "", 100, 5},
			},
			func(t *testing.T) {
				err := os.Setenv("BCDA_FHIR_MAX_RECORDS_EOB", "15")
				assert.NoError(t, err)
			},
			func(t *testing.T) {
				err := os.Unsetenv("BCDA_FHIR_MAX_RECORDS_EOB")
				assert.NoError(t, err)
			},
		},
		{
			"Coverage",
			Job{ACOID: uuid.Parse(constants.DevACOUUID), RequestURL: "/api/v1/Patient/$export?_type=Coverage", Status: "Pending"},
			"A9994",
			"",
			[]string{"Coverage"},
			"",
			false,
			50,
			0, // no new benes because we're retrieveNewBenes == false
			// Distribution based on BCDA_FHIR_MAX_RECORDS_COVERAGE and numBenes
			[]expectedJobArgs{
				expectedJobArgs{"Coverage", "", 20, 5},
				expectedJobArgs{"Coverage", "", 20, 5},
				expectedJobArgs{"Coverage", "", 20, 5},
				expectedJobArgs{"Coverage", "", 20, 5},
				expectedJobArgs{"Coverage", "", 20, 5},
				expectedJobArgs{"Coverage", "", 20, 5},
				expectedJobArgs{"Coverage", "", 20, 5},
				expectedJobArgs{"Coverage", "", 20, 5},
				expectedJobArgs{"Coverage", "", 20, 5},
				expectedJobArgs{"Coverage", "", 20, 5},
			},
			func(t *testing.T) {
				err := os.Setenv("BCDA_FHIR_MAX_RECORDS_COVERAGE", "5")
				assert.NoError(t, err)
			},
			func(t *testing.T) {
				err := os.Unsetenv("BCDA_FHIR_MAX_RECORDS_COVERAGE")
				assert.NoError(t, err)
			},
		},
		{
			"Patient_WithHighPriorityACOs",
			Job{ACOID: uuid.Parse(constants.DevACOUUID), RequestURL: "/api/v1/Patient/$export?_type=Patient", Status: "Pending"},
			"A9994",
			"A9990,A9991,A9992,A9993,A9994",
			[]string{"Patient"},
			"",
			false,
			50,
			0, // no new benes because we're retrieveNewBenes == false
			[]expectedJobArgs{
				expectedJobArgs{"Patient", "", 10, 50}, // Lower priority because A9994 is the last entry in the priority list
			},
			nil,
			nil,
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			// Need to make sure we start with a fresh mock instance each time.
			// That way expectations are cleared.
			s.service = &MockService{}
			serviceInstance = s.service

			s.db.Save(&tt.j)
			defer s.db.Delete(&tt.j)

			if tt.setup != nil {
				tt.setup(t)
			}
			if tt.cleanup != nil {
				defer tt.cleanup(t)
			}

			priorityACOsDefault := os.Getenv("PRIORITY_ACO_IDS")
			os.Setenv("PRIORITY_ACO_IDS", tt.priorityACOs)
			defer os.Setenv("PRIORITY_ACO_IDS", priorityACOsDefault)

			// NOTE: We can have some ID collisions but that's fine
			oldBenes := make([]*CCLFBeneficiary, 0, tt.numOldBenes)
			for i := 0; i < tt.numOldBenes; i++ {
				oldBenes = append(oldBenes, &CCLFBeneficiary{Model: gorm.Model{ID: uint(random.Uint64())}})
			}

			newBenes := make([]*CCLFBeneficiary, 0, tt.numNewBenes)
			for i := 0; i < tt.numNewBenes; i++ {
				newBenes = append(newBenes, &CCLFBeneficiary{Model: gorm.Model{ID: uint(random.Uint64())}})
			}

			if tt.retrieveNewBenes {
				sinceTime, err := time.Parse(time.RFC3339Nano, tt.since)
				assert.NoError(t, err)
				s.service.On("GetNewAndExistingBeneficiaries", tt.cmsID, sinceTime).Return(newBenes, oldBenes, nil)
			} else {
				s.service.On("GetBeneficiaries", tt.cmsID).Return(oldBenes, nil)
			}

			enqueueJobs, err := tt.j.GetEnqueJobs(tt.resourceTypes, tt.since, tt.retrieveNewBenes)
			assert.Nil(t, err)
			assert.Equal(t, len(tt.expectedJobArgs), len(enqueueJobs))

			for i, expected := range tt.expectedJobArgs {
				jobArgs := JobEnqueueArgs{}
				err := json.Unmarshal(enqueueJobs[i].Args, &jobArgs)
				if err != nil {
					assert.NoError(t, err)
				}
				assert.Equal(t, expected.resourceType, jobArgs.ResourceType)
				assert.Equal(t, expected.since, jobArgs.Since)
				assert.Equal(t, expected.priority, enqueueJobs[i].Priority)
				assert.Equal(t, expected.numBenes, len(jobArgs.BeneficiaryIDs))
			}

			s.service.AssertExpectations(t)
		})
	}
}
func (s *ModelsTestSuite) TestJobStatusMessage() {
	j := Job{Status: "In Progress", JobCount: 25, CompletedJobCount: 6}
	assert.Equal(s.T(), "In Progress (24%)", j.StatusMessage())

	j = Job{Status: "In Progress", JobCount: 0, CompletedJobCount: 0}
	assert.Equal(s.T(), "In Progress", j.StatusMessage())

	j = Job{Status: "Completed", JobCount: 25, CompletedJobCount: 25}
	assert.Equal(s.T(), "Completed", j.StatusMessage())
}

func (s *ModelsTestSuite) TestGetMaxBeneCount() {
	assert := s.Assert()

	// ExplanationOfBenefit
	eobMax, err := GetMaxBeneCount("ExplanationOfBenefit")
	assert.Equal(BCDA_FHIR_MAX_RECORDS_EOB_DEFAULT, eobMax)
	assert.Nil(err)

	err = os.Setenv("BCDA_FHIR_MAX_RECORDS_EOB", "5")
	if err != nil {
		s.T().Error(err)
	}
	eobMax, err = GetMaxBeneCount("ExplanationOfBenefit")
	assert.Equal(5, eobMax)
	assert.Nil(err)
	os.Unsetenv("BCDA_FHIR_MAX_RECORDS_EOB")

	// Patient
	patientMax, err := GetMaxBeneCount("Patient")
	assert.Equal(BCDA_FHIR_MAX_RECORDS_PATIENT_DEFAULT, patientMax)
	assert.Nil(err)

	err = os.Setenv("BCDA_FHIR_MAX_RECORDS_PATIENT", "10")
	if err != nil {
		s.T().Error(err)
	}
	patientMax, err = GetMaxBeneCount("Patient")
	assert.Equal(10, patientMax)
	assert.Nil(err)
	os.Unsetenv("BCDA_FHIR_MAX_RECORDS_PATIENT")

	// Coverage
	coverageMax, err := GetMaxBeneCount("Coverage")
	assert.Equal(BCDA_FHIR_MAX_RECORDS_COVERAGE_DEFAULT, coverageMax)
	assert.Nil(err)

	err = os.Setenv("BCDA_FHIR_MAX_RECORDS_COVERAGE", "15")
	if err != nil {
		s.T().Error(err)
	}
	coverageMax, err = GetMaxBeneCount("Coverage")
	assert.Equal(15, coverageMax)
	assert.Nil(err)
	os.Unsetenv("BCDA_FHIR_MAX_RECORDS_COVERAGE")

	// Invalid type
	max, err := GetMaxBeneCount("Coverages")
	assert.Equal(-1, max)
	assert.EqualError(err, "invalid request type")
}

func (s *ModelsTestSuite) TestGetBlueButtonID_CCLFBeneficiary() {
	assert := s.Assert()
	cclfBeneficiary := CCLFBeneficiary{MBI: "MBI"}
	bbc := testUtils.BlueButtonClient{}
	bbc.MBI = &cclfBeneficiary.MBI

	bbc.On("GetPatientByIdentifierHash", client.HashIdentifier(cclfBeneficiary.MBI)).Return(bbc.GetData("Patient", "BB_VALUE"))

	cclfBeneficiary.BlueButtonID = ""
	// New never seen before mbi, asks the mock blue button client for the value
	blueButtonID, err := cclfBeneficiary.GetBlueButtonID(&bbc)
	assert.Nil(err)
	assert.Equal("BB_VALUE", blueButtonID)

	// The object has a BB ID set on it already, but we still ask mock blue button client for the value
	// We should receive the BB_VALUE since we are ignoring cached values
	cclfBeneficiary.BlueButtonID = "LOCAL_VAL"
	blueButtonID, err = cclfBeneficiary.GetBlueButtonID(&bbc)
	assert.Nil(err)
	assert.Equal("BB_VALUE", blueButtonID)

	// Should be making two calls to BB for the MBI_MODE attemptsm, but this number will be four with the earlier test in this method.
	// This is due to the fact that we are not relying on cached identifiers
	bbc.AssertNumberOfCalls(s.T(), "GetPatientByIdentifierHash", 2)
}

func (s *ModelsTestSuite) TestDuplicateCCLFFileNames() {
	tests := []struct {
		name     string
		fileName string
		acoIDs   []string
		errMsg   string
	}{
		{"Different ACO ID", uuid.New(), []string{"ACO1", "ACO2"},
			""},
		{"Duplicate ACO ID", uuid.New(), []string{"ACO3", "ACO3"},
			`pq: duplicate key value violates unique constraint "idx_cclf_files_name_aco_cms_id_key"`},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			var err error
			var expectedFileCount int
			for _, acoID := range tt.acoIDs {
				cclfFile := &CCLFFile{
					Name:            tt.fileName,
					ACOCMSID:        acoID,
					Timestamp:       time.Now(),
					PerformanceYear: 20,
				}
				if err1 := s.db.Create(cclfFile).Error; err1 != nil {
					err = err1
					continue
				}
				expectedFileCount++
				defer func() {
					assert.Empty(t, cclfFile.Delete())
				}()
			}

			if tt.errMsg != "" {
				assert.EqualError(t, err, tt.errMsg)
			} else {
				assert.NoError(t, err)
			}

			var count int
			s.db.Model(&CCLFFile{}).Where("name = ?", tt.fileName).Count(&count)
			assert.True(t, expectedFileCount > 0)
			assert.Equal(t, expectedFileCount, count)
		})
	}
}

// TestCMSID verifies that we can store and retrieve the CMS_ID as expected
// i.e. the value is not padded with any extra characters
func (s *ModelsTestSuite) TestCMSID() {
	cmsID := "V001"
	cclfFile := &CCLFFile{CCLFNum: 1, Name: "someName", ACOCMSID: cmsID, Timestamp: time.Now(), PerformanceYear: 20}
	aco := &ACO{UUID: uuid.NewUUID(), CMSID: &cmsID, Name: "someName"}

	assert.NoError(s.T(), s.db.Save(cclfFile).Error)
	defer s.db.Unscoped().Delete(cclfFile)
	assert.NoError(s.T(), s.db.Save(aco).Error)
	defer s.db.Unscoped().Delete(aco)

	var actualCMSID []string
	assert.NoError(s.T(), s.db.Find(&ACO{}, "id = ?", aco.ID).Pluck("cms_id", &actualCMSID).Error)
	assert.Equal(s.T(), 1, len(actualCMSID))
	assert.Equal(s.T(), cmsID, actualCMSID[0])

	assert.NoError(s.T(), s.db.Find(&CCLFFile{}, "id = ?", cclfFile.ID).Pluck("aco_cms_id", &actualCMSID).Error)
	assert.Equal(s.T(), 1, len(actualCMSID))
	assert.Equal(s.T(), cmsID, actualCMSID[0])
}
