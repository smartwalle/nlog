package main

import (
	"github.com/smartwalle/nlog"
	"github.com/smartwalle/nlog/rfile"
	"log/slog"
	"os"
)

func main() {
	var mHandler = nlog.NewMultiHandler()
	mHandler.Add(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	var file, _ = rfile.New("logs/test.log", rfile.WithBuffer(1*1024*1024))
	defer file.Close() // 因为启用了 Buffer，所以必须调用 Close 方法或者 Sync 方法以写入数据
	mHandler.Add(slog.NewTextHandler(file, &slog.HandlerOptions{Level: slog.LevelDebug}))

	var logger = slog.New(mHandler)
	logger.Info("Info", slog.Int("key1", 1), slog.Int("key2", 2), slog.Int("key3", 3), slog.Int("key4", 4), slog.Int("key5", 5))
	logger.Debug("Debug", slog.Int("key1", 1), slog.Int("key2", 2), slog.Int("key3", 3), slog.Int("key4", 4), slog.Int("key5", 5))
}
