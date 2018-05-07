package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)


//*********//
//  Types  //
//*********//
type JiminyDevice struct {
	ID        string
	Count     int
	LastSeen  time.Time
}

//*********//
// Globals //
//*********//
var devices map[string]JiminyDevice

func main() {
	devices = make(map[string]JiminyDevice)

	//create a ClientOptions struct setting the broker address, clientid, turn
	//off trace output and set the default message handler
	opts := MQTT.NewClientOptions().AddBroker("tcp://10.1.68.60:1883")
	opts.SetClientID("go-simple")

	//create and start a client using the above ClientOptions
	c := MQTT.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	if token := c.Subscribe("/jiminy/reply", 0, handleReply); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	http.HandleFunc("/jiminy/devices", httpDevices)
	go http.ListenAndServe(":8080", nil)

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

//***********//
// Functions //
//***********//

func handleReply(client MQTT.Client, msg MQTT.Message) {

	fmt.Printf("TOPIC: %s\n", msg.Topic())

	cmd, opts := parsePacket(string(msg.Payload()))
	fmt.Printf(" CMD:  %s\n", cmd)
	fmt.Printf(" OPTS: %s\n", opts)

	switch cmd {
	case "PONG":
		count, err := strconv.Atoi(opts[1])
		if err != nil {
			return
		}
		pongResponse(opts[0], count)
	}
}

func httpDevices(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-Type", "application/json")

	if request.Method == "GET" {
		b, _ := json.Marshal(devices)

		fmt.Printf(" OPTS: %s\n", b)
		fmt.Fprintf(response, "%s", b)
		return

	} else {
		response.WriteHeader(405)

		fmt.Fprint(response, request.Method)
		return
	}
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

func pongResponse(id string, count int) {
	devices[id] = JiminyDevice{ID: id, Count: count, LastSeen: time.Now()}
}
