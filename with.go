package wrap

import (
	"errors"
	reflectlite "reflect" // reflectlite is a package internal to the stdlib, but its API is the same as reflect.
)

// With returns an error that represents top wrapped over bottom.
// Unwrap will unwrap the top error first, until it runs out of wrapped
// errors, and then return the bottom error. This is also the order that
// Is and As will read the wrapped errors.
// The returned error's message will read as
// fmt.Sprintf("%s: %s", top.Error(), bottom.Error()).
func With(bottom, top error) error {
	if bottom == nil && top == nil {
		return nil
	}
	if top == nil {
		return bottom
	}
	if bottom == nil {
		return top
	}
	return stack{top: top, bottom: bottom}
}

type stack struct {
	top    error
	bottom error
}

func (s stack) Is(target error) bool {
	// Copied from errors.Is, but without iterative unwrapping.
	// If top doesn't match, errors.Is will Unwrap, which does the right thing.
	if target == nil {
		return false
	}

	isComparable := reflectlite.TypeOf(target).Comparable()
	if isComparable && s.top == target {
		return true
	}
	if x, ok := s.top.(interface{ Is(error) bool }); ok && x.Is(target) {
		return true
	}
	return false
}

func (s stack) As(target interface{}) bool {
	// copied from errors.As, but without the iterative unwrapping.
	// If top doesn't match, errors.Is will Unwrap, which does the right thing.

	if target == nil {
		panic("errors: target cannot be nil")
	}
	val := reflectlite.ValueOf(target)
	typ := val.Type()
	if typ.Kind() != reflectlite.Ptr || val.IsNil() {
		panic("errors: target must be a non-nil pointer")
	}
	targetType := typ.Elem()
	if targetType.Kind() != reflectlite.Interface && !targetType.Implements(errorType) {
		panic("errors: *target must be interface or implement error")
	}
	if reflectlite.TypeOf(s.top).AssignableTo(targetType) {
		val.Elem().Set(reflectlite.ValueOf(s.top))
		return true
	}
	if x, ok := s.top.(interface{ As(interface{}) bool }); ok && x.As(target) {
		return true
	}
	return false
}

var errorType = reflectlite.TypeOf((*error)(nil)).Elem()

// Unwrap iteratively unwraps the errors stacked on top until it runs,
// out of wrapped errors, and then returns the bottom error.
func (s stack) Unwrap() error {
	if err := errors.Unwrap(s.top); err != nil {
		return stack{top: err, bottom: s.bottom}
	}
	// otherwise we ran out of errors on top to unwrap, so return the underlying error.
	return s.bottom
}

func (s stack) Error() string {
	return s.top.Error() + ": " + s.bottom.Error()
}
