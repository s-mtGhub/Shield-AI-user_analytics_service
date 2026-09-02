package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"user-analytics-service/internal/service"
)

// Handler wires HTTP endpoints to the service layer.
type Handler struct {
	ingestion *service.IngestionService
	query     *service.QueryService
}

// NewHandler builds a Handler backed by the given services.
func NewHandler(ingestion *service.IngestionService, query *service.QueryService) *Handler {
	return &Handler{ingestion: ingestion, query: query}
}

// NewRouter registers all routes on a fresh http.ServeMux.
func NewRouter(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc(http.MethodPost+" "+pathLoginEvent, h.RecordLogin)
	mux.HandleFunc(http.MethodGet+" "+pathDailyActiveUsers, h.DailyActiveUsers)
	mux.HandleFunc(http.MethodGet+" "+pathMonthlyActiveUsers, h.MonthlyActiveUsers)
	return mux
}

func (h *Handler) RecordLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, msgInvalidJSONBody)
		return
	}

	ts := time.Now().UTC()
	if req.Timestamp != "" {
		parsed, err := time.Parse(time.RFC3339, req.Timestamp)
		if err != nil {
			writeError(w, http.StatusBadRequest, msgInvalidTimestamp)
			return
		}

		if parsed.After(ts) {
			writeError(w, http.StatusBadRequest, msgFutureTimestamp)
			return
		}
		ts = parsed
	}

	if err := h.ingestion.RecordLogin(r.Context(), req.UserID, ts); err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, msgRecordLoginFailed)
		return
	}

	writeJSON(w, http.StatusCreated, LoginEventResponse{
		UserID:    req.UserID,
		Timestamp: ts.Format(time.RFC3339),
	})
}

func (h *Handler) DailyActiveUsers(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get(queryParamDate)
	if date == "" {
		writeError(w, http.StatusBadRequest, msgMissingDateParam)
		return
	}

	count, err := h.query.DailyActiveUsers(r.Context(), date)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, msgDailyActiveUsersFail)
		return
	}

	writeJSON(w, http.StatusOK, DailyActiveUsersResponse{Date: date, UniqueActiveUsers: count})
}

func (h *Handler) MonthlyActiveUsers(w http.ResponseWriter, r *http.Request) {
	month := r.URL.Query().Get(queryParamMonth)
	if month == "" {
		writeError(w, http.StatusBadRequest, msgMissingMonthParam)
		return
	}

	count, err := h.query.MonthlyActiveUsers(r.Context(), month)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, msgMonthlyActiveUsersFail)
		return
	}

	writeJSON(w, http.StatusOK, MonthlyActiveUsersResponse{Month: month, UniqueActiveUsers: count})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}
