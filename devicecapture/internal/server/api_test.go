package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"devicecapture/internal/app"
	deps "devicecapture/internal/domain"
	"devicecapture/internal/domain/event"
	"devicecapture/internal/domain/history"
)

var errMockRepo = errors.New("mock repo error")

func newMockTestApp(t *testing.T) *app.App {
	t.Helper()

	return &app.App{
		AppDeps: deps.NewMockDeps(),
	}
}

func assertStatus(t *testing.T, rr *httptest.ResponseRecorder, want int) {
	t.Helper()

	if rr.Code != want {
		t.Fatalf("status = %d, want %d, body: %s", rr.Code, want, rr.Body.String())
	}
}

func decodeJSONBody[T any](t *testing.T, rr *httptest.ResponseRecorder) T {
	t.Helper()

	var got T
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("response body is not valid JSON: %v\nbody: %s", err, rr.Body.String())
	}
	return got
}

func assertStringSlicesEqual(t *testing.T, got, want []string) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("slice mismatch: got %#v, want %#v", got, want)
	}
}

func assertTimeBetween(t *testing.T, name string, got time.Time, lower time.Time, upper time.Time) {
	t.Helper()

	if got.Before(lower) || got.After(upper) {
		t.Fatalf("%s = %s, want between %s and %s", name, got, lower, upper)
	}
}

type capturingHistoryRepo struct {
	delegate history.DetectionHistoryRepo

	recentLabelsErr error
	imagesErr       error
	timelineErr     error

	recentLabelsCalled bool
	imagesCalled       bool
	timelineCalled     bool

	gotImagesParams   history.DetectionWithImageParams
	gotTimelineParams history.DetectionTimelineParams
}

func (r *capturingHistoryRepo) GetRecentLabels(ctx context.Context) ([]string, error) {
	r.recentLabelsCalled = true
	if r.recentLabelsErr != nil {
		return nil, r.recentLabelsErr
	}
	return r.delegate.GetRecentLabels(ctx)
}

func (r *capturingHistoryRepo) GetDetectionImagesByLabel(
	ctx context.Context,
	params history.DetectionWithImageParams,
) ([]history.DetectionWithImage, error) {
	r.imagesCalled = true
	r.gotImagesParams = params
	if r.imagesErr != nil {
		return nil, r.imagesErr
	}
	return r.delegate.GetDetectionImagesByLabel(ctx, params)
}

func (r *capturingHistoryRepo) GetDetectionTimeline(
	ctx context.Context,
	params history.DetectionTimelineParams,
) ([]history.DetectionEvent, error) {
	r.timelineCalled = true
	r.gotTimelineParams = params
	if r.timelineErr != nil {
		return nil, r.timelineErr
	}
	return r.delegate.GetDetectionTimeline(ctx, params)
}

type capturingEventRepo struct {
	delegate event.DetectionEventRepo

	listErr   error
	detailErr error

	listCalled   bool
	detailCalled bool

	gotQueryParams event.QueryParams
	gotEventID     int64
}

func (r *capturingEventRepo) GetEvent(ctx context.Context, id int64) (event.DetectionEvent, error) {
	return r.delegate.GetEvent(ctx, id)
}

func (r *capturingEventRepo) GetLatest(
	ctx context.Context,
	deviceID int64,
) (event.DetectionEvent, error) {
	return r.delegate.GetLatest(ctx, deviceID)
}

func (r *capturingEventRepo) StartEvent(
	ctx context.Context,
	deviceID int64,
	labels []string,
) (event.DetectionEvent, error) {
	return r.delegate.StartEvent(ctx, deviceID, labels)
}

func (r *capturingEventRepo) EndEvent(
	ctx context.Context,
	e event.DetectionEvent,
) (event.DetectionEvent, error) {
	return r.delegate.EndEvent(ctx, e)
}

func (r *capturingEventRepo) GetDetectionEvents(
	ctx context.Context,
	p event.QueryParams,
) ([]event.DetectionEvent, error) {
	r.listCalled = true
	r.gotQueryParams = p
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.delegate.GetDetectionEvents(ctx, p)
}

func (r *capturingEventRepo) GetDetectionsForEvent(
	ctx context.Context,
	eventID int64,
) (event.DetectionDetail, error) {
	r.detailCalled = true
	r.gotEventID = eventID
	if r.detailErr != nil {
		return event.DetectionDetail{}, r.detailErr
	}
	return r.delegate.GetDetectionsForEvent(ctx, eventID)
}

