package main

import (
	"encoding/json"
	"fmt"
	"os"

	webpush "github.com/SherClockHolmes/webpush-go"
)

const (
	subscription    = ``
	vapidPublicKey  = ""
	vapidPrivateKey = ""
)

func main() {
	a := os.Args

	// Decode subscription
	s := &webpush.Subscription{}
	json.Unmarshal([]byte(subscription), s)

	// Send Notification
	resp, err := webpush.SendNotification([]byte(a[1]), s, &webpush.Options{
		Subscriber:      "", // Do not include "mailto:"
		VAPIDPublicKey:  vapidPublicKey,
		VAPIDPrivateKey: vapidPrivateKey,
		TTL:             30,
	})
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	fmt.Println(resp)
}

func Generate() {
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		panic(err)
	}
	fmt.Println(privateKey, publicKey)
}
