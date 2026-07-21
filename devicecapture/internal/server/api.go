package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"devicecapture/internal/app"
	"devicecapture/internal/domain/event"
	"devicecapture/internal/domain/history"
	"devicecapture/internal/logger"
)

func GetRecentLabelsHandler(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		labels, dbErr := a.AppDeps.HistoryRepo.GetRecentLabels(ctx)
		if dbErr != nil {
			logger.Error().
				Msgf("GetRecentLabelsHandler -> dbErr %v", dbErr)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err := json.NewEncoder(w).Encode(labels); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
}

func GetDetectionImagesByLabelHandler(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		labelQuery := []string{"person"}
		if r.URL.Query().Has("label") {
			queryValue := r.URL.Query()["label"]
			if len(labelQuery) > 0 {
				labelQuery = queryValue
			}
		}
		deviceID := int64(0)
		queryDevice := r.URL.Query().Get("device_id")
		if queryDevice != "" {
			idInt, err := strconv.ParseInt(queryDevice, 10, 64)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			deviceID = idInt
		}
		params := history.DetectionWithImageParams{
			Label:    labelQuery,
			DeviceID: deviceID,
		}
		ds, dbErr := a.AppDeps.HistoryRepo.GetDetectionImagesByLabel(ctx, params)
		if dbErr != nil {
			logger.Error().Err(dbErr).Str("endpoint", "GetDetectionImagesByLabelHandler").
				Msg("dbErr")
			http.Error(w, dbErr.Error(), http.StatusInternalServerError)
			return
		}
		if err := json.NewEncoder(w).Encode(ds); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
}

// GetDetectionTimelineHandler Timeline endpoint
func GetDetectionTimelineHandler(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqParams, err := getQueryPageDeviceID(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ds, dbErr := a.AppDeps.HistoryRepo.GetDetectionTimeline(
			r.Context(),
			history.DetectionTimelineParams{
				DeviceID: reqParams.DeviceID,
				Page:     reqParams.Page,
			},
		)
		if dbErr != nil {
			logger.Error().Err(dbErr).Str("endpoint", "GetDetectionTimelineHandler").
				Msg("dbErr")
			http.Error(w, dbErr.Error(), http.StatusInternalServerError)
			return
		}
		if err := json.NewEncoder(w).Encode(ds); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
}

func DetectionEventListHandler(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqParams, err := parseEventParams(r)
		if err != nil {
			logger.Error().Err(err).Str("endpoint", "DetectionEventListHandler").
				Msg("queryParamParseErr")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ds, dbErr := a.AppDeps.EventRepo.GetDetectionEvents(r.Context(), event.QueryParams{
			DeviceID: reqParams.DeviceID,
			Page:     reqParams.Page,
			Start:    reqParams.Start,
			End:      reqParams.End,
		})
		if dbErr != nil {
			logger.Error().Err(dbErr).Str("endpoint", "DetectionEventListHandler").
				Msg("dbErr")
			http.Error(w, dbErr.Error(), http.StatusInternalServerError)
			return
		}
		if err := json.NewEncoder(w).Encode(ds); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
}

func DetectionEventDetailHandler(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		idInt, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		logger.Debug().Str("endpoint", "DetectionEventDetailHandler").
			Str("id", r.PathValue("id")).
			Send()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		eventDetails, dbErr := a.AppDeps.EventRepo.GetDetectionsForEvent(ctx, idInt)
		if dbErr != nil {
			logger.Error().Err(dbErr).
				Str("endpoint", "DetectionEventDetailHandler").
				Str("id", r.PathValue("id")).
				Msg("dbErr")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err = json.NewEncoder(w).Encode(eventDetails); err != nil {
			logger.Error().Err(dbErr).
				Str("endpoint", "DetectionEventDetailHandler").
				Str("id", r.PathValue("id")).
				Msg("dbErr")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
}

type EventParams struct {
	DeviceID int64
	Page     int
	// Start is the minimum created_at value
	Start time.Time
	// End is the (optional) maximum created_at value
	End time.Time
}

func parseEventParams(r *http.Request) (EventParams, error) {
	pID, err := getQueryPageDeviceID(r)
	if err != nil {
		return EventParams{}, err
	}
	// start = now - (7 days)
	start := time.Now().AddDate(0, 0, -7)
	queryStart := r.URL.Query().Get("start")
	if queryStart != "" {
		startInt, sErr := strconv.ParseInt(queryStart, 10, 64)
		if sErr != nil {
			return EventParams{}, sErr
		}
		// Cast the epoch MS to a [time.Time]
		start = time.UnixMilli(startInt)
	}
	end := time.Now()
	queryEnd := r.URL.Query().Get("end")
	if queryEnd != "" {
		endInt, eErr := strconv.ParseInt(queryEnd, 10, 64)
		if eErr != nil {
			return EventParams{}, eErr
		}
		end = time.UnixMilli(endInt)
	}
	return EventParams{DeviceID: pID.DeviceID, Page: pID.Page, Start: start, End: end}, nil
}

type PageAndId struct {
	Page     int
	DeviceID int64
}

func getQueryPageDeviceID(r *http.Request) (PageAndId, error) {
	deviceID := int64(0)
	queryDevice := r.URL.Query().Get("device_id")
	if queryDevice != "" {
		idInt, err := strconv.ParseInt(queryDevice, 10, 64)
		if err != nil {
			//http.Error(w, err.Error(), http.StatusBadRequest)
			return PageAndId{}, err
		}
		deviceID = idInt
	}
	page := 1
	queryPage := r.URL.Query().Get("page")
	if queryPage != "" {
		pageInt, err := strconv.Atoi(queryPage)
		if err != nil {
			logger.Error().Err(err).Str("endpoint", "GetDetectionTimelineHandler").
				Str("param", "page").Send()
			//http.Error(w, err.Error(), http.StatusBadRequest)
			return PageAndId{}, err
		}
		page = pageInt
	}
	return PageAndId{Page: page, DeviceID: deviceID}, nil
}
