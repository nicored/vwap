package service

import "github.com/stretchr/testify/mock"

type VWAPMock struct {
	mock.Mock
}

func (v *VWAPMock) Value() float64 {
	ret := v.Called()

	var r0 float64
	if rf, ok := ret.Get(0).(float64); ok {
		r0 = rf
	}

	return r0
}

func (v *VWAPMock) NPoints() int {
	ret := v.Called()

	var r0 int
	if rf, ok := ret.Get(0).(int); ok {
		r0 = rf
	}

	return r0
}

func (v *VWAPMock) Push(price float64, volume float64) error {
	ret := v.Called(price, volume)

	var r0 error
	if rf, ok := ret.Get(0).(error); ok {
		r0 = rf
	}

	return r0
}

type StreamerMock struct {
	mock.Mock
}

func (s *StreamerMock) Subscribe(channel string, productIDs ...string) error {
	ret := s.Called(channel, productIDs)

	var r0 error
	if rf, ok := ret.Get(0).(error); ok {
		r0 = rf
	}
	return r0
}

func (s *StreamerMock) Unsubscribe(channel string, productIDs ...string) error {
	ret := s.Called(channel, productIDs)

	var r0 error
	if rf, ok := ret.Get(0).(error); ok {
		r0 = rf
	}
	return r0
}

func (s *StreamerMock) Feeds() (feeds chan []byte, feedsErr chan error) {
	ret := s.Called()

	var r0 chan []byte
	if rf, ok := ret.Get(0).(chan []byte); ok {
		r0 = rf
	}

	var r1 chan error
	if rf, ok := ret.Get(1).(chan error); ok {
		r1 = rf
	}

	return r0, r1
}

func (s *StreamerMock) Close() error {
	ret := s.Called()

	var r0 error
	if rf, ok := ret.Get(0).(error); ok {
		r0 = rf
	}
	return r0
}
