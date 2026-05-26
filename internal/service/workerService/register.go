package workerService

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
)

func RegisterSelf() error {
	registration, err := http.Post(
		fmt.Sprintf("http://%v:%v/worker", os.Getenv("QUEUE_HOST"), os.Getenv("QUEUE_PORT")),
		"application/json",
		bytes.NewReader([]byte{}),
	)
	if err != nil {
		log.Printf("Registeration erred: %v\n", err.Error())
		return err
	}
	if registration.StatusCode != 200 {
		log.Printf("Registeration failed: %v\n", registration.StatusCode)
		return err
	}
	log.Printf("Added self, got %#v", registration)
	return nil
}
