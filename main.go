package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("ca66c829-a2e1-4ecb-abf9-5f6e03dabd97")
		fmt.Fprintf(w, "Hello")
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
