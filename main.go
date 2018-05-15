package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
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

type SafeDeviceList struct {
	Devices map[string]JiminyDevice
	Lock    sync.RWMutex
}

//*********//
// Globals //
//*********//
var devices SafeDeviceList

func main() {
	devices = SafeDeviceList{Devices: make(map[string]JiminyDevice)}

	//create a ClientOptions struct setting the broker address, clientid, turn
	//off trace output and set the default message handler
	opts := MQTT.NewClientOptions().AddBroker("tcp://10.1.68.60:1883")
	myMac := getMacAddr()
	opts.SetClientID(myMac)

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

	// Start Emitting Pings
	go pongEmitter(c)

	// Wait for SIGINT and SIGTERM (HIT CTRL-C)
	nch := make(chan os.Signal)
	signal.Notify(nch, syscall.SIGINT, syscall.SIGTERM)
	log.Println(<-nch)

	//unsubscribe from mqtt
	if token := c.Unsubscribe("/jiminy/reply"); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}
}

//***********//
// Functions //
//***********//
func getMacAddr() (addr string) {
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, i := range interfaces {
			if i.Flags&net.FlagUp != 0 && bytes.Compare(i.HardwareAddr, nil) != 0 {
				// Don't use random as we have a real address
				addr = i.HardwareAddr.String()
				break
			}
		}
	}
	return
}

func (dl *SafeDeviceList) getDevice(id string) JiminyDevice {
	dl.Lock.RLock()
	defer dl.Lock.RUnlock()

	return dl.Devices[id]
}

func (dl *SafeDeviceList) getDevices() map[string]JiminyDevice {
	dl.Lock.RLock()
	defer dl.Lock.RUnlock()

	return dl.Devices
}

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
		dl := devices.getDevices()
		b, _ := json.Marshal(dl)

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

func pongEmitter(c MQTT.Client) {
	for {
		token := c.Publish("/jiminy/c/all", 0, false, "<PING>")
		token.Wait()

		time.Sleep(10 * time.Second)
	}
}

func pongResponse(id string, count int) {
	devices.setDevice(id, JiminyDevice{ID: id, Count: count, LastSeen: time.Now()})
}

func (dl *SafeDeviceList) setDevice(id string, d JiminyDevice) {
	dl.Lock.Lock()
	defer dl.Lock.Unlock()

	dl.Devices[id] = d
}