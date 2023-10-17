package rfile

import (
	"bufio"
	"os"
)

func WithBuffer(bytes int) Option {
	return WithBuilder(func(name string, flag int, perm os.FileMode) (FileWriter, error) {
		return openBufferedFile(name, flag, perm, bytes)
	})
}

type bufferedFile struct {
	*bufio.Writer
	file *os.File
}

func openBufferedFile(name string, flag int, perm os.FileMode, size int) (*bufferedFile, error) {
	var file, err = os.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}

	var buffer = &bufferedFile{}
	buffer.Writer = bufio.NewWriterSize(file, size)
	buffer.file = file
	return buffer, nil
}

func (b *bufferedFile) Sync() error {
	if err := b.Writer.Flush(); err != nil {
		return err
	}
	return b.file.Sync()
}

func (b *bufferedFile) Close() error {
	if err := b.Sync(); err != nil {
		return err
	}
	return b.file.Close()
}
