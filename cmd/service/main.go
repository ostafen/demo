package main

import (
	"demo/api"
	"demo/store"
	"flag"
	"log"

	"github.com/gin-gonic/gin"
)

const (
	addrDefault        = "localhost:8080"
	storagePathDefault = "."
)

func main() {
	storagePath := flag.String("storage", storagePathDefault, "root directory where persistent data will be stored")
	listenAddr := flag.String("host", addrDefault, "bind address of the server")

	flag.Parse()

	s, err := store.Open(*storagePath)
	if err != nil {
		log.Fatal(err)
	}

	controller := api.NewEventController(s)

	gin.SetMode(gin.ReleaseMode)
	engine := gin.Default()
	controller.Register(engine)

	log.Printf("Starting server on %s with storage path \"%s\"\n", *listenAddr, *storagePath)

	engine.Run(*listenAddr)
}
