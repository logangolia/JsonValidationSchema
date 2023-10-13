// This file is a skeleton for your project. You should replace this
// comment with true documentation.

package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/RICE-COMP318-FALL23/owldb-p1group37/handler"
	"github.com/RICE-COMP318-FALL23/owldb-p1group37/jsonschema"
)

// "github.com/santhosh-tekuri/jsonschema/v5/httploader"

func main() {

	var server http.Server
	var port int
	var schemaFilename string
	var err error

	// Your code goes here.

	// Set port as defined at -p, with default port 3318
	portPtr := flag.Int("p", 3318, "port on which the server will listen")
	//schemaPtr := flag.String("s", "", "schema file")
	flag.StringVar(&schemaFilename, "d", "", "JSON Data File")
	tokenPtr := flag.String("t", "", "token file")
	flag.Parse()

	port = *portPtr
	// Accept -s and -t flags but ignore them for now
	//schemaFilename = *schemaPtr
	schemaValidator, err := jsonschema.NewSchemaValidator(schemaFilename)

	if err != nil {
		slog.Error("Error creating schema validator", "error", err)
		return
	}

	//slog.Info("Schema validator created", "filename", schemaFilename)

	_ = tokenPtr

	// Set server address based on port
	server.Addr = ":" + fmt.Sprintf("%d", port)

	// Assign the handler to the server
	server.Handler = handler.New(schemaValidator)

	// The following code should go last and remain unchanged.
	// Note that you must actually initialize 'server' and 'port'
	// before this.

	// signal.Notify requires the channel to be buffered
	ctrlc := make(chan os.Signal, 1)
	signal.Notify(ctrlc, os.Interrupt, syscall.SIGTERM)
	go func() {
		// Wait for Ctrl-C signal
		<-ctrlc
		server.Close()
	}()

	// Start server
	slog.Info("Listening", "port", port)
	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		slog.Error("Server closed", "error", err)
	} else {
		slog.Info("Server closed", "error", err)
	}
}