func TestGetRecentLabelsHandler(t *testing.T) {
	a := newMockTestApp(t)

	historyRepo := &capturingHistoryRepo{delegate: a.AppDeps.HistoryRepo}
	a.AppDeps.HistoryRepo = historyRepo

	req := httptest.NewRequest(http.MethodGet, "/api/labels/recent", nil)
	rr := httptest.NewRecorder()

	GetRecentLabelsHandler(a).ServeHTTP(rr, req)

	assertStatus(t, rr, http.StatusOK)

	got := decodeJSONBody[[]string](t, rr)
	assertStringSlicesEqual(t, got, []string{"car", "truck", "person", "bicycle", "dog"})

	if !historyRepo.recentLabelsCalled {
		t.Fatal("expected GetRecentLabels to be called")
	}
}

func TestGetRecentLabelsHandlerRepoError(t *testing.T) {
	a := newMockTestApp(t)

	historyRepo := &capturingHistoryRepo{
		delegate:        a.AppDeps.HistoryRepo,
		recentLabelsErr: errMockRepo,
	}
	a.AppDeps.HistoryRepo = historyRepo

	req := httptest.NewRequest(http.MethodGet, "/api/labels/recent", nil)
	rr := httptest.NewRecorder()

	GetRecentLabelsHandler(a).ServeHTTP(rr, req)

	assertStatus(t, rr, http.StatusInternalServerError)

	if !historyRepo.recentLabelsCalled {
		t.Fatal("expected GetRecentLabels to be called")
	}
}

func TestGetDetectionImagesByLabelHandler(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		wantParams history.DetectionWithImageParams
	}{
		{
			name:   "defaults to person label and all devices",
			target: "/api/detections/images",
			wantParams: history.DetectionWithImageParams{
				Label:    []string{"person"},
				DeviceID: 0,
			},
		},
		{
			name:   "uses explicit label and device id",
			target: "/api/detections/images?label=car&device_id=2",
			wantParams: history.DetectionWithImageParams{
				Label:    []string{"car"},
				DeviceID: 2,
			},
		},
		{
			name:   "supports multiple labels",
			target: "/api/detections/images?label=person&label=car&device_id=2",
			wantParams: history.DetectionWithImageParams{
				Label:    []string{"person", "car"},
				DeviceID: 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newMockTestApp(t)

			historyRepo := &capturingHistoryRepo{delegate: a.AppDeps.HistoryRepo}
			a.AppDeps.HistoryRepo = historyRepo

			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			rr := httptest.NewRecorder()

			GetDetectionImagesByLabelHandler(a).ServeHTTP(rr, req)

			assertStatus(t, rr, http.StatusOK)

			if !historyRepo.imagesCalled {
				t.Fatal("expected GetDetectionImagesByLabel to be called")
			}

			assertStringSlicesEqual(t, historyRepo.gotImagesParams.Label, tt.wantParams.Label)
			if historyRepo.gotImagesParams.DeviceID != tt.wantParams.DeviceID {
				t.Fatalf(
					"DeviceID = %d, want %d",
					historyRepo.gotImagesParams.DeviceID,
					tt.wantParams.DeviceID,
				)
			}

			got := decodeJSONBody[[]history.DetectionWithImage](t, rr)
			if len(got) != 2 {
				t.Fatalf("len(response) = %d, want 2", len(got))
			}

			if got[0].Label != "car" {
				t.Fatalf("first detection label = %q, want %q", got[0].Label, "car")
			}

			if got[1].Label != "person" {
				t.Fatalf("second detection label = %q, want %q", got[1].Label, "person")
			}
		})
	}
}

func TestGetDetectionImagesByLabelHandlerBadRequest(t *testing.T) {
	a := newMockTestApp(t)

	historyRepo := &capturingHistoryRepo{delegate: a.AppDeps.HistoryRepo}
	a.AppDeps.HistoryRepo = historyRepo

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/detections/images?device_id=bad-device-id",
		nil,
	)
	rr := httptest.NewRecorder()

	GetDetectionImagesByLabelHandler(a).ServeHTTP(rr, req)

	assertStatus(t, rr, http.StatusBadRequest)

	if historyRepo.imagesCalled {
		t.Fatal("expected GetDetectionImagesByLabel not to be called for invalid device_id")
	}
}

