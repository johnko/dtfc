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
	"bytes"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/kennygrant/sanitize"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	//"strconv"
	"strings"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "All systems go.")
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "404 Not Found.", 404)
}

func putHandler(w http.ResponseWriter, r *http.Request) {
	if config.ALLOWPUT == "true" {
		vars := mux.Vars(r)
		filename := sanitize.Path(filepath.Base(vars["filename"]))
		contentLength := r.ContentLength
		var reader io.Reader
		var err error
		reader = r.Body
		if contentLength == -1 {
			// queue file to disk, because s3 needs content length
			var f io.Reader
			f = reader
			var b bytes.Buffer
			n, err := io.CopyN(&b, f, _24K+1)
			if err != nil && err != io.EOF {
				log.Printf("%s", err.Error())
				http.Error(w, "Internal server error.", 500)
				return
			}
			if n > _24K {
				file, err := ioutil.TempFile(config.Temp, "transfer-")
				if err != nil {
					log.Printf("%s", err.Error())
					http.Error(w, "Internal server error.", 500)
					return
				}
				defer file.Close()
				n, err = io.Copy(file, io.MultiReader(&b, f))
				if err != nil {
					os.Remove(file.Name())
					log.Printf("%s", err.Error())
					http.Error(w, "Internal server error.", 500)
					return
				}
				reader, err = os.Open(file.Name())
			} else {
				reader = bytes.NewReader(b.Bytes())
			}
			contentLength = n
		}
		token := Encode(10000000 + int64(rand.Intn(1000000000)))
		log.Printf("Uploading %s %s %d", token, filename, contentLength)
		if err = storage.Put(token, filename, reader, uint64(contentLength)); err != nil {
			log.Printf("%s", err.Error())
			http.Error(w, errors.New("Could not save file").Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		var hash string
		if hash, err = storage.HardLinkSha512(token, filename); err != nil {
			log.Printf("%s", err.Error())
			//fmt.Fprintf(w, "https://%s%s/%s/%s\n", ipAddrFromRemoteAddr(r.Host), config.NONROOTPATH, token, filename)
		} else if err == nil {
			log.Printf("Hashed %s %s as %s", token, filename, hash)
			fmt.Fprintf(w, "{\"sha512\":\"%s\",\"filename\":\"%s\"}",hash,filename)
		}
	} else {
		http.Error(w, "403 Forbidden. Uploading is disabled.", 403)
	}
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	if config.ALLOWGET == "true" {
		vars := mux.Vars(r)
		hash := vars["hash"]
		filename, reader, _, modTime, err := storage.Seeker(hash)
		if err != nil {
			if strings.Index(err.Error(), "no such file or directory") >= 0 {
				log.Printf("%s", err.Error())
				notFoundHandler(w, r)
				return
			} else {
				log.Printf("%s", err.Error())
				http.Error(w, "Could not retrieve file.", 500)
				return
			}
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		http.ServeContent(w, r, "", modTime, reader)
	} else {
		http.Error(w, "403 Forbidden. Downloading is disabled.", 403)
	}
}

func RedirectHandler(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health.html" {
		} else if ipAddrFromRemoteAddr(r.Host) == "127.0.0.1" {
		} else if r.Header.Get("X-Forwarded-Proto") != "https" && r.Method == "GET" {
			http.Redirect(w, r, "https://"+r.Host+r.RequestURI, 301)
			return
		}
		h.ServeHTTP(w, r)
	}
}
