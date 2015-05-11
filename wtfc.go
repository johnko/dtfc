/*
The MIT License (MIT)

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
	"bytes"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	//"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func headHandler(w http.ResponseWriter, r *http.Request) {
	// force close because curl tries to keep open
	w.Header().Set("Connection", "close")
	if allowedHead() {
		vars := mux.Vars(r)
		hash := vars["hash"]
		_, _, _, err := storage.Head(hash)
		//filename, contentLength, _, err := storage.Head(hash)
		//contentType := mime.TypeByExtension(filepath.Ext(filename))
		if err != nil {
			// we only care about the status code
			w.WriteHeader(404)
			return
		} else {
			// you may want the json?
			//fmt.Fprintf(w, "{\"sha512\":\"%s\",\"filename\":\"%s\",\"length\":%d,\"content_type\":\"%s\",\"stub\":true}", hash, filename, contentLength, strings.Split(contentType, ";")[0])
		}
	} else {
		// we only care about the status code
		w.WriteHeader(403)
	}
}

func SplitHashToPairSlash(token string) string {
	// split hash to pairs because a folder of 400+ items is slow
	pairs := regexp.MustCompile("[0-9a-f]{2}").FindAll([]byte(token), -1)
	// join pairs with os.PathSeparator
	newtoken := bytes.Join(pairs, []byte(string(os.PathSeparator)))
	return string(newtoken[:])
}

func NameLengthTime(path string) (filename string, contentLength uint64, modTime time.Time, err error) {
	// content length
	var fi os.FileInfo
	if fi, err = os.Lstat(filepath.Join(path, "data")); err != nil {
		return
	}
	contentLength = uint64(fi.Size())
	modTime = fi.ModTime()
	// Use tail to get the last real filename
	var lastname []byte
	lastname, _ = exec.Command(cmdTAIL, "-n", "1", filepath.Join(path, "filename")).Output()
	// Assume the first output before space is the mimeType
	filename = strings.TrimSpace(fmt.Sprintf("%s", lastname))
	return
}

func Sha512(str string, word string) (hash string, err error) {
	// FreeBSD specific call /sbin/sha512 instead of using the import crypto/sha512
	// because the import has high memory usage (loads the data in RAM)
	// and Go lang uses garbage collection so the high RAM lingers
	// Assume the output is the hash, need to trim \n

	//hash, err := exec.Command(cmdSHA512, "-q", "-s", word).Output()
	//if err != nil {
	//      return
	//}

	// TODO: is shasum more universal on *nix systems?
	cmd := exec.Command(cmdSHASUM, "--algorithm", "512", str)
	if word != "" {
		cmd.Stdin = strings.NewReader(word)
	}
	tmpout, err := cmd.Output()
	// Assume the first output before space is the hash
	hash = strings.Split(strings.TrimSpace(fmt.Sprintf("%s", tmpout)), " ")[0]
	return
}

func Sha512Word(word string) (hash string, err error) {
	hash, err = Sha512("-", word)
	if err != nil {
		return
	}
	return
}

func (s *LocalStorage) saveFilename(hash string, filename string) {
	var err error
	newpath := filepath.Join(s.basedir, SplitHashToPairSlash(hash))
	var f1 io.WriteCloser
	f1, err = os.OpenFile(filepath.Join(newpath, "filename"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	defer f1.Close()
	io.Copy(f1, strings.NewReader(fmt.Sprintf("%s\n", filename)))
}

func (s *LocalStorage) HardLinkSha512Path(oldpath string, filename string) (hash string, contentLength uint64, err error) {
	var fi os.FileInfo
	if fi, err = os.Lstat(oldpath); err != nil {
		return
	}
	contentLength = uint64(fi.Size())
	hash, err = Sha512(oldpath, "")
	if err != nil {
		return
	}
	newpath := filepath.Join(s.basedir, SplitHashToPairSlash(hash))
	// mkdir -p
	if err = os.MkdirAll(newpath, 0700); err != nil && !os.IsExist(err) {
		return
	}
	// Link the oldpath to sha512/data
	if err = os.Link(oldpath, filepath.Join(newpath, "data")); err != nil {
		if strings.Index(err.Error(), "file exists") >= 0 {
			err = nil
		}
	}
	storage.saveFilename(newpath, filename)
	os.Remove(oldpath)
	return
}

func (s *LocalStorage) HardLinkSha512(token string, filename string) (hash string, contentLength uint64, err error) {
	oldpath := filepath.Join(config.Temp, token, filename)
	hash, contentLength, err = storage.HardLinkSha512Path(oldpath, filename)
	return
}

func (s *LocalStorage) DeleteFile(token string, filename string) error {
	oldpath := filepath.Join(config.Temp, token)
	os.Remove(filepath.Join(oldpath, filename))
	os.Remove(oldpath)
	return nil
}
