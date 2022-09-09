package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ostafen/demo/api"
	"github.com/ostafen/demo/store"

	"github.com/gin-gonic/gin"
)

const (
	addrDefault        = "localhost:8080"
	storagePathDefault = "."
)

func startServer(server *http.Server) {
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func shutdownServer(ctx context.Context, server *http.Server) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal(err)
	} else {
		log.Println("server successfully stopped")
	}
}

func listenSignals() {
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	<-stopCh
	close(stopCh)
}

func main() {
	storagePath := flag.String("storage", storagePathDefault, "root directory where persistent data will be stored")
	listenAddr := flag.String("host", addrDefault, "bind address of the server")

	flag.Parse()

	s, err := store.Open(*storagePath)
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	controller := api.NewEventController(s)

	gin.SetMode(gin.ReleaseMode)
	engine := gin.Default()
	controller.Register(engine)

	log.Printf("Starting server on %s with storage path \"%s\"\n", *listenAddr, *storagePath)

	server := &http.Server{Addr: *listenAddr, Handler: engine}
	go startServer(server)

	listenSignals()

	log.Println("shutting down server...")
	shutdownServer(context.Background(), server)
}
