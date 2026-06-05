package tasks

import (
	"os"
	"strconv"
	"time"
)

func Noop() error {
	seconds, err := strconv.Atoi(os.Getenv("NOOP_TIME"))
	if err != nil {
		return err
	}
	time.Sleep(time.Second * time.Duration(seconds))
	return nil
}
