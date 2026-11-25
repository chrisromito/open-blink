package deviceimage_test

import (
	"context"
	"devicecapture/internal/config"
	"devicecapture/internal/domain"
	"devicecapture/internal/domain/detection"
	"devicecapture/internal/postgres"
	"devicecapture/internal/postgres/db"
	"devicecapture/internal/postgres/repos"
	"devicecapture/internal/pubsub/deviceimage"
	"devicecapture/internal/testutils"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
	"time"

	"devicecapture/internal/domain/devices"
)

func newTestReceiver() (*deviceimage.Receiver, *domain.Deps, *postgres.AppDb, error) {
	deps, appDb, err := testutils.NewTestDeps("/videos", "/archive")

	detector := detection.MockDetectionService{}
	conf := config.Config{}
	processor := deviceimage.NewDeviceImageProcessor(&conf, deps, detector)
	return deviceimage.NewReceiver(processor), deps, appDb, err
}

func getTestDeviceID(ctx context.Context, queries *db.Queries) (int64, error) {
	td, err := repos.GetTestDevice(ctx, queries)
	if err != nil {
		return int64(0), err
	}
	return td.ID, err
}

func TestReceiverHandle(t *testing.T) {
	a := assert.New(t)

	receiver, deps, appDb, rErr := newTestReceiver()
	a.NoError(rErr)
	a.NotNil(receiver)
	defer func() {
		if appDb != nil {
			appDb.Db.Close()
		}
	}()

	deviceID, dErr := getTestDeviceID(t.Context(), appDb.GetQueries())
	a.NoError(dErr)
	defer func() {
		_ = deps.DeviceRepo.DeleteDevice(t.Context(), deviceID)
	}()

	t.Run("receiver_rejects_topic_payload_device_mismatch", func(t *testing.T) {
		img := devices.DeviceImage{
			ID:       456,
			DeviceID: 999,
		}

		payload, err := deviceimage.Marshal(deviceimage.NewMessage(img))
		err = receiver.Handle(t.Context(), deviceimage.CreatedTopic(deviceID), payload)
		if err == nil {
			t.Fatal("Handle() error = nil, want error")
		}
	})

	t.Run("receiver_handle", func(t *testing.T) {
		tempDir := t.TempDir()
		epoch := time.Now().Unix()
		imagePath := filepath.Join(
			tempDir,
			fmt.Sprintf("%d-%d", deviceID, epoch),
			fmt.Sprintf("output-%d-%d.jpeg", deviceID, 1),
		)
		img, err := createTestImage(t.Context(), deps.ImageRepo, deviceID, imagePath)
		require.NoError(t, err)
		payload, err := deviceimage.Marshal(deviceimage.NewMessage(img))
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}
		err = receiver.Handle(t.Context(), deviceimage.CreatedTopic(deviceID), payload)
		if err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
	})

	t.Run("rejects_invalid_topic", func(t *testing.T) {
		err := receiver.Handle(t.Context(), deviceimage.CreatedTopic(123), []byte("{}"))
		if err == nil {
			t.Fatal("Handle() error = nil, want error")
		}
	})

	t.Run("rejects_invalid_payload", func(t *testing.T) {
		err := receiver.Handle(t.Context(), deviceimage.CreatedTopic(deviceID), []byte("{"))
		if err == nil {
			t.Fatal("Handle() error = nil, want error")
		}
	})
}

func createTestImage(ctx context.Context, repo devices.ImageRepo, deviceID int64, imagePath string) (devices.DeviceImage, error) {
	err := writeTestImage(imagePath)
	if err != nil {
		return devices.DeviceImage{}, err
	}
	// Write to repo and return
	return repo.CreateImage(ctx, devices.CreateImageParams{
		DeviceID:  deviceID,
		ImagePath: imagePath,
	})
}

func writeTestImage(imgPath string) error {
	// 1. Create a new RGBA image with bounds (width: 200, height: 100)
	rect := image.Rect(0, 0, 200, 100)
	img := image.NewRGBA(rect)

	// 2. Fill the image with a solid color (e.g., blue)
	blueColor := color.RGBA{R: 0, G: 100, B: 255, A: 255}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.Set(x, y, blueColor)
		}
	}

	// 3. Create the output file
	if err := os.MkdirAll(filepath.Dir(imgPath), 0o755); err != nil {
		return err
	}

	file, err := os.Create(imgPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 4. Set JPEG quality options (1 to 100, default is 75)
	options := &jpeg.Options{Quality: 5}

	// 5. Encode and write the image to the file
	err = jpeg.Encode(file, img, options)
	if err != nil {
		return err
	}
	return nil
}
