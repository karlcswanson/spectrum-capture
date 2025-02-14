package main

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MQTTConnection represents the connection object for each MQTT server
type MQTTConnection struct {
	Client   mqtt.Client
	Messages chan MQTTMessage // Channel to send messages
}

type MQTTMessage struct {
	Topic   string
	Payload []byte
	Retain  bool
}

// ConnectToMQTT establishes a connection to an MQTT broker
func ConnectToMQTT(broker, clientID string) (*MQTTConnection, error) {
	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetKeepAlive(2 * time.Second).
		SetPingTimeout(1 * time.Second).
		SetAutoReconnect(true).
		SetOnConnectHandler(func(client mqtt.Client) {
			fmt.Printf("Connected to MQTT Broker: %s\n", broker)
		}).
		SetConnectionLostHandler(func(client mqtt.Client, err error) {
			fmt.Printf("Connection lost to MQTT Broker: %s. Error: %v\n", broker, err)
		})

	client := mqtt.NewClient(opts)
	token := client.Connect()

	// Wait for connection to complete
	if token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	return &MQTTConnection{
		Client:   client,
		Messages: make(chan MQTTMessage), // Initialize the message channel
	}, nil
}

// KeepSendingMessages loops to send messages from the channel to the broker
func (conn *MQTTConnection) KeepSendingMessages() {
	for msg := range conn.Messages {
		token := conn.Client.Publish(msg.Topic, 0, msg.Retain, msg.Payload)
		token.Wait()
		if token.Error() != nil {
			fmt.Printf("Error publishing message: %v\n", token.Error())
		}
	}
}

// StopMQTTConnection closes the MQTT connection gracefully
func (conn *MQTTConnection) StopMQTTConnection() {
	close(conn.Messages)        // Close the message channel
	conn.Client.Disconnect(200) // Disconnect the MQTT client
	fmt.Println("Disconnected from broker.")
}
