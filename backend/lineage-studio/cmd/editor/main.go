package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"connectrpc.com/connect"
	connectcors "connectrpc.com/cors"
	"connectrpc.com/validate"

	lsc "github.com/SunMaungOo/lineage-studio/internal/lineage-studio/connect"
	"github.com/rs/cors"
)

func printHelp() {

	fmt.Println("Arguments : <port> <repo location> <front end location>")
	fmt.Println("<port> = port to run service on")
	fmt.Println("<repo location> = folder where repo is saved")
	fmt.Println("<front end location> =  folder where front end is located (must contain index.html)")

}

func main() {

	args := os.Args[1:]

	if len(args) != 3 {

		printHelp()

		return
	}

	serverPort, err := strconv.Atoi(args[0])

	if err != nil {
		log.Fatal(err)
	}

	if !(serverPort >= 0 && serverPort <= 65535) {
		log.Fatal(fmt.Errorf("Invalid port number:%v", serverPort))
	}

	repoFolder := args[1]

	frontEndLocation := filepath.Join(args[2], "index.html")

	serverImplementation := LineageStudioServer{
		RepoLocation: repoFolder,
		RemoteName:   "origin",
		WriteLocks:   newRepoLocks(),
	}

	mux := http.NewServeMux()

	path, handler := lsc.NewLineageStudioServiceHandler(
		&serverImplementation,
		connect.WithInterceptors(validate.NewInterceptor()),
	)

	mux.Handle(path, handler)

	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		http.ServeFile(writer, request, frontEndLocation)
	})

	protocol := new(http.Protocols)
	protocol.SetHTTP1(true)
	protocol.SetUnencryptedHTTP2(true)

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   connectcors.AllowedMethods(),
		AllowedHeaders:   connectcors.AllowedHeaders(),
		ExposedHeaders:   connectcors.ExposedHeaders(),
		AllowCredentials: false,
		MaxAge:           7200,
	})

	serverAddr := fmt.Sprintf("localhost:%v", serverPort)

	server := http.Server{
		Addr:      serverAddr,
		Handler:   corsHandler.Handler(mux),
		Protocols: protocol,
	}

	server.ListenAndServe()

}
