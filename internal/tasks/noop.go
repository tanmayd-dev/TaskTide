package tasks

import "time"

func Noop() error {
	time.Sleep(time.Second * 10)
	return nil
}