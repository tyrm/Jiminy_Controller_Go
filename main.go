package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

func handleReply(client MQTT.Client, msg MQTT.Message) {

	fmt.Printf("TOPIC: %s\n", msg.Topic())

	cmd , opts := parsePacket(string(msg.Payload()))
	fmt.Printf(" CMD:  %s\n", cmd)
	fmt.Printf(" OPTS: %s\n", opts)
}

func parsePacket(packet string) (cmd string, opts []string) {
	if (strings.HasPrefix(packet, "<") && strings.HasSuffix(packet, ">")) {
		packet = packet[1:len(packet)-1]        // Remove <> caps
		parts := strings.Split(packet, "|") // Split string at |
		cmd = parts[0]                          // First part is Command

		if len(parts) > 1 {
			opts = parts[1:]                    // if there are options return them
		}
	}
    return
}

func main() {
	//create a ClientOptions struct setting the broker address, clientid, turn
	//off trace output and set the default message handler
	opts := MQTT.NewClientOptions().AddBroker("tcp://10.1.68.60:1883")
	opts.SetClientID("go-simple")
	//opts.SetDefaultPublishHandler(f)

	//create and start a client using the above ClientOptions
	c := MQTT.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	//subscribe to the topic /go-mqtt/sample and request messages to be delivered
	//at a maximum qos of zero, wait for the receipt to confirm the subscription
	if token := c.Subscribe("/jiminy/reply", 0, handleReply); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	// Wait for SIGINT and SIGTERM (HIT CTRL-C)
	nch := make(chan os.Signal)
	signal.Notify(nch, syscall.SIGINT, syscall.SIGTERM)
	log.Println(<-nch)

	//unsubscribe from /go-mqtt/sample
	if token := c.Unsubscribe("/jiminy/c/all"); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

}