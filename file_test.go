package nlog_test

import (
	"log"
	"nlog"
	"testing"
)

func BenchmarkLogger_Write(b *testing.B) {
	var file, _ = nlog.NewFile("./logs")

	b.SetParallelism(100)
	log.SetOutput(file)
	log.SetFlags(log.Ldate | log.Llongfile)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			log.Println("xxxxx")
		}
	})
}