func TestGetDetectionImagesByLabelHandlerRepoError(t *testing.T) {
	a := newMockTestApp(t)

	historyRepo := &capturingHistoryRepo{
		delegate:  a.AppDeps.HistoryRepo,
		imagesErr: errMockRepo,
	}
	a.AppDeps.HistoryRepo = historyRepo

	req := httptest.NewRequest(http.MethodGet, "/api/detections/images?label=car", nil)
	rr := httptest.NewRecorder()

	GetDetectionImagesByLabelHandler(a).ServeHTTP(rr, req)

	assertStatus(t, rr, http.StatusInternalServerError)

	if !historyRepo.imagesCalled {
		t.Fatal("expected GetDetectionImagesByLabel to be called")
	}
}

func TestGetDetectionTimelineHandler(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		wantParams history.DetectionTimelineParams
	}{
		{
			name:   "defaults to page one and all devices",
			target: "/api/detections/timeline",
			wantParams: history.DetectionTimelineParams{
				DeviceID: 0,
				Page:     1,
			},
		},
		{
			name:   "uses explicit page and device id",
			target: "/api/detections/timeline?page=3&device_id=2",
			wantParams: history.DetectionTimelineParams{
				DeviceID: 2,
				Page:     3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newMockTestApp(t)

			historyRepo := &capturingHistoryRepo{delegate: a.AppDeps.HistoryRepo}
			a.AppDeps.HistoryRepo = historyRepo

			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			rr := httptest.NewRecorder()

			GetDetectionTimelineHandler(a).ServeHTTP(rr, req)

			assertStatus(t, rr, http.StatusOK)

			if !historyRepo.timelineCalled {
				t.Fatal("expected GetDetectionTimeline to be called")
			}

			if historyRepo.gotTimelineParams != tt.wantParams {
				t.Fatalf(
					"timeline params = %+v, want %+v",
					historyRepo.gotTimelineParams,
					tt.wantParams,
				)
			}

			got := decodeJSONBody[[]history.DetectionEvent](t, rr)
			if len(got) != 2 {
				t.Fatalf("len(response) = %d, want 2", len(got))
			}

			if got[0].DeviceID != 1 {
				t.Fatalf("first event DeviceID = %d, want 1", got[0].DeviceID)
			}

			if got[0].Detections[0].Label != "car" {
				t.Fatalf("first event detection label = %q, want car", got[0].Detections[0].Label)
			}
		})
	}
}

func TestGetDetectionTimelineHandlerBadRequest(t *testing.T) {
	tests := []struct {
		name   string
		target string
	}{
		{
			name:   "invalid page",
			target: "/api/detections/timeline?page=bad-page",
		},
		{
			name:   "invalid device id",
			target: "/api/detections/timeline?device_id=bad-device-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newMockTestApp(t)

			historyRepo := &capturingHistoryRepo{delegate: a.AppDeps.HistoryRepo}
			a.AppDeps.HistoryRepo = historyRepo

			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			rr := httptest.NewRecorder()

			GetDetectionTimelineHandler(a).ServeHTTP(rr, req)

			assertStatus(t, rr, http.StatusBadRequest)

			if historyRepo.timelineCalled {
				t.Fatal("expected GetDetectionTimeline not to be called for invalid query params")
			}
		})
	}
}

func TestGetDetectionTimelineHandlerRepoError(t *testing.T) {
	a := newMockTestApp(t)

	historyRepo := &capturingHistoryRepo{
		delegate:    a.AppDeps.HistoryRepo,
		timelineErr: errMockRepo,
	}
	a.AppDeps.HistoryRepo = historyRepo

	req := httptest.NewRequest(http.MethodGet, "/api/detections/timeline?page=1&device_id=2", nil)
	rr := httptest.NewRecorder()

	GetDetectionTimelineHandler(a).ServeHTTP(rr, req)

	assertStatus(t, rr, http.StatusInternalServerError)

	if !historyRepo.timelineCalled {
		t.Fatal("expected GetDetectionTimeline to be called")
	}
}

