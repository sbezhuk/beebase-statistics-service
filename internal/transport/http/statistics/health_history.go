package statistics

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/sbezhuk/beebase-common/httpx"
	appstatistics "github.com/sbezhuk/beebase-statistics-service/internal/application/statistics"
)

const (
	CodeInvalidHiveID            = "invalid_hive_id"
	CodeFromRequired             = "from_required"
	CodeFromInvalid              = "from_invalid"
	CodeToRequired               = "to_required"
	CodeToInvalid                = "to_invalid"
	CodeFromAfterTo              = "from_after_to"
	CodeHistoryRangeTooLong      = "history_range_too_long"
	CodeIntervalUnsupported      = "interval_unsupported"
	CodeHealthHistoryProRequired = "health_history_pro_required"
	maxHealthHistoryPoints       = 365
	dateLayout                   = "2006-01-02"
)

func (h *Handler) HealthHistory(w http.ResponseWriter, r *http.Request) {
	token, ok := h.requireToken(w, r)
	if !ok {
		return
	}
	hiveID, err := uuid.Parse(chi.URLParam(r, "hiveId"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, CodeInvalidHiveID, "hive id must be a valid UUID")
		return
	}
	from, to, fields := parseHealthHistoryRange(r)
	if len(fields) > 0 {
		httpx.WriteValidationError(w, fields)
		return
	}
	result, err := h.service.HealthHistory(r.Context(), token, hiveID, from, to)
	if err != nil {
		switch {
		case errors.Is(err, appstatistics.ErrHiveNotFound):
			httpx.WriteError(w, http.StatusNotFound, "hive_not_found", "hive not found")
		case errors.Is(err, appstatistics.ErrHealthHistoryProRequired):
			httpx.WriteError(w, http.StatusForbidden, CodeHealthHistoryProRequired, "colony health history requires pro")
		default:
			httpx.WriteInternalError(w, h.log, err)
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newHealthHistoryResponse(result))
}

func parseHealthHistoryRange(r *http.Request) (from, to time.Time, fields map[string]string) {
	fields = map[string]string{}
	rawFrom, rawTo := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	if rawFrom == "" {
		fields["from"] = CodeFromRequired
	} else if parsed, err := time.Parse(dateLayout, rawFrom); err != nil {
		fields["from"] = CodeFromInvalid
	} else {
		from = parsed
	}
	if rawTo == "" {
		fields["to"] = CodeToRequired
	} else if parsed, err := time.Parse(dateLayout, rawTo); err != nil {
		fields["to"] = CodeToInvalid
	} else {
		to = parsed
	}
	if interval := r.URL.Query().Get("interval"); interval != "" && !strings.EqualFold(interval, "day") {
		fields["interval"] = CodeIntervalUnsupported
	}
	if len(fields) > 0 {
		return time.Time{}, time.Time{}, fields
	}
	if from.After(to) {
		fields["to"] = CodeFromAfterTo
		return time.Time{}, time.Time{}, fields
	}
	if int(to.Sub(from).Hours()/24)+1 > maxHealthHistoryPoints {
		fields["to"] = CodeHistoryRangeTooLong
		return time.Time{}, time.Time{}, fields
	}
	return from, to, nil
}
