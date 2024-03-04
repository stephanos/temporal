package testing

import (
	"github.com/golang/mock/gomock"
)

var _ gomock.Matcher = (*CustomMatcher[any])(nil)

type CustomMatcher[T any] struct {
	f func(val T) bool
}

func NewCustomMatcher[T any](f func(val T) bool) gomock.Matcher {
	return &CustomMatcher[T]{f: f}
}

func (m *CustomMatcher[T]) Matches(val interface{}) bool {
	return m.f(val.(T))
}

func (m *CustomMatcher[T]) String() string {
	return "CustomMatcher"
}
