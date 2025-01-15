package config

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockConfiguration is a mock of Configuration interface.
type MockConfiguration struct {
	ctrl     *gomock.Controller
	recorder *MockConfigurationMockRecorder
}

// MockConfigurationMockRecorder is the mock recorder for MockConfiguration.
type MockConfigurationMockRecorder struct {
	mock *MockConfiguration
}

// NewMockConfiguration creates a new mock instance.
func NewMockConfiguration(ctrl *gomock.Controller) *MockConfiguration {
	mock := &MockConfiguration{ctrl: ctrl}
	mock.recorder = &MockConfigurationMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockConfiguration) EXPECT() *MockConfigurationMockRecorder {
	return m.recorder
}

// GetConfig mocks base method.
func (m *MockConfiguration) GetConfig() *AllConfiguration {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetConfig")
	ret0, _ := ret[0].(*AllConfiguration)
	return ret0
}

// GetConfig indicates an expected call of GetConfig.
func (mr *MockConfigurationMockRecorder) GetConfig() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetConfig", reflect.TypeOf((*MockConfiguration)(nil).GetConfig))
}

// LoadConfiguration mocks base method.
func (m *MockConfiguration) LoadConfiguration() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "LoadConfiguration")
}

// LoadConfiguration indicates an expected call of LoadConfiguration.
func (mr *MockConfigurationMockRecorder) LoadConfiguration() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LoadConfiguration", reflect.TypeOf((*MockConfiguration)(nil).LoadConfiguration))
}
