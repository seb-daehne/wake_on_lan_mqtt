package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	wol "github.com/linde12/gowol"
)

type WakeMessage struct {
	Mac  string `json:"mac`
	Name string `json:"name`
}

var connectionHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("OnConnected")
}

var connectionLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connection Lost: %v\n", err)
}

var messageHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())

	var message WakeMessage
	json.Unmarshal(msg.Payload(), &message)

	fmt.Printf(" + sending wake on lan to (%s) mac: %s\n", message.Name, message.Mac)
	if packet, err := wol.NewMagicPacket("03:AA:FF:67:64:05"); err == nil {
		packet.Send("255.255.255.255")
		packet.SendPort("255.255.255.255", "7")
	}

}

func subscribe(client mqtt.Client, topic string) {
	token := client.Subscribe(topic, 1, nil)
	token.Wait()
	fmt.Printf(" + subscribed to topic: %s\n", topic)

}

func waitForSignal() {
	signals := make(chan os.Signal, 1)
	ready := make(chan bool, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		signal := <-signals
		fmt.Printf("\n ! signal received: %s", signal)
		ready <- true
	}()

	<-ready
}

func main() {
	fmt.Println("+ start")

	value, exists := os.LookupEnv("MQTT_SERVER")
	if exists == false {
		fmt.Println("- missing MQTT_SERVER environment variable!")
		os.Exit(1)
	}
	mqttServer := value

	connectOptions := mqtt.NewClientOptions()
	connectOptions.SetClientID("wake_on_lan_mqtt")
	connectOptions.AddBroker(mqttServer)
	connectOptions.SetDefaultPublishHandler(messageHandler)
	connectOptions.OnConnect = connectionHandler
	connectOptions.OnConnectionLost = connectionLostHandler
	client := mqtt.NewClient(connectOptions)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	subscribe(client, "/wake_on_lan")

	waitForSignal()

	client.Disconnect(250)
}
