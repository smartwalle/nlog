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

func (f *File) Write(b []byte) (n int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return 0, fs.ErrClosed
	}

	var wLen = int64(len(b))
	if f.file == nil {
		if err = f.openOrCreate(wLen); err != nil {
			return 0, err
		}
	}

	if f.size+wLen > f.maxSize {
		if err = f.rotate(); err != nil {
			return 0, err
		}
	}

	n, err = f.file.Write(b)
	f.size += int64(n)
	return n, err
}

func (f *File) openOrCreate(size int64) error {
	f.needClear()

	// 获取文件信息
	var info, err = os.Stat(f.filename)
	if os.IsNotExist(err) {
		// 如果文件不存在，直接创建新的文件
		return f.create()
	}
	if err != nil {
		return err
	}

	// 文件存在，但是其文件大小已超出设定的阈值
	if info.Size()+size >= f.maxSize {
		return f.rotate()
	}

	// 打开现有的文件
	file, err := f.builder(f.filename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		// 如果打开文件出错，则创建新的文件
		return f.create()
	}

	f.file = file
	f.size = info.Size()
	return nil
}

func (f *File) create() error {
	var file, err = f.builder(f.filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	f.file = file
	f.size = 0
	return nil
}

func (f *File) rename() error {
	_, err := os.Stat(f.filename)
	if err == nil {
		var name = fmt.Sprintf(f.backup, time.Now().Format("2006_01_02_15_04_05.000000"))
		if err = os.Rename(f.filename, name); err != nil {
			return err
		}
	}
	return err
}

func (f *File) rotate() error {
	if err := f.close(); err != nil {
		return err
	}

	if err := f.rename(); err != nil {
		return err
	}

	if err := f.create(); err != nil {
		return err
	}

	f.needClear()
	return nil
}

func (f *File) Sync() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return fs.ErrClosed
	}
	return f.file.Sync()
}

func (f *File) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return nil
	}
	f.closed = true
	close(f.clear)
	return f.close()
}

func (f *File) close() error {
	if f.file == nil {
		return nil
	}
	var err = f.file.Close()
	f.file = nil
	return err
}

func (f *File) needClear() {
	select {
	case f.clear <- struct{}{}:
	default:
	}
}

func (f *File) runClear() {
	if f.maxAge <= 0 {
		return
	}

	for {
		select {
		case _, ok := <-f.clear:
			if !ok {
				return
			}
			var files, _ = os.ReadDir(f.filepath)
			for _, file := range files {
				info, _ := file.Info()
				if info != nil && !info.IsDir() && info.ModTime().Unix() < (time.Now().Unix()-f.maxAge) {
					if info.Name() != f.basename && filepath.Ext(info.Name()) == f.extension {
						os.Remove(filepath.Join(f.filepath, info.Name()))
					}
				}
			}
		}
	}
}
