package camera

import (
	"context"
	"devicecapture/internal/config"
	"devicecapture/internal/domain"
	"devicecapture/internal/domain/detection"
	"devicecapture/internal/pubsub"
	"testing"
	"time"
)

// CameraService Tests
// -----------------------------
// ------------------------------

func TestCameraService_Start(t *testing.T) {
	deps := domain.NewMockDeps()
	detect := detection.MockDetectionService{}
	conf := config.Config{
		MqttHost:            "tcp://0.0.0.0:1883",
		DbUrl:               "postgres://postgres:postgres@localhost:5432/openblink",
		VideoPath:           "",
		DetectionServiceUrl: "",
	}
	client := &pubsub.MqttClient{}
	svc := NewCameraService(&conf, deps, detect, client)

	tests := []struct {
		deviceId string
		wantErr  bool
		name     string
	}{
		{
			deviceId: "1",
			wantErr:  false,
			name:     "valid devices do not return errors",
		},
		{
			deviceId: "-1000",
			wantErr:  true,
			name:     "invalid devices return errors",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
			defer cancel()
			t.Logf("starting stream for device %s", test.deviceId)
			sesh, err := svc.StartStream(ctx, test.deviceId)
			// if we expect an error, make sure we get an error
			if test.wantErr && err == nil {
				t.Logf("we expected an error, but didnt get one for deviceId %s", test.deviceId)
				t.Logf("    we got %v instead", sesh)
				t.Error(err)
			}
			// make sure we didn't get an error when we expect one
			if (!test.wantErr) && err != nil {
				t.Error(err)
			}
		})
	}
}
