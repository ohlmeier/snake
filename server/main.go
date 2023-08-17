package main

import (
	"log"
	"net/http"

	"github.com/ohlmeier/snake/game"
)

func main() {
	game.StartManager()
	log.Println("setup ok")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
