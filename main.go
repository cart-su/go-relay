package main

import (
	"cart-su/go-relay/api"
	"cart-su/go-relay/config"
	"log"
	"os"
)

func main() {
	config.LoadConfig()

	file, err := os.OpenFile("relay.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(file)
	log.Println("Starting server...")

	s := api.NewServer()
	api.SetRoutes(s.Engine)
	log.Printf("Server is running %s:%v\n", config.Config.ListenRange, config.Config.Port)

	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}
