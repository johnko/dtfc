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
	"fmt"
	"github.com/gorilla/mux"
	"github.com/kennygrant/sanitize"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "All systems go.")
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "404 Not Found.", 404)
}

func putHandler(w http.ResponseWriter, r *http.Request) {
	if allowedPut() {
		var contentLength uint64
		var err error
		vars := mux.Vars(r)
		filename := sanitize.Path(filepath.Base(vars["filename"]))
		contentType := mime.TypeByExtension(filepath.Ext(filename))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		file, err := ioutil.TempFile(config.Temp, "transfer-")
		if err != nil {
			log.Printf("%s", err.Error())
			http.Error(w, "Internal server error.", 500)
			return
		} else {
			defer file.Close()
			_, err = io.Copy(file, r.Body)
			if err != nil {
				os.Remove(file.Name())
				log.Printf("%s", err.Error())
				http.Error(w, "Internal server error.", 500)
				return
			} else {
				var hash string
				if hash, contentLength, err = storage.HardLinkSha512Path(file.Name(), filename); err != nil {
					log.Printf("%s", err.Error())
				} else if err == nil {
					log.Printf("Hashed %s as %s", filename, hash)
					w.Header().Set("Content-Type", "text/plain")
					fmt.Fprintf(w, "{\"sha512\":\"%s\",\"filename\":\"%s\",\"length\":%d,\"content_type\":\"%s\",\"stub\":true}", hash, filename, contentLength, contentType)
				}
			}
		}
	} else {
		http.Error(w, "403 Forbidden. Uploading is disabled.", 403)
	}
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	var filename string
	var reader io.ReadSeeker
	var modTime time.Time
	var err error
	if allowedGet() {
		vars := mux.Vars(r)
		hash := strings.TrimSpace(vars["hash"])
		option := strings.TrimSpace(vars["option"])
		if hash == "" {
			http.Error(w, "404 Not Found.", 404)
			return
		}
		if storage.has403(hash) {
			http.Error(w, "403 Forbidden.", 403)
			return
		}
		if storage.has404(hash) {
			http.Error(w, "404 Not Found.", 404)
			return
		}
		filename, reader, _, modTime, err = storage.Seeker(hash)
		if err != nil {
			if strings.Index(err.Error(), "no such file or directory") >= 0 {
				log.Printf("%s", err.Error())
				// try from peer
				var found = false
				// if not GET /hash/nopeerload, then getFromPeers
				if (strings.Index(option, "nopeerload") >= 0) == false {
					// dtfc specific
					found, filename, reader, modTime, err = getFromPeers(hash)
					if err != nil {
						if strings.Index(err.Error(), "Already peerloading") >= 0 {
							// https://tools.ietf.org/html/rfc2616#section-10.5.4
							log.Printf("%s", err.Error())
							// Retry in 2 minutes (120 seconds)
							w.Header().Set("Retry-After", fmt.Sprintf("120"))
							http.Error(w, "Service Unavailable. Try again in 2 minutes.", 503)
							return
						} else {
							log.Printf("Error while getFromPeers. %s", err.Error())
							http.Error(w, "Internal server error.", 500)
							return
						}
					}
				}
				// end try from peer
				if found == false {
					http.Error(w, "404 Not Found.", 404)
					return
				}
			} else {
				log.Printf("%s", err.Error())
				http.Error(w, "Could not retrieve file.", 500)
				return
			}
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		http.ServeContent(w, r, filename, modTime, reader)
		// refresh peers a percentage of the time
		if rand.Intn(10) == 0 {
			refreshPeerList()
		}
	} else {
		http.Error(w, "403 Forbidden. Downloading is disabled.", 403)
	}
}

func RedirectHandler(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health.html" {
		} else if ipAddrFromRemoteAddr(r.Host) == "127.0.0.1" {
		} else if ipAddrFromRemoteAddr(r.Host) == "::1" {
		} else if r.Header.Get("X-Forwarded-Proto") != "https" && r.Method == "GET" {
			http.Redirect(w, r, "https://"+r.Host+r.RequestURI, 301)
			return
		}
		h.ServeHTTP(w, r)
	}
}
