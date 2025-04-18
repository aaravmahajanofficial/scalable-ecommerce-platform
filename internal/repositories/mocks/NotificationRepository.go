// Code generated by mockery v2.53.3. DO NOT EDIT.

package mocks

import (
	context "context"

	models "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	mock "github.com/stretchr/testify/mock"

	uuid "github.com/google/uuid"
)

// NotificationRepository is an autogenerated mock type for the NotificationRepository type
type NotificationRepository struct {
	mock.Mock
}

// CreateNotification provides a mock function with given fields: ctx, notification
func (_m *NotificationRepository) CreateNotification(ctx context.Context, notification *models.Notification) error {
	ret := _m.Called(ctx, notification)

	if len(ret) == 0 {
		panic("no return value specified for CreateNotification")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.Notification) error); ok {
		r0 = rf(ctx, notification)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetNotificationById provides a mock function with given fields: ctx, id
func (_m *NotificationRepository) GetNotificationById(ctx context.Context, id uuid.UUID) (*models.Notification, error) {
	ret := _m.Called(ctx, id)

	if len(ret) == 0 {
		panic("no return value specified for GetNotificationById")
	}

	var r0 *models.Notification
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, uuid.UUID) (*models.Notification, error)); ok {
		return rf(ctx, id)
	}
	if rf, ok := ret.Get(0).(func(context.Context, uuid.UUID) *models.Notification); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Notification)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, uuid.UUID) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListNotifications provides a mock function with given fields: ctx, page, size
func (_m *NotificationRepository) ListNotifications(ctx context.Context, page int, size int) ([]*models.Notification, int, error) {
	ret := _m.Called(ctx, page, size)

	if len(ret) == 0 {
		panic("no return value specified for ListNotifications")
	}

	var r0 []*models.Notification
	var r1 int
	var r2 error
	if rf, ok := ret.Get(0).(func(context.Context, int, int) ([]*models.Notification, int, error)); ok {
		return rf(ctx, page, size)
	}
	if rf, ok := ret.Get(0).(func(context.Context, int, int) []*models.Notification); ok {
		r0 = rf(ctx, page, size)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*models.Notification)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, int, int) int); ok {
		r1 = rf(ctx, page, size)
	} else {
		r1 = ret.Get(1).(int)
	}

	if rf, ok := ret.Get(2).(func(context.Context, int, int) error); ok {
		r2 = rf(ctx, page, size)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// UpdateNotificationStatus provides a mock function with given fields: ctx, id, status, errorMsg
func (_m *NotificationRepository) UpdateNotificationStatus(ctx context.Context, id uuid.UUID, status models.NotificationStatus, errorMsg string) error {
	ret := _m.Called(ctx, id, status, errorMsg)

	if len(ret) == 0 {
		panic("no return value specified for UpdateNotificationStatus")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, uuid.UUID, models.NotificationStatus, string) error); ok {
		r0 = rf(ctx, id, status, errorMsg)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewNotificationRepository creates a new instance of NotificationRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewNotificationRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *NotificationRepository {
	mock := &NotificationRepository{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
