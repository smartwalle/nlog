package rfile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type FileWriter interface {
	Write(b []byte) (n int, err error)

	Sync() error

	Close() error
}

type FileBuilder func(name string, flag int, perm os.FileMode) (FileWriter, error)

type Option func(opts *File)

func WithMaxSize(bytes int64) Option {
	return func(opts *File) {
		if bytes <= 0 {
			return
		}
		opts.maxSize = bytes
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

func WithBuilder(builder FileBuilder) Option {
	return func(opts *File) {
		opts.builder = builder
	}
}

type File struct {
	filename  string // logs/test.txt
	filepath  string // logs
	basename  string // test.txt
	extension string // .txt
	backup    string // logs/test-%s.txt

	maxSize int64
	maxAge  int64

	mu      sync.Mutex
	builder FileBuilder
	file    FileWriter
	size    int64
	closed  bool
	clear   chan struct{}
}

func New(filename string, opts ...Option) (*File, error) {
	if filename == "" {
		return nil, errors.New("filename cannot be empty")
	}

	info, err := os.Stat(filename)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if info != nil && info.IsDir() {
		return nil, fmt.Errorf("a folder with the name %s already exists", filename)
	}

	var file = &File{}
	file.filename = filename
	file.filepath = filepath.Dir(filename)
	file.basename = filepath.Base(filename)
	file.extension = filepath.Ext(filename)
	file.backup = filepath.Join(file.filepath, strings.Split(file.basename, ".")[0]+"-%s"+file.extension)

	file.maxSize = 10 * 1024 * 1024
	file.maxAge = 0

	file.builder = func(name string, flag int, perm os.FileMode) (FileWriter, error) {
		return os.OpenFile(name, flag, perm)
	}
	file.clear = make(chan struct{}, 1)

	for _, opt := range opts {
		if opt != nil {
			opt(file)
		}
	}

	if err = os.MkdirAll(file.filepath, 0755); err != nil {
		return nil, err
	}

	go file.runClear()
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
	this.needClear()

	// 获取文件信息
	var info, err = os.Stat(this.filename)
	if os.IsNotExist(err) {
		// 如果文件不存在，直接创建新的文件
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
	file, err := this.builder(this.filename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		// 如果打开文件出错，则创建新的文件
		return this.create()
	}

	this.file = file
	this.size = info.Size()
	return nil
}

func (this *File) create() error {
	var file, err = this.builder(this.filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	this.file = file
	this.size = 0
	return nil
}

func (this *File) rename() error {
	_, err := os.Stat(this.filename)
	if err == nil {
		var name = fmt.Sprintf(this.backup, time.Now().Format("2006_01_02_15_04_05.000000"))
		if err = os.Rename(this.filename, name); err != nil {
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

	this.needClear()
	return nil
}

func (this *File) Sync() error {
	this.mu.Lock()
	defer this.mu.Unlock()
	if this.closed {
		return fs.ErrClosed
	}
	return this.file.Sync()
}

func (this *File) Close() error {
	this.mu.Lock()
	defer this.mu.Unlock()
	if this.closed {
		return nil
	}
	this.closed = true
	close(this.clear)
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

func (this *File) needClear() {
	select {
	case this.clear <- struct{}{}:
	default:
	}
}

func (this *File) runClear() {
	if this.maxAge <= 0 {
		return
	}

	for {
		select {
		case _, ok := <-this.clear:
			if !ok {
				return
			}
			var files, _ = os.ReadDir(this.filepath)
			for _, file := range files {
				info, _ := file.Info()
				if info != nil && !info.IsDir() && info.ModTime().Unix() < (time.Now().Unix()-this.maxAge) {
					if info.Name() != this.basename && filepath.Ext(info.Name()) == this.extension {
						os.Remove(filepath.Join(this.filepath, info.Name()))
					}
				}
			}
		}
	}
}
