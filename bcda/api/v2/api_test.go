package v2_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	v2 "github.com/CMSgov/bcda-app/bcda/api/v2"
	"github.com/samply/golang-fhir-models/fhir-models/fhir"
	"github.com/stretchr/testify/assert"
)

func TestMetadataResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(v2.Metadata))
	defer ts.Close()

	res, err := http.Get(ts.URL)
	assert.NoError(t, err)

	assert.Equal(t, "application/json", res.Header.Get("Content-Type"))
	assert.Equal(t, http.StatusOK, res.StatusCode)

	resp, err := ioutil.ReadAll(res.Body)
	assert.NoError(t, err)

	cs, err := fhir.UnmarshalCapabilityStatement(resp)
	assert.NoError(t, err)

	// Expecting an R4 response so we'll evaluate some fields to reflect that
	assert.Equal(t, fhir.FHIRVersion4_0_1, cs.FhirVersion)
	assert.Equal(t, 1, len(cs.Rest))
	assert.Equal(t, 2, len(cs.Rest[0].Resource))
	resourceData := []struct {
		rt           fhir.ResourceType
		opName       string
		opDefinition string
	}{
		{fhir.ResourceTypePatient, "patient-export", "http://hl7.org/fhir/uv/bulkdata/OperationDefinition/patient-export"},
		{fhir.ResourceTypeGroup, "group-export", "http://hl7.org/fhir/uv/bulkdata/OperationDefinition/group-export"},
	}

	for _, rd := range resourceData {
		var rr *fhir.CapabilityStatementRestResource
		for _, r := range cs.Rest[0].Resource {
			if r.Type == rd.rt {
				rr = &r
				break
			}
		}
		assert.NotNil(t, rr)
		assert.Equal(t, 1, len(rr.Operation))
		assert.Equal(t, rd.opName, rr.Operation[0].Name)
		assert.Equal(t, rd.opDefinition, rr.Operation[0].Definition)
	}

	// Need to validate our security.extensions manually since the extension
	// field does not have support for polymorphic types
	// See: https://github.com/samply/golang-fhir-models/issues/1
	var obj map[string]interface{}
	err = json.Unmarshal(resp, &obj)
	assert.NoError(t, err)

	targetExtension := obj["rest"].([]interface{})[0].(map[string]interface{})["security"].(map[string]interface{})["extension"].([]interface{})[0].(map[string]interface{})
	subExtension := targetExtension["extension"].([]interface{})[0].(map[string]interface{})

	assert.Equal(t, "http://fhir-registry.smarthealthit.org/StructureDefinition/oauth-uris", targetExtension["url"])
	assert.Equal(t, "token", subExtension["url"])
	assert.Equal(t, ts.URL, subExtension["valueUri"])

}
