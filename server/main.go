package main

import (
	"log"
	"net/http"

	"github.com/ohlmeier/snake/connection"
)

func main() {
	connection.Setup()
	log.Fatal(http.ListenAndServe(":5000", nil))
}
