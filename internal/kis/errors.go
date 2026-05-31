// Package kis implements KIS (Korea Investment Securities) Open API clients.
package kis

import "fmt"

// KisAPIBusinessError is raised when KIS returns a business error in a 200 response body.
type KisAPIBusinessError struct {
	Code    string
	Message string
}

func (e *KisAPIBusinessError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
