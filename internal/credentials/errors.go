package credentials

import "errors"

// ErrAuthenticationFailed indicates that the supplied credential
// could not authenticate the uniShell payload.
var ErrAuthenticationFailed = errors.New("authentication failed")
