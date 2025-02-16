package main

import (
	"encoding/json"
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

type StatusMessage struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Command     string `json:"command"`
	Status      string `json:"status"`
}

// ConnectToMQTT establishes a connection to an MQTT broker
func ConnectToMQTT(broker, clientID string) (*MQTTConnection, error) {

	lastWill := StatusMessage{
		Id:          clientID,
		Name:        config.Name,
		Description: config.Description,
		Command:     config.RTLCommand,
		Status:      "offline",
	}

	lastWillPayload, err := json.Marshal(lastWill)
	if err != nil {
		fmt.Printf("Error marshalling last will message: %v\n", err)
	}

	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetKeepAlive(2*time.Second).
		SetPingTimeout(1*time.Second).
		SetAutoReconnect(true).
		SetWill("scanner/"+clientID+"/status", string(lastWillPayload), 0, true).
		SetOnConnectHandler(func(client mqtt.Client) {
			fmt.Printf("Connected to MQTT Broker: %s\n", broker)
			statusMsg := StatusMessage{
				Id:          clientID,
				Name:        config.Name,
				Description: config.Description,
				Command:     config.RTLCommand,
				Status:      "online",
			}
			statusMsgPayload, err := json.Marshal(statusMsg)
			if err != nil {
				fmt.Printf("Error marshalling status message: %v\n", err)
			}
			client.Publish("scanner/"+clientID+"/status", 0, true, string(statusMsgPayload))
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
