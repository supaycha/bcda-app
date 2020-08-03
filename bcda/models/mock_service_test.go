// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

// Suffixed with _test to avoid placing test code in main path.
// We need this in the same package as models.go to avoid the circular reference.
// Once models.go no longer needs access to Service, we can move this file to the mocks package

package models

import (
	time "time"

	mock "github.com/stretchr/testify/mock"
)

// MockService is an autogenerated mock type for the Service type
type MockService struct {
	mock.Mock
}

// GetBeneficiaries provides a mock function with given fields: cmsID
func (_m *MockService) GetBeneficiaries(cmsID string) ([]*CCLFBeneficiary, error) {
	ret := _m.Called(cmsID)

	var r0 []*CCLFBeneficiary
	if rf, ok := ret.Get(0).(func(string) []*CCLFBeneficiary); ok {
		r0 = rf(cmsID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*CCLFBeneficiary)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(cmsID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetNewAndExistingBeneficiaries provides a mock function with given fields: cmsID, since
func (_m *MockService) GetNewAndExistingBeneficiaries(cmsID string, since time.Time) ([]*CCLFBeneficiary, []*CCLFBeneficiary, error) {
	ret := _m.Called(cmsID, since)

	var r0 []*CCLFBeneficiary
	if rf, ok := ret.Get(0).(func(string, time.Time) []*CCLFBeneficiary); ok {
		r0 = rf(cmsID, since)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*CCLFBeneficiary)
		}
	}

	var r1 []*CCLFBeneficiary
	if rf, ok := ret.Get(1).(func(string, time.Time) []*CCLFBeneficiary); ok {
		r1 = rf(cmsID, since)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).([]*CCLFBeneficiary)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(string, time.Time) error); ok {
		r2 = rf(cmsID, since)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
