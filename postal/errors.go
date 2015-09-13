package postal

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/cloudfoundry-incubator/notifications/uaa"
)

type UAAScopesError struct {
	Err error
}

func (e UAAScopesError) Error() string {
	return e.Err.Error()
}

type UAAUserNotFoundError struct {
	Err error
}

func (e UAAUserNotFoundError) Error() string {
	return e.Err.Error()
}

type TemplateLoadError struct {
	Err error
}

func (e TemplateLoadError) Error() string {
	return e.Err.Error()
}

type CriticalNotificationError struct {
	Err error
}

func NewCriticalNotificationError(kindID string) CriticalNotificationError {
	return CriticalNotificationError{fmt.Errorf("Insufficient privileges to send notification %s", kindID)}
}

func (e CriticalNotificationError) Error() string {
	return e.Err.Error()
}

func UAAErrorFor(err error) error {
	switch err.(type) {
	case *url.Error:
		return UAADownError{errors.New("UAA is unavailable")}
	case uaa.Failure:
		failure := err.(uaa.Failure)

		if failure.Code() == http.StatusNotFound {
			if strings.Contains(failure.Message(), "Requested route") {
				return UAADownError{errors.New("UAA is unavailable")}
			} else {
				return UAAGenericError{errors.New("UAA Unknown 404 error message: " + failure.Message())}
			}
		}

		return UAADownError{errors.New(failure.Message())}
	default:
		return UAAGenericError{errors.New("UAA Unknown Error: " + err.Error())}
	}
}

type UAADownError struct {
	Err error
}

func (e UAADownError) Error() string {
	return e.Err.Error()
}

type UAAGenericError struct {
	Err error
}

func (e UAAGenericError) Error() string {
	return e.Err.Error()
}
