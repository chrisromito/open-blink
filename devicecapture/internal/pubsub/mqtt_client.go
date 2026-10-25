package pubsub

import (
	"errors"
	"fmt"
	"time"

	"devicecapture/internal/config"
	"devicecapture/internal/logger"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MessageHandler func(topic string, payload []byte) error

type MqttClient struct {
	client   mqtt.Client
	qos      byte
	retained bool
	opts     MqttClientOptions
}

type MqttClientOptions struct {
	BrokerURL      string
	ClientID       string
	Username       string
	Password       string
	QOS            byte
	Retained       bool
	ConnectTimeout time.Duration
}

//func NewMqttClient(client mqtt.Client) *MqttClient {
//	return &MqttClient{
//		client:   client,
//		qos:      defaultQOS,
//		retained: defaultRetained,
//	}
//}

func NewMqttClientOptions(conf *config.Config, clientID string) MqttClientOptions {
	return MqttClientOptions{
		BrokerURL: conf.MqttHost,
		ClientID:  clientID,
		Username:  conf.MqttUser,
		Password:  conf.MqttPassword,
	}
}

func (o *MqttClientOptions) Copy(clientID string) MqttClientOptions {
	return MqttClientOptions{
		ClientID:       clientID,
		BrokerURL:      o.BrokerURL,
		Username:       o.Username,
		Password:       o.Password,
		QOS:            o.QOS,
		Retained:       o.Retained,
		ConnectTimeout: o.ConnectTimeout,
	}
}

func NewConnectedMqttClient(opts MqttClientOptions) (*MqttClient, error) {
	if opts.BrokerURL == "" {
		return nil, errors.New("mqtt broker url is required")
	}
	if opts.ClientID == "" {
		return nil, errors.New("mqtt client id is required")
	}

	connectTimeout := opts.ConnectTimeout
	if connectTimeout == 0 {
		connectTimeout = 10 * time.Second
	}

	clientOptions := mqtt.NewClientOptions().
		AddBroker(opts.BrokerURL).
		SetClientID(opts.ClientID).
		SetConnectTimeout(connectTimeout)

	if opts.Username != "" {
		clientOptions.SetUsername(opts.Username)
	}
	if opts.Password != "" {
		clientOptions.SetPassword(opts.Password)
	}

	client := mqtt.NewClient(clientOptions)
	token := client.Connect()
	if !token.WaitTimeout(connectTimeout) {
		return nil, fmt.Errorf("mqtt connect timed out after %s", connectTimeout)
	}
	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("mqtt connect: %w", err)
	}

	return &MqttClient{
		client:   client,
		qos:      opts.QOS,
		retained: opts.Retained,
		opts:     opts,
	}, nil
}

func (c *MqttClient) Copy(clientID string) (*MqttClient, error) {
	copyOpts := c.opts.Copy(clientID)
	return NewConnectedMqttClient(copyOpts)
}

func (c *MqttClient) Publish(topic string, payload any) error {
	if c == nil || c.client == nil {
		return errors.New("mqtt client is nil")
	}
	if topic == "" {
		return errors.New("mqtt topic is required")
	}

	token := c.client.Publish(topic, c.qos, c.retained, payload)
	token.Wait()

	if err := token.Error(); err != nil {
		return fmt.Errorf("mqtt publish to %q: %w", topic, err)
	}

	return nil
}

func (c *MqttClient) Subscribe(topic string, handler MessageHandler) error {
	if c == nil || c.client == nil {
		return errors.New("mqtt client is nil")
	}
	if topic == "" {
		return errors.New("mqtt topic is required")
	}
	if handler == nil {
		return errors.New("mqtt message handler is required")
	}

	token := c.client.Subscribe(topic, c.qos, func(_ mqtt.Client, msg mqtt.Message) {
		payload := append([]byte(nil), msg.Payload()...)

		if err := handler(msg.Topic(), payload); err != nil {
			logger.Error().
				Str("topic", msg.Topic()).
				Err(err).
				Msg("mqtt subscription handler failed")
		}
	})
	token.Wait()

	if err := token.Error(); err != nil {
		return fmt.Errorf("mqtt subscribe to %q: %w", topic, err)
	}

	return nil
}

func (c *MqttClient) Disconnect(quiesce uint) {
	if c == nil || c.client == nil {
		return
	}

	c.client.Disconnect(quiesce)
}
