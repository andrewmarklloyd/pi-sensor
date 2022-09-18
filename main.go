package main

import (
	"fmt"
	"time"
)

func main() {
	go forever()
	select {} // block forever
}

func forever() {
	for {
		fmt.Println(time.Now().String())
		time.Sleep(5 * time.Minute)
	}
}
