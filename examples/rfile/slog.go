package main

import (
	"fmt"
	"github.com/smartwalle/nlog/rfile"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	var file, _ = rfile.New("slogs/test.log", rfile.WithMaxAge(10))
	var logger = slog.New(slog.NewTextHandler(file, nil))
	slog.SetDefault(logger)

	defer file.Close()

	var closed = make(chan struct{}, 1)

	go func() {
		var i = 0
		defer func() {
			fmt.Println(i)
		}()

		for {
			time.Sleep(time.Millisecond * 1)
			select {
			case <-closed:
				return
			default:
				i++

				slog.Info("世间有些人从不相信一见钟情，相比偶然一次的巧合相遇，他们更愿意相信时间，因为时间总能验证巧合是否能成为相濡以沫或是心有灵犀。", "index", i)

			}
		}
	}()

	var c = make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
MainLoop:
	for {
		s := <-c
		switch s {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			close(closed)
			time.Sleep(time.Second * 1)
			break MainLoop
		}
	}
}
