// Code generated by MockGen. DO NOT EDIT.
// Source: handlers.go

// Package mock_handlers is a generated GoMock package.
package mock_handlers

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	models "github.com/vkupriya/go-metrics/internal/server/models"
)

// MockStorage is a mock of Storage interface.
type MockStorage struct {
	ctrl     *gomock.Controller
	recorder *MockStorageMockRecorder
}

// MockStorageMockRecorder is the mock recorder for MockStorage.
type MockStorageMockRecorder struct {
	mock *MockStorage
}

// NewMockStorage creates a new mock instance.
func NewMockStorage(ctrl *gomock.Controller) *MockStorage {
	mock := &MockStorage{ctrl: ctrl}
	mock.recorder = &MockStorageMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStorage) EXPECT() *MockStorageMockRecorder {
	return m.recorder
}

// Close mocks base method.
func (m *MockStorage) Close() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Close")
}

// Close indicates an expected call of Close.
func (mr *MockStorageMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockStorage)(nil).Close))
}

// GetAllMetrics mocks base method.
func (m *MockStorage) GetAllMetrics(c *models.Config) (map[string]float64, map[string]int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAllMetrics", c)
	ret0, _ := ret[0].(map[string]float64)
	ret1, _ := ret[1].(map[string]int64)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetAllMetrics indicates an expected call of GetAllMetrics.
func (mr *MockStorageMockRecorder) GetAllMetrics(c interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAllMetrics", reflect.TypeOf((*MockStorage)(nil).GetAllMetrics), c)
}

// GetCounterMetric mocks base method.
func (m *MockStorage) GetCounterMetric(c *models.Config, name string) (int64, bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCounterMetric", c, name)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(bool)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetCounterMetric indicates an expected call of GetCounterMetric.
func (mr *MockStorageMockRecorder) GetCounterMetric(c, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCounterMetric", reflect.TypeOf((*MockStorage)(nil).GetCounterMetric), c, name)
}

// GetGaugeMetric mocks base method.
func (m *MockStorage) GetGaugeMetric(c *models.Config, name string) (float64, bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGaugeMetric", c, name)
	ret0, _ := ret[0].(float64)
	ret1, _ := ret[1].(bool)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetGaugeMetric indicates an expected call of GetGaugeMetric.
func (mr *MockStorageMockRecorder) GetGaugeMetric(c, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGaugeMetric", reflect.TypeOf((*MockStorage)(nil).GetGaugeMetric), c, name)
}

// PingStore mocks base method.
func (m *MockStorage) PingStore(c *models.Config) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PingStore", c)
	ret0, _ := ret[0].(error)
	return ret0
}

// PingStore indicates an expected call of PingStore.
func (mr *MockStorageMockRecorder) PingStore(c interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PingStore", reflect.TypeOf((*MockStorage)(nil).PingStore), c)
}

// UpdateBatch mocks base method.
func (m *MockStorage) UpdateBatch(c *models.Config, g, cr models.Metrics) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateBatch", c, g, cr)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateBatch indicates an expected call of UpdateBatch.
func (mr *MockStorageMockRecorder) UpdateBatch(c, g, cr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateBatch", reflect.TypeOf((*MockStorage)(nil).UpdateBatch), c, g, cr)
}

// UpdateCounterMetric mocks base method.
func (m *MockStorage) UpdateCounterMetric(c *models.Config, name string, value int64) (int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateCounterMetric", c, name, value)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateCounterMetric indicates an expected call of UpdateCounterMetric.
func (mr *MockStorageMockRecorder) UpdateCounterMetric(c, name, value interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateCounterMetric", reflect.TypeOf((*MockStorage)(nil).UpdateCounterMetric), c, name, value)
}

// UpdateGaugeMetric mocks base method.
func (m *MockStorage) UpdateGaugeMetric(c *models.Config, name string, value float64) (float64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateGaugeMetric", c, name, value)
	ret0, _ := ret[0].(float64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateGaugeMetric indicates an expected call of UpdateGaugeMetric.
func (mr *MockStorageMockRecorder) UpdateGaugeMetric(c, name, value interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateGaugeMetric", reflect.TypeOf((*MockStorage)(nil).UpdateGaugeMetric), c, name, value)
}
