package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ryanjyoder/stackfar"
)

func main() {
	var err error

	sitesStr, ok := os.LookupEnv("STACKFAR_SITES")
	if !ok {
		log.Fatal("STACKFAR_SITES must be set")
	}
	storageDir, ok := os.LookupEnv("STORAGE_DIR")
	if !ok {
		log.Fatal("STORAGE_DIR must be set")
	}
	fmt.Println("Saving sqlite to:", storageDir)

	templateDir, ok := os.LookupEnv("TEMPLATE_DIR")
	if !ok {
		log.Fatal("TEMPLATE_DIR must be set")
	}

	streamsDir, ok := os.LookupEnv("STREAMS_DIR")
	if !ok {
		log.Fatal("STREAMS_DIR must be set")
	}

	listenPort, ok := os.LookupEnv("LISTEN_PORT")
	if !ok {
		log.Fatal("LISTEN_PORT must be set")
	}

	sites := strings.Split(sitesStr, ",")
	stores, err := stackfar.NewStores(storageDir, sites)
	if err != nil {
		log.Fatal(err)
	}

	go stackfar.LoadStores(stores, streamsDir)

	webserver, err := stackfar.NewWebServer(stores, templateDir, ":"+listenPort)
	if err != nil {
		log.Fatal("failed to create handler:", err)
	}

	log.Fatal(webserver.ListenAndServer())
}
