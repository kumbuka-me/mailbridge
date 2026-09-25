package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/containeroo/mailbridge/internal/application"
	"github.com/containeroo/mailbridge/internal/response"
)

const maxRequestBytes = 1 << 20

// Forwarder forwards one decoded mail request.
type Forwarder interface {
	Forward(context.Context, application.Request) error
}

// Mail decodes one JSON mail request and forwards it as exactly one email.
func Mail(forwarder Forwarder, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		request, err := decodeRequest(w, r)
		if err != nil {
			response.Problem(w, http.StatusBadRequest, "Invalid mail payload.")
			return
		}

		if err := forwarder.Forward(r.Context(), request); err != nil {
			if errors.Is(err, application.ErrInvalidRequest) {
				response.Problem(w, http.StatusBadRequest, err.Error())
				return
			}

			logger.ErrorContext(
				r.Context(),
				"email delivery failed",
				"event", "email_delivery_failed",
				"to_count", len(request.Recipients.To),
				"cc_count", len(request.Recipients.CC),
				"bcc_count", len(request.Recipients.BCC),
				"duration", time.Since(started).String(),
				"error", err,
			)
			response.Problem(w, http.StatusBadGateway, "Email delivery failed.")
			return
		}

		logger.InfoContext(
			r.Context(),
			"email accepted by SMTP server",
			"event", "email_delivery_succeeded",
			"to_count", len(request.Recipients.To),
			"cc_count", len(request.Recipients.CC),
			"bcc_count", len(request.Recipients.BCC),
			"duration", time.Since(started).String(),
		)
		w.WriteHeader(http.StatusNoContent)
	}
}

// decodeRequest reads exactly one bounded JSON mail request from the request body.
func decodeRequest(w http.ResponseWriter, r *http.Request) (application.Request, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var request application.Request
	if err := decoder.Decode(&request); err != nil {
		return application.Request{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return application.Request{}, errors.New("multiple JSON values")
		}
		return application.Request{}, err
	}

	return request, nil
}
