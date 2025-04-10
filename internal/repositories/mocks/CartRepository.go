// Code generated by mockery v2.53.3. DO NOT EDIT.

package mocks

import (
	context "context"

	models "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	mock "github.com/stretchr/testify/mock"

	uuid "github.com/google/uuid"
)

// CartRepository is an autogenerated mock type for the CartRepository type
type CartRepository struct {
	mock.Mock
}

// CreateCart provides a mock function with given fields: ctx, cart
func (_m *CartRepository) CreateCart(ctx context.Context, cart *models.Cart) error {
	ret := _m.Called(ctx, cart)

	if len(ret) == 0 {
		panic("no return value specified for CreateCart")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.Cart) error); ok {
		r0 = rf(ctx, cart)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetCartByCustomerID provides a mock function with given fields: ctx, customerID
func (_m *CartRepository) GetCartByCustomerID(ctx context.Context, customerID uuid.UUID) (*models.Cart, error) {
	ret := _m.Called(ctx, customerID)

	if len(ret) == 0 {
		panic("no return value specified for GetCartByCustomerID")
	}

	var r0 *models.Cart
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, uuid.UUID) (*models.Cart, error)); ok {
		return rf(ctx, customerID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, uuid.UUID) *models.Cart); ok {
		r0 = rf(ctx, customerID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Cart)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, uuid.UUID) error); ok {
		r1 = rf(ctx, customerID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateCart provides a mock function with given fields: ctx, cart
func (_m *CartRepository) UpdateCart(ctx context.Context, cart *models.Cart) error {
	ret := _m.Called(ctx, cart)

	if len(ret) == 0 {
		panic("no return value specified for UpdateCart")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.Cart) error); ok {
		r0 = rf(ctx, cart)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewCartRepository creates a new instance of CartRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewCartRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *CartRepository {
	mock := &CartRepository{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
