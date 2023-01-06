package message

import "errors"

// ServiceError wrappers errors from the service layer of coordinator/roller.
// It is separated from the network errors.
type ServiceError error

var (
	// ErrSignInvalid means verify signature failed.
	ErrSignInvalid = errors.New("auth signature verify fail")
)
