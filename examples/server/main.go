package main

import (
	"github.com/evolidev/evoli/framework/logging"
	"github.com/evolidev/evoli/framework/use"
	"github.com/evolidev/reload"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	listenForSignal()

	args := os.Args[1:]

	if len(args) > 0 && args[0] == "watch" {
		log.Println("Args: ", args)

		conf := &reload.Configuration{
			AppRoot:            use.BasePath(),
			IncludedExtensions: []string{".go"},
			BuildPath:          "",
			BinaryName:         "main.go",
			Command:            "go run main.go",
			Debug:              false,
			ForcePolling:       false,
		}

		logging.Fatal(reload.RunBackground(conf))

		os.Exit(0)
	}

	log.Println("Starting server...")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World!"))
	})

	log.Println("Listening on port 8080")

	panic(http.ListenAndServe(":8080", nil))

}

func listenForSignal() {
	log.Printf("PID: %d\n", os.Getpid())
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	go func() {
		s := <-sigChannel
		log.Printf("received signal: %s\n", s)
		os.Exit(0)
	}()
}
