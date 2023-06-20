package nlog_test

import (
	"fmt"
	"github.com/smartwalle/nlog"
	"log"
	"math/rand"
	"testing"
)

func BenchmarkLogger_Write(b *testing.B) {
	var n = rand.Int()
	var file, err = nlog.NewFile(fmt.Sprintf("logs/%d.log", n), nlog.WithBuffer(10*1024*1024))
	if err != nil {
		b.Fatal(err)
	}

	b.SetParallelism(100)
	log.SetOutput(file)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	for i := 0; i < b.N; i++ {
		log.Println(n, i, "xxxxx")
	}

	file.Close()
	b.Log(n, b.N)
}
