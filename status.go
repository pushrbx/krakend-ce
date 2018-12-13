package krakend

import (
	"context"
	"net/http"
	"github.com/devopsfaith/krakend/transport/http/client"
)

func RestlessHTTPStatusHandler(ctx context.Context, resp *http.Response) (*http.Response, error) {
	switch resp.StatusCode {
	case
		http.StatusOK,
		http.StatusCreated,
		http.StatusBadRequest,
		http.StatusAccepted,
		http.StatusConflict,
		http.StatusAlreadyReported,
		http.StatusExpectationFailed,
		http.StatusForbidden,
		http.StatusFound,
		http.StatusNotFound,
		http.StatusNotModified,
		http.StatusGone,
		http.StatusInsufficientStorage,
		http.StatusGatewayTimeout,
		http.StatusMethodNotAllowed,
		http.StatusNotImplemented,
		http.StatusUnauthorized,
		http.StatusServiceUnavailable,
		http.StatusUnprocessableEntity,
		http.StatusNoContent,
		http.StatusPermanentRedirect,
		http.StatusTemporaryRedirect:
			return resp, nil
	default:
		return nil, client.ErrInvalidStatusCode
	}
}