package device

import (
	"context"
	"devicecapture/internal/device/devices"
	"devicecapture/internal/device/receiver"
	"testing"
	"time"
)

// CameraService Tests
// -----------------------------
// ------------------------------

func TestCameraService_Start(t *testing.T) {
	dr := devices.NewMockRepo()
	fr := receiver.NewMockFrameRepo()
	svc := NewCameraService(dr, fr)

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
			deviceId: "-1",
			wantErr:  true,
			name:     "invalid devices return errors",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
			defer cancel()
			err := svc.StartStream(ctx, test.deviceId)
			if err != nil {
				t.Error(err)
			}
		})
	}
}

// CameraDevice Tests
// -----------------------------
// ------------------------------

func TestCameraDevice_Start(t *testing.T) {
	fr := receiver.NewMockFrameRepo()
	d := devices.GetMockDevice()
	cam := NewCameraDevice(d, fr)
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	go func() {
		err := cam.Start(ctx)
		if err != nil {
			t.Error(err)
		}
		select {
		case <-ctx.Done():
			return
		}
	}()
	<-ctx.Done()
	session := fr.GetSession()
	if session == nil {
		t.Error("starting the camera device sets the session")
	}
	frame := fr.GetFrame()
	if frame == nil {
		t.Error("successful starts create frame streams")
	}
	if cam.StartedAt == 0 {
		t.Error("camera field StartedAt was not set")
	}
}

func TestCameraDevice_Start_Fail(t *testing.T) {
	fr := receiver.NewMockFrameRepo()

	dFail := devices.GetTestFailDevice()
	cam := NewCameraDevice(dFail, fr)
	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
	defer cancel()
	go func() {
		err := cam.Start(ctx)
		if err != nil {
			t.Error(err)
		}
		select {
		case <-ctx.Done():
			return
		}
	}()
	<-ctx.Done()
	session := fr.GetSession()
	if session == nil {
		t.Error("starting the camera device sets the session")
	}
	frame := fr.GetFrame()
	if frame != nil {
		t.Error("failures do not create frame streams")
	}
}

func TestCameraDevice_Stop_EndsSession(t *testing.T) {
	fr := receiver.NewMockFrameRepo()
	d := devices.GetMockDevice()
	cam := NewCameraDevice(d, fr)
	fr.Running = true
	cam.Stop()
	if fr.Running {
		t.Error("camera did not update the repository when it stopped")
	}
}
