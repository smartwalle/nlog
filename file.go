package nlog

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"
)

type Option func(opts *File)

func WithMaxSize(mb int64) Option {
	return func(opts *File) {
		if mb <= 0 {
			return
		}
		opts.maxSize = mb * 1024 * 1024
	}
}

func WithMaxAge(seconds int64) Option {
	return func(opts *File) {
		if seconds <= 0 {
			return
		}
		opts.maxAge = seconds
	}
}

type File struct {
	path    string
	name    string
	maxSize int64
	maxAge  int64

	mu     sync.Mutex
	file   *os.File
	size   int64
	closed bool
	clean  chan struct{}
}

const (
	kFilename = "temp_log.log"
	kFileExt  = ".log"
)

func NewFile(path string, opts ...Option) (*File, error) {
	var file = &File{}
	file.path = path
	file.name = filepath.Join(path, kFilename)
	file.maxSize = 10 * 1024 * 1024
	file.maxAge = 0
	file.clean = make(chan struct{}, 1)
	for _, opt := range opts {
		if opt != nil {
			opt(file)
		}
	}
	if err := os.MkdirAll(file.path, 0755); err != nil {
		return nil, err
	}

	go file.runClean()
	return file, nil
}

func (this *File) Write(b []byte) (n int, err error) {
	this.mu.Lock()
	defer this.mu.Unlock()
	if this.closed {
		return 0, fs.ErrClosed
	}

	var wLen = int64(len(b))
	if this.file == nil {
		if err = this.openOrCreate(wLen); err != nil {
			return 0, err
		}
	}

	if this.size+wLen > this.maxSize {
		if err = this.rotate(); err != nil {
			return 0, err
		}
	}

	n, err = this.file.Write(b)
	this.size += int64(n)
	return n, err
}

func (this *File) openOrCreate(size int64) error {
	this.needClean()

	// 获取log文件信息
	var info, err = os.Stat(this.name)
	if os.IsNotExist(err) {
		// 如果log文件不存在，直接创建新的log文件
		return this.create()
	}
	if err != nil {
		return err
	}

	// 文件存在，但是其文件大小已超出设定的阈值
	if info.Size()+size >= this.maxSize {
		return this.rotate()
	}

	// 打开现有的文件
	file, err := os.OpenFile(this.name, os.O_APPEND|os.O_WRONLY, 0777)
	if err != nil {
		// 如果打开文件出错，则创建新的文件
		return this.create()
	}

	this.file = file
	this.size = info.Size()
	return nil
}

func (this *File) create() error {
	var file, err = os.OpenFile(this.name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0777)
	if err != nil {
		return err
	}
	this.file = file
	this.size = 0
	return nil
}

func (this *File) rename() error {
	_, err := os.Stat(this.name)
	if err == nil {
		var name = path.Join(this.path, fmt.Sprintf("log_%s.log", time.Now().Format("2006_01_02_15_04_05.000000")))
		if err = os.Rename(this.name, name); err != nil {
			return err
		}
	}
	return err
}

func (this *File) rotate() error {
	if err := this.close(); err != nil {
		return err
	}

	if err := this.rename(); err != nil {
		return err
	}

	if err := this.create(); err != nil {
		return err
	}

	this.needClean()
	return nil
}

func (this *File) Close() error {
	this.mu.Lock()
	defer this.mu.Unlock()
	if this.closed {
		return nil
	}
	this.closed = true
	close(this.clean)
	return this.close()
}

func (this *File) close() error {
	if this.file == nil {
		return nil
	}
	var err = this.file.Close()
	this.file = nil
	return err
}

func (this *File) needClean() {
	select {
	case this.clean <- struct{}{}:
	default:
	}
}

func (this *File) runClean() {
	if this.maxAge <= 0 {
		return
	}

	for {
		select {
		case _, ok := <-this.clean:
			if !ok {
				return
			}
			var files, _ = os.ReadDir(this.path)
			for _, file := range files {
				info, _ := file.Info()
				if info != nil && !info.IsDir() && info.ModTime().Unix() < (time.Now().Unix()-this.maxAge) {
					if filepath.Ext(info.Name()) == kFileExt && info.Name() != kFilename {
						os.Remove(filepath.Join(this.path, info.Name()))
					}
				}
			}
		}
	}

}
