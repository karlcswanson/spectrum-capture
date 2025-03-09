package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	_ "github.com/joho/godotenv/autoload"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type MQTTConfig struct {
	Broker   string `json:"broker"`
	ClientID string `json:"client_id"`
}

type Config struct {
	RTLCommand        string       `json:"command"`
	Name              string       `json:"name"`
	Description       string       `json:"description"`
	MQTTServerConfigs []MQTTConfig `json:"servers"`
}

type RTLPowerLine struct {
	Id        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	HzLo      float64   `json:"hz_lo"`
	HzHigh    float64   `json:"hz_hi"`
	Step      float64   `json:"step"`
	Samples   float64   `json:"samples"`
	Power     []float64 `json:"power"`
}

type ScanTable struct {
	HzLo          float64
	HzHigh        float64
	Step          float64
	Power         []float64
	PowerTable    [][]float64
	FirstScanDone bool
	Counter       int
}

var config Config
var scanTable ScanTable

func main() {
	loadConfig()
	mqttChannel, err := ConnectToMQTT(config.MQTTServerConfigs[0].Broker, config.MQTTServerConfigs[0].ClientID)
	if err != nil {
		fmt.Println("Error connecting to mqtt servers")
	}
	defer mqttChannel.StopMQTTConnection()
	go mqttChannel.KeepSendingMessages()

	command := config.RTLCommand
	commands := strings.Split(command, " ")
	// Create a channel for passing parsed output
	dataChannel := make(chan RTLPowerLine)

	// Start a goroutine to handle the output as it comes in
	go processOutput(dataChannel, mqttChannel)

	// Create the command object
	cmd := exec.Command(commands[0], commands[1:]...)

	// Get a pipe to capture the command's standard output
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Error creating stdout pipe: %v\n", err)
		return
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		fmt.Printf("Error creating stdout pipe: %v\n", err)
		return
	}

	// Start the command execution
	if err := cmd.Start(); err != nil {
		fmt.Printf("Error starting command: %v\n", err)
		return
	}

	// Use a scanner to capture the real-time output line by line
	scanner := bufio.NewScanner(stdoutPipe)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			parsedData := parseOutput(line)
			dataChannel <- *parsedData // Send parsed data to the channel
		}
		if err := scanner.Err(); err != nil {
			fmt.Printf("Error while reading command output: %v\n", err)
		}
		// Close the channel when done
		close(dataChannel)
	}()

	errScanner := bufio.NewScanner(stderrPipe)
	go func() {
		for errScanner.Scan() {
			log.Printf("rtl %s\n", errScanner.Text())

		}
		if err := scanner.Err(); err != nil {
			fmt.Printf("Error while reading command output: %v\n", err)
		}
	}()

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		fmt.Printf("Command execution ended with error: %v\n", err)
	}
}

// Function to parse output (modify as per needs)
func parseOutput(outputLine string) *RTLPowerLine {
	fields := strings.Split(outputLine, ", ")
	if len(fields) < 5 {
		fmt.Printf("Invalid output line: %s\n", outputLine)
		return nil
	}
	//fmt.Println(outputLine)

	timestamp, err := time.Parse("2006-01-02 15:04:05", strings.Join([]string{fields[0], fields[1]}, " "))
	if err != nil {
		fmt.Printf("Error parsing timestamp: %v\n", err)
	}

	hzLo, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		fmt.Printf("Error parsing HzLo: %v\n", err)
	}

	hzHi, err := strconv.ParseFloat(fields[3], 64)
	if err != nil {
		fmt.Printf("Error parsing HzHi: %v\n", err)
	}

	step, err := strconv.ParseFloat(fields[4], 64)
	if err != nil {
		fmt.Printf("Error parsing step: %v\n", err)
	}

	samples, err := strconv.ParseFloat(fields[5], 64)
	if err != nil {
		fmt.Printf("Error parsing samples: %v\n", 64)
	}

	power := make([]float64, 0, len(fields)-6) // Pre-allocate the slice capacity
	for _, value := range fields[6:] {
		p, err := strconv.ParseFloat(value, 64)
		if err != nil {
			fmt.Printf("Error parsing power value: %v\n", err)
			continue
		}
		power = append(power, p)
	}

	return &RTLPowerLine{
		Id:        config.MQTTServerConfigs[0].ClientID,
		Timestamp: timestamp,
		HzLo:      hzLo,
		HzHigh:    hzHi,
		Step:      step,
		Samples:   samples,
		Power:     power,
	}
}

// Process the output received via the channel
func processOutput(dataChannel <-chan RTLPowerLine, mqttConnection *MQTTConnection) {
	for data := range dataChannel {
		// Print or process the received data
		fmt.Printf("start: %v stop: %v step: %v counts: %d\n", data.HzLo/1000000, data.HzHigh/1000000, data.Step, len(data.Power))
		// Add additional logic here as needed
		scanManager(&data, mqttConnection)
		msg, err := json.Marshal(data)
		if err != nil {
			fmt.Printf("error \n")
		}

		topic := fmt.Sprintf("scanner/%s/stream", config.MQTTServerConfigs[0].ClientID)
		mqttConnection.Messages <- MQTTMessage{
			Topic:   topic,
			Payload: msg,
			Retain:  false,
		}
	}
}

func loadConfig() {
	cfg := os.Getenv("SPECTRUM_CFG")
	if cfg == "" {
		fmt.Printf("SPECTRUM_CFG is empty\n")
	}

	err := json.Unmarshal([]byte(cfg), &config)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
	}
	fmt.Println("Configuration loaded successfully:", config)

	//length := int((config.HzHi - config.HzLo) / config.Step)
	//if length <= 0 {
	//	log.Fatalf("Invalid configuration: HzHi must be greater than HzLo, and Step must be positive")
	//}

	scanTable.Power = make([]float64, 0)
	fmt.Println(len(scanTable.Power))

}

func scanManager(line *RTLPowerLine, mqttConnection *MQTTConnection) {
	if scanTable.FirstScanDone == false {
		if scanTable.HzLo == 0 {
			scanTable.HzLo = line.HzLo
			scanTable.Step = line.Step
		}

		if line.HzHigh > scanTable.HzHigh {
			scanTable.HzHigh = line.HzHigh
		}
		if line.HzHigh < scanTable.HzHigh {
			scanTable.FirstScanDone = true
			fmt.Printf("FIRST SCAN DONE start: %v %v\n", scanTable.HzLo/1000000, scanTable.HzHigh/1000000)
		}
	}

	if scanTable.FirstScanDone && line.HzLo <= scanTable.HzLo {
		scanTable.Counter += 1
		fmt.Printf("DONE %d %d\n", scanTable.Counter, len(scanTable.Power))

		pl := RTLPowerLine{
			Id:        config.MQTTServerConfigs[0].ClientID,
			Timestamp: line.Timestamp,
			HzLo:      scanTable.HzLo,
			HzHigh:    scanTable.HzHigh,
			Step:      scanTable.Step,
			Samples:   0,
			Power:     scanTable.Power,
		}

		msg, err := json.Marshal(pl)
		if err != nil {
			fmt.Printf("error \n")
		}

		topic := fmt.Sprintf("scanner/%s/scan", config.MQTTServerConfigs[0].ClientID)
		mqttConnection.Messages <- MQTTMessage{
			Topic:   topic,
			Payload: msg,
			Retain:  true,
		}

		scanTable.Power = make([]float64, 0)
	}

	scanTable.Power = append(scanTable.Power, line.Power...)
}
