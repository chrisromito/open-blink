package app

import (
	"devicecapture/internal/config"
	"devicecapture/internal/domain"
	"devicecapture/internal/postgres"
	"devicecapture/internal/pubsub"
)

type StartStreamMessage struct {
	DeviceId string `json:"device_id"`
}

type App struct {
	Conf       *config.Config
	MqttClient *pubsub.MqttClient
	Db         *postgres.AppDb
	AppDeps    *domain.Deps
	//CameraService *domain.CameraService
	//DeviceRepo    devices.DeviceRepository
	//HeartbeatRepo devices.HeartbeatRepo
	//DetectionRepo devices.DetectionRepo
}

// NewApp create an App, under the assumption that the MqttClient & AppDb are initialized/connected
func NewApp(conf *config.Config, mqttClient *pubsub.MqttClient, db *postgres.AppDb, deps *domain.Deps) *App {
	return &App{
		Conf:       conf,
		MqttClient: mqttClient,
		Db:         db,
		AppDeps:    deps,
		//CameraService: deps.CameraService,
		//HeartbeatRepo: deps.HeartbeatRepo,
		//DetectionRepo: deps.DetectionRepo,
	}
}

//
//func (a *App) SubscribeToStartStreamTopic(ctx context.Context) error {
//	mc := make(chan mqtt.Message)
//	defer close(mc)
//	topics := []string{"start-stream", "motion-detected"}
//	for _, t := range topics {
//		err := a.MqttClient.Subscribe(t, func(_ mqtt.Client, m mqtt.Message) {
//			log.Print("devicecapture -> subscribe -> pushing message to channel")
//			mc <- m
//		})
//		if err != nil {
//			log.Printf("Error subscribing to topic: %v", err)
//			return err
//		}
//	}
//
//	for {
//		select {
//		case m := <-mc:
//			err := a.ReceiveStartStreamMessage(m)
//			if err != nil {
//
//				log.Printf("Error receiving message: %v", err)
//			}
//		case <-ctx.Done():
//			return nil
//		}
//	}
//}
//
//
//func (a *App) ReceiveStartStreamMessage(m mqtt.Message) error {
//	log.Print("devicecapture.app -> ReceiveStartStreamMessage()")
//	value := m.Payload()
//	var msg StartStreamMessage
//	err := json.Unmarshal(value, &msg)
//	if err != nil {
//		return err
//	}
//	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Second)
//	defer cancel()
//	err = a.CameraService.StartStream(ctx, msg.DeviceID)
//	if err != nil {
//		return err
//	}
//	return nil
//}
