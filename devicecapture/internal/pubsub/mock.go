package pubsub

import (
	"sync"

	"github.com/eclipse/paho.mqtt.golang"
)

// FakeClient implements mqtt.Client for minimal boilerplate testing
type FakeClient struct {
	mqtt.Client // Embed interface so unexported methods are implicitly handled

	messages map[string][][]byte
	handlers map[string]mqtt.MessageHandler
	mu       sync.Mutex
}

func NewMockClient() *FakeClient {
	c := mqtt.NewClient(mqtt.NewClientOptions())
	return &FakeClient{
		Client:   c,
		messages: map[string][][]byte{},
		mu:       sync.Mutex{},
	}
}

// FakeToken Mock Token to satisfy Paho's return requirements
type FakeToken struct {
	mqtt.Token
}

func (f *FakeToken) Wait() bool   { return true }
func (f *FakeToken) Error() error { return nil }

// Publish Override the exact method your application under test triggers
func (f *FakeClient) Publish(
	topic string,
	qos byte,
	retained bool,
	payload interface{},
) mqtt.Token {
	f.pushMessage(topic, payload)
	return &FakeToken{}
}

func (f *FakeClient) Subscribe(t string, h mqtt.MessageHandler) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.handlers[t] = h
	return nil
}

func (f *FakeClient) pushMessage(topic string, payload interface{}) {
	f.mu.Lock()
	defer f.mu.Unlock()
	value := payload.([]byte)
	f.messages[topic] = append(f.messages[topic], value)
}
