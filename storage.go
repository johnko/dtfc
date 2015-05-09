/*
The MIT License (MIT)

transfer.sh was originally written by and
Copyright (c) 2014 DutchCoders [https://github.com/dutchcoders/]

Some modifications
Copyright (c) 2015 John Ko

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package main

import (
	"io"
	"os"
	"path/filepath"
	"time"
)

type Storage interface {
	Seeker(token string) (filename string, reader io.ReadSeeker, contentLength uint64, modTime time.Time, err error)
	Get(token string) (filename string, reader io.ReadCloser, contentLength uint64, modTime time.Time, err error)
	Head(token string) (filename string, contentLength uint64, modTime time.Time, err error)
	Put(token string, filename string, reader io.Reader, contentLength uint64) error

	HardLinkSha512Path(oldpath string, filename string) (hash string, contentLength uint64, err error)
	HardLinkSha512(token string, filename string) (hash string, contentLength uint64, err error)
	DeleteFile(token string, filename string) error
}

type LocalStorage struct {
	Storage
	basedir string
}

func NewLocalStorage(basedir string) (*LocalStorage, error) {
	return &LocalStorage{basedir: basedir}, nil
}

func (s *LocalStorage) Head(token string) (filename string, contentLength uint64, modTime time.Time, err error) {
	path := filepath.Join(s.basedir, SplitHashToPairSlash(token))
	filename, contentLength, modTime, err = NameLengthTime(path)
	return
}

func (s *LocalStorage) Get(token string) (filename string, reader io.ReadCloser, contentLength uint64, modTime time.Time, err error) {
	path := filepath.Join(s.basedir, SplitHashToPairSlash(token))
	if reader, err = os.Open(filepath.Join(path, "data")); err != nil {
		return
	}
	filename, contentLength, modTime, err = NameLengthTime(path)
	return
}

func (s *LocalStorage) Seeker(token string) (filename string, reader io.ReadSeeker, contentLength uint64, modTime time.Time, err error) {
	path := filepath.Join(s.basedir, SplitHashToPairSlash(token))
	if reader, err = os.Open(filepath.Join(path, "data")); err != nil {
		return
	}
	filename, contentLength, modTime, err = NameLengthTime(path)
	return
}

func (s *LocalStorage) Put(token string, filename string, reader io.Reader, contentLength uint64) error {
	var err error
	var f io.WriteCloser
	path := filepath.Join(config.Temp, token)
	if err = os.Mkdir(path, 0700); err != nil && !os.IsExist(err) {
		return err
	}
	if f, err = os.OpenFile(filepath.Join(path, filename), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600); err != nil {
		return err
	}
	defer f.Close()
	if _, err = io.Copy(f, reader); err != nil {
		return err
	}
	return nil
}
