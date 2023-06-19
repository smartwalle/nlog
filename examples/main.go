package main

import (
	"github.com/smartwalle/nlog"
	"log"

	"time"
)

func main() {
	var file, _ = nlog.NewFile("logs/test.log", nlog.WithMaxAge(10), nlog.WithMaxSize(1))
	log.SetOutput(file)
	log.SetFlags(log.Ldate | log.Ltime | log.Llongfile)

	for {
		time.Sleep(time.Millisecond * 1)
		log.Println("世间有些人从不相信一见钟情，相比偶然一次的巧合相遇，他们更愿意相信时间，因为时间总能验证巧合是否能成为相濡以沫或是心有灵犀。")
	}
}
