package allocation

import "errors"

// StatusError carries the HTTP status a backend returned, so callers can
// tell a credential failure (401, 403) from an outage. The message never
// contains the API token or the Authorization header.
type StatusError struct {
	Code int
	Msg  string
}

func (e *StatusError) Error() string { return e.Msg }

// HTTPStatus reports the backend HTTP status an error carries, if any.
func HTTPStatus(err error) (int, bool) {
	var statusErr *StatusError
	if errors.As(err, &statusErr) {
		return statusErr.Code, true
	}
	return 0, false
}
