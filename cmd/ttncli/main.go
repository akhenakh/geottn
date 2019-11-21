package main

import (
	"context"
	"encoding/hex"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	ttnsdk "github.com/TheThingsNetwork/go-app-sdk"
)

const appName = "ttncli"

var (
	appID        = flag.String("appID", "akhtestapp", "The things network application ID")
	appAccessKey = flag.String("appAccessKey", "", "The things network access key")
)

func main() {
	flag.Parse()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	config := ttnsdk.NewCommunityConfig(appName)
	config.ClientVersion = "1.0"

	// Create a new SDK client for the application
	client := config.NewClient(*appID, *appAccessKey)

	// Make sure the client is closed before the function returns
	// In your application, you should call this before the application shuts down
	defer client.Close()

	// Start Publish/Subscribe client (MQTT)
	pubsub, err := client.PubSub()
	if err != nil {
		log.Fatal("can't get pub/sub", err)
	}

	// Get a publish/subscribe client for all devices
	allDevicesPubSub := pubsub.AllDevices()

	// Make sure the pubsub client is closed before the function returns
	// In your application, you will probably call this before the application shuts down
	// This also stops existing subscriptions, in case you forgot to unsubscribe
	defer allDevicesPubSub.Close()

	// Subscribe to events

	msgs, err := allDevicesPubSub.SubscribeUplink()
	if err != nil {
		log.Fatal("can't subscribe to events", err)
	}

	// catch termination
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupt)

	go func() {
		for {
			select {
			case <-ctx.Done():
				// Unsubscribe from events
				log.Println("unsubscribe from all devices")
				if err = allDevicesPubSub.UnsubscribeUplink(); err != nil {
					log.Fatal("can't unsubscribe from uplink msg", err)
				}
				return
			case msg := <-msgs:
				log.Println("msg", msg)

				if msg != nil && msg.PayloadRaw != nil {
					hexPayload := hex.EncodeToString(msg.PayloadRaw)
					log.Println("received msg", "data", hexPayload)
				}
			}
		}
	}()

	select {
	case <-interrupt:
		break
	case <-ctx.Done():
		break
	}

}