func TestDetectionEventListHandler(t *testing.T) {
	const (
		startMS = int64(1700000000000)
		endMS   = int64(1700100000000)
	)

	a := newMockTestApp(t)

	eventRepo := &capturingEventRepo{delegate: a.AppDeps.EventRepo}
	a.AppDeps.EventRepo = eventRepo

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/events?page=3&device_id=2&start=1700000000000&end=1700100000000",
		nil,
	)
	rr := httptest.NewRecorder()

	DetectionEventListHandler(a).ServeHTTP(rr, req)

	assertStatus(t, rr, http.StatusOK)

	if !eventRepo.listCalled {
		t.Fatal("expected GetDetectionEvents to be called")
	}

	wantParams := event.QueryParams{
		DeviceID: 2,
		Page:     3,
		Start:    time.UnixMilli(startMS),
		End:      time.UnixMilli(endMS),
	}

	if eventRepo.gotQueryParams.DeviceID != wantParams.DeviceID {
		t.Fatalf(
			"DeviceID = %d, want %d",
			eventRepo.gotQueryParams.DeviceID,
			wantParams.DeviceID,
		)
	}

	if eventRepo.gotQueryParams.Page != wantParams.Page {
		t.Fatalf("Page = %d, want %d", eventRepo.gotQueryParams.Page, wantParams.Page)
	}

	if !eventRepo.gotQueryParams.Start.Equal(wantParams.Start) {
		t.Fatalf(
			"Start = %s, want %s",
			eventRepo.gotQueryParams.Start,
			wantParams.Start,
		)
	}

	if !eventRepo.gotQueryParams.End.Equal(wantParams.End) {
		t.Fatalf("End = %s, want %s", eventRepo.gotQueryParams.End, wantParams.End)
	}

	got := decodeJSONBody[[]event.DetectionEvent](t, rr)
	if len(got) != 10 {
		t.Fatalf("len(response) = %d, want 10", len(got))
	}

	for _, evt := range got {
		if evt.DeviceID != 2 {
			t.Fatalf("event DeviceID = %d, want 2", evt.DeviceID)
		}

		if evt.State != int(event.Ended) {
			t.Fatalf("event State = %d, want %d", evt.State, int(event.Ended))
		}
	}
}

func TestDetectionEventListHandlerDefaults(t *testing.T) {
	a := newMockTestApp(t)

	eventRepo := &capturingEventRepo{delegate: a.AppDeps.EventRepo}
	a.AppDeps.EventRepo = eventRepo

	before := time.Now()
	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	rr := httptest.NewRecorder()

	DetectionEventListHandler(a).ServeHTTP(rr, req)
	after := time.Now()

	assertStatus(t, rr, http.StatusOK)

	if !eventRepo.listCalled {
		t.Fatal("expected GetDetectionEvents to be called")
	}

	if eventRepo.gotQueryParams.DeviceID != 0 {
		t.Fatalf("DeviceID = %d, want 0", eventRepo.gotQueryParams.DeviceID)
	}

	if eventRepo.gotQueryParams.Page != 1 {
		t.Fatalf("Page = %d, want 1", eventRepo.gotQueryParams.Page)
	}

	assertTimeBetween(
		t,
		"Start",
		eventRepo.gotQueryParams.Start,
		before.AddDate(0, 0, -7),
		after.AddDate(0, 0, -7),
	)

	assertTimeBetween(t, "End", eventRepo.gotQueryParams.End, before, after)

	got := decodeJSONBody[[]event.DetectionEvent](t, rr)
	if len(got) != 0 {
		t.Fatalf("len(response) = %d, want 0 for default DeviceID", len(got))
	}
}

func TestDetectionEventListHandlerBadRequest(t *testing.T) {
	tests := []struct {
		name   string
		target string
	}{
		{
			name:   "invalid page",
			target: "/api/events?page=bad-page",
		},
		{
			name:   "invalid device id",
			target: "/api/events?device_id=bad-device-id",
		},
		{
			name:   "invalid start",
			target: "/api/events?start=bad-start",
		},
		{
			name:   "invalid end",
			target: "/api/events?end=bad-end",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newMockTestApp(t)

			eventRepo := &capturingEventRepo{delegate: a.AppDeps.EventRepo}
			a.AppDeps.EventRepo = eventRepo

			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			rr := httptest.NewRecorder()

			DetectionEventListHandler(a).ServeHTTP(rr, req)

			assertStatus(t, rr, http.StatusBadRequest)

			if eventRepo.listCalled {
				t.Fatal("expected GetDetectionEvents not to be called for invalid query params")
			}
		})
	}
}

func TestDetectionEventListHandlerRepoError(t *testing.T) {
	a := newMockTestApp(t)

	eventRepo := &capturingEventRepo{
		delegate: a.AppDeps.EventRepo,
		listErr:  errMockRepo,
	}
	a.AppDeps.EventRepo = eventRepo

	req := httptest.NewRequest(http.MethodGet, "/api/events?page=1&device_id=2", nil)
	rr := httptest.NewRecorder()

	DetectionEventListHandler(a).ServeHTTP(rr, req)

	assertStatus(t, rr, http.StatusInternalServerError)

	if !eventRepo.listCalled {
		t.Fatal("expected GetDetectionEvents to be called")
	}
}

func TestDetectionEventDetailHandler(t *testing.T) {
	const eventID = int64(18)

	a := newMockTestApp(t)

	eventRepo := &capturingEventRepo{delegate: a.AppDeps.EventRepo}
	a.AppDeps.EventRepo = eventRepo

	req := httptest.NewRequest(http.MethodGet, "/api/events/18", nil)
	req.SetPathValue("id", "18")
	rr := httptest.NewRecorder()

	DetectionEventDetailHandler(a).ServeHTTP(rr, req)

	assertStatus(t, rr, http.StatusOK)

	if !eventRepo.detailCalled {
		t.Fatal("expected GetDetectionsForEvent to be called")
	}

	if eventRepo.gotEventID != eventID {
		t.Fatalf("eventID = %d, want %d", eventRepo.gotEventID, eventID)
	}

	got := decodeJSONBody[event.DetectionDetail](t, rr)
	if got.ID != eventID {
		t.Fatalf("detail ID = %d, want %d", got.ID, eventID)
	}
}

func TestDetectionEventDetailHandlerBadRequest(t *testing.T) {
	a := newMockTestApp(t)

	eventRepo := &capturingEventRepo{delegate: a.AppDeps.EventRepo}
	a.AppDeps.EventRepo = eventRepo

	req := httptest.NewRequest(http.MethodGet, "/api/events/bad-event-id", nil)
	req.SetPathValue("id", "bad-event-id")
	rr := httptest.NewRecorder()

	DetectionEventDetailHandler(a).ServeHTTP(rr, req)

	assertStatus(t, rr, http.StatusBadRequest)

	if eventRepo.detailCalled {
		t.Fatal("expected GetDetectionsForEvent not to be called for invalid path id")
	}
}

func TestGetQueryPageDeviceID(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		want    PageAndId
		wantErr bool
	}{
		{
			name:   "defaults",
			target: "/api/test",
			want: PageAndId{
				Page:     1,
				DeviceID: 0,
			},
		},
		{
			name:   "explicit values",
			target: "/api/test?page=5&device_id=42",
			want: PageAndId{
				Page:     5,
				DeviceID: 42,
			},
		},
		{
			name:    "invalid page",
			target:  "/api/test?page=bad-page",
			wantErr: true,
		},
		{
			name:    "invalid device id",
			target:  "/api/test?device_id=bad-device-id",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.target, nil)

			got, err := getQueryPageDeviceID(req)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("getQueryPageDeviceID() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseEventParams(t *testing.T) {
	const (
		startMS = int64(1700000000000)
		endMS   = int64(1700100000000)
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/events?page=4&device_id=44&start=1700000000000&end=1700100000000",
		nil,
	)

	got, err := parseEventParams(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.DeviceID != 44 {
		t.Fatalf("DeviceID = %d, want 44", got.DeviceID)
	}

	if got.Page != 4 {
		t.Fatalf("Page = %d, want 4", got.Page)
	}

	if !got.Start.Equal(time.UnixMilli(startMS)) {
		t.Fatalf("Start = %s, want %s", got.Start, time.UnixMilli(startMS))
	}

	if !got.End.Equal(time.UnixMilli(endMS)) {
		t.Fatalf("End = %s, want %s", got.End, time.UnixMilli(endMS))
	}
}

func TestParseEventParamsDefaults(t *testing.T) {
	before := time.Now()
	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)

	got, err := parseEventParams(req)
	after := time.Now()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.DeviceID != 0 {
		t.Fatalf("DeviceID = %d, want 0", got.DeviceID)
	}

	if got.Page != 1 {
		t.Fatalf("Page = %d, want 1", got.Page)
	}

	assertTimeBetween(
		t,
		"Start",
		got.Start,
		before.AddDate(0, 0, -7),
		after.AddDate(0, 0, -7),
	)

	assertTimeBetween(t, "End", got.End, before, after)
}

func TestParseEventParamsInvalidQuery(t *testing.T) {
	tests := []struct {
		name   string
		target string
	}{
		{
			name:   "invalid page",
			target: "/api/events?page=bad-page",
		},
		{
			name:   "invalid device id",
			target: "/api/events?device_id=bad-device-id",
		},
		{
			name:   "invalid start",
			target: "/api/events?start=bad-start",
		},
		{
			name:   "invalid end",
			target: "/api/events?end=bad-end",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.target, nil)

			_, err := parseEventParams(req)

			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}
