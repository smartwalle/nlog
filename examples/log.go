package main

import (
	"fmt"
	"github.com/smartwalle/rollingfile"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	var file, _ = rollingfile.New("logs/test.log", rollingfile.WithMaxAge(10))
	log.SetOutput(file)
	log.SetFlags(log.Ldate | log.Ltime | log.Llongfile)

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
				log.Println(i, "世间有些人从不相信一见钟情，相比偶然一次的巧合相遇，他们更愿意相信时间，因为时间总能验证巧合是否能成为相濡以沫或是心有灵犀。")

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
