// Code generated by mockery. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	v1alpha1 "github.com/kyma-project/telemetry-manager/apis/telemetry/v1alpha1"
)

// DryRunner is an autogenerated mock type for the DryRunner type
type DryRunner struct {
	mock.Mock
}

// RunPipeline provides a mock function with given fields: ctx, pipeline
func (_m *DryRunner) RunPipeline(ctx context.Context, pipeline *v1alpha1.LogPipeline) error {
	ret := _m.Called(ctx, pipeline)

	if len(ret) == 0 {
		panic("no return value specified for RunPipeline")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1alpha1.LogPipeline) error); ok {
		r0 = rf(ctx, pipeline)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewDryRunner creates a new instance of DryRunner. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewDryRunner(t interface {
	mock.TestingT
	Cleanup(func())
}) *DryRunner {
	mock := &DryRunner{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
