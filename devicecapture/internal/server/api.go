package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"devicecapture/internal/app"
	"devicecapture/internal/domain/history"
	"devicecapture/internal/logger"
)

func GetRecentLabelsHandler(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
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
		w.Header().Set("Content-Type", "application/json")
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
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()
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
		page := 1
		queryPage := r.URL.Query().Get("page")
		if queryPage != "" {
			pageInt, err := strconv.Atoi(queryPage)
			if err != nil {
				logger.Error().Err(err).Str("endpoint", "GetDetectionTimelineHandler").
					Str("param", "page").Send()
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			page = pageInt
		}
		params := history.DetectionTimelineParams{
			DeviceID: deviceID,
			Page:     page,
		}
		ds, dbErr := a.AppDeps.HistoryRepo.GetDetectionTimeline(ctx, params)
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
