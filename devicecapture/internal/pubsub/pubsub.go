package pubsub

import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
)

func BrokerHelper(cId string, broker string) (MqttClient, error) {
	urls := []string{broker, "localhost", "0.0.0.0", "mosquitto", "host.docker.internal"}
	//brokers := []string{broker, "0.0.0.0:1883", "host.docker.internal:1883"}
	for _, url := range urls {
		b := fmt.Sprintf("%s:1883", url)
		c, err := NewDeviceClient(cId, b)
		if err == nil {
			return c, nil
		}

		log.Printf("Error connecting to broker: %s", b)
		log.Printf("Error: %v", err)
		// Return the error if this is the last broker
		if b == urls[len(urls)-1] {
			return c, err
		}
	}
	return MqttClient{
		Client: nil,
		opts:   nil,
	}, nil
}

func NewDeviceClient(cId string, broker string) (MqttClient, error) {
	opt := ClientOptions{
		ClientID: cId,
		Broker:   broker,
	}
	c := MqttClient{
		opts: &opt,
	}
	err := c.Connect()
	if err != nil {
		return c, err
	}
	return c, nil
}

type ClientOptions struct {
	Broker   string
	ClientID string
}

type MqttClient struct {
	Client mqtt.Client
	opts   *ClientOptions
	topics []string
}

func (m *MqttClient) Valid() bool {
	return m.Client != nil
}

func (m *MqttClient) CopyWithClientId(clientId string) MqttClient {
	opts := ClientOptions{
		ClientID: clientId,
		Broker:   m.opts.Broker,
	}
	return MqttClient{
		opts: &opts,
	}
}

func (m *MqttClient) PublishImage(img []byte, deviceId string) error {
	if m.Client == nil {
		return fmt.Errorf("client not connected")
	}
	topic := fmt.Sprintf("image/%s", deviceId)
	log.Printf("Publishing image to topic: %s", topic)
	err := m.Publish(topic, img)
	if err != nil {
		return err
	}
	log.Printf("Published image to topic: %s", topic)
	return nil
}

func (m *MqttClient) Connect() error {
	if m.Client != nil {
		return fmt.Errorf("client already connected")
	}
	// This method creates some default options for us, most notably it sets the auto reconnect option to be true, and the default port to `1883`. Auto reconnect is really useful in IOT applications as the internet connection may not always be extremely strong.
	mqOptions := mqtt.NewClientOptions()
	mqOptions.AddBroker(m.opts.Broker)
	mqOptions.SetClientID(m.opts.ClientID)
	log.Printf("MqttClient: Connecting to broker: %s, clientID: %s", m.opts.Broker, m.opts.ClientID)
	mClient := mqtt.NewClient(mqOptions)
	// We have to create the connection to the broker manually and verify that there is no error.
	if token := mClient.Connect(); token.Wait() && token.Error() != nil {
		log.Printf("MqttClient: Error connecting to broker: %s", m.opts.Broker)
		err := token.Error()
		log.Printf("MqttClient: Error: %v", err)
		return err
	}
	log.Printf("MqttClient: Connected to broker: %s", m.opts.Broker)
	m.Client = mClient
	return nil
}

// Publish publishes a message on a specific topic. An error is returned if there was problem. This function will publish with a QOS of 1.
func (m *MqttClient) Publish(topic string, payload interface{}) error {
	if m.Client == nil {
		return fmt.Errorf("client not connected")
	}
	if token := m.Client.Publish(topic, 1, false, payload); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

// Subscribe creates a subscription for the passed topic. The callBack function is used to process any messages that the client receives on that topic. The subscription created will have a QOS of 1.
func (m *MqttClient) Subscribe(topic string, f mqtt.MessageHandler) error {
	if token := m.Client.Subscribe(topic, 1, f); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	m.topics = append(m.topics, topic)
	return nil
}

func (m *MqttClient) Close() error {
	errorTokens := make([]error, 0)
	for _, t := range m.topics {
		token := m.Client.Unsubscribe(t)
		token.Wait()
		if token.Error() != nil {
			errorTokens = append(errorTokens, token.Error())
		}
	}
	m.Client.Disconnect(250)
	if len(errorTokens) > 0 {
		return fmt.Errorf("error unsubscribing from topics: %v", errorTokens)
	}
	return nil
}
