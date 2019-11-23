package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/jinzhu/configor"

	"github.com/xdbsoft/grest"
)

func loadConfig(path string) (grest.Config, error) {

	var cfg grest.Config
	if err := configor.Load(&cfg, path); err != nil {
		return cfg, err
	}

	return cfg, nil
}

var configPath = flag.String("config", "grest_server.toml", "path to the configuration file")
var listenAddr = flag.String("addr", ":9889", "address and port to listen on")

func main() {

	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		panic(err)
	}

	log.Println(cfg)

	grestHandler, err := grest.Server(cfg)
	if err != nil {
		panic(err)
	}

	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"authorization", "content-type"}),
	)(grestHandler)

	loggedRouter := handlers.LoggingHandler(os.Stdout, corsHandler)

	s := &http.Server{
		Addr:           *listenAddr,
		Handler:        loggedRouter,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())

}
