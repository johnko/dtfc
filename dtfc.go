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
	//"crypto/tls"
	//"crypto/x509"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"log"
	//"math/rand"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	if allowedDelete() {
		vars := mux.Vars(r)
		hash := vars["hash"]
		err := storage.add404(hash)
		if err != nil {
			// we only care about the status code
			http.Error(w, "404 Not Found.", 404)
			return
		}
		//_, _, _, err := storage.Head(hash)
		filename, contentLength, _, err := storage.Head(hash)
		contentType := mime.TypeByExtension(filepath.Ext(filename))
		if err != nil {
			// we only care about the status code
			http.Error(w, "404 Not Found.", 404)
			return
		} else {
			// you may want the json?
			fmt.Fprintf(w, "{\"deleted\":true,\"sha512\":\"%s\",\"filename\":\"%s\",\"length\":%d,\"content_type\":\"%s\",\"stub\":true}", hash, filename, contentLength, strings.Split(contentType, ";")[0])
		}
	} else {
		// we only care about the status code
		http.Error(w, "403 Forbidden. Deletion is disabled.", 403)
	}
}

func (s *LocalStorage) add404(hash string) error {
	var err error
	newpath := filepath.Join(s.basedir, SplitHashToPairSlash(hash))
	// mkdir -p
	if err = os.MkdirAll(newpath, 0700); err != nil && !os.IsExist(err) {
		return err
	}
	var f1 io.WriteCloser
	f1, err = os.OpenFile(filepath.Join(newpath, "404"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err == nil {
		defer f1.Close()
		io.Copy(f1, strings.NewReader(" "))
	}
	return nil
}

func (s *LocalStorage) has404(hash string) bool {
	var err error
	var result = false
	newpath := filepath.Join(s.basedir, SplitHashToPairSlash(hash), "404")
	// mkdir -p
	if _, err = os.Lstat(newpath); err != nil {
		if !os.IsExist(err) {
			result = false
		}
	} else {
		result = true
	}
	return result
}

func (s *LocalStorage) has403(hash string) bool {
	var err error
	var result = false
	newpath := filepath.Join(s.basedir, SplitHashToPairSlash(hash), "403")
	// mkdir -p
	if _, err = os.Lstat(newpath); err != nil {
		if !os.IsExist(err) {
			result = false
		}
	} else {
		result = true
	}
	return result
}

func fileNotExists(str string) bool {
	var err error
	if _, err = os.Lstat(config.DENY + str); err != nil {
		// denyput not found, so allowed = true
		return true
	}
	return false
}

func allowedPut() bool {
	return fileNotExists("put")
}

func allowedGet() bool {
	return fileNotExists("get")
}

func allowedHead() bool {
	return fileNotExists("head")
}

func allowedDelete() bool {
	return fileNotExists("delete")
}

func refreshPeerList() error {
	newhash, err := Sha512(config.PEERLIST, "")
	if err != nil {
		log.Printf("Error while hashing peerlist. %s", err.Error())
	} else {
		if config.PEERLISTHASH != newhash {
			config.PEERS, err = readLines(config.PEERLIST)
			if err != nil {
				log.Printf("Error while reading peerlist. %s", err.Error())
			} else {
				log.Printf("config.PEERS: %s", config.PEERS)
				config.PEERLISTHASH = newhash
			}
		}
	}
	return err
}

func foundHardLinkSha512Path(oldhash string, oldfile string) (found bool, filename string, reader io.ReadSeeker, modTime time.Time, err error) {
	found = false
	var hash string
	if hash, err = Sha512(oldfile, ""); err != nil {
		log.Printf("%s", err.Error())
		return
	} else {
		// compare oldhash to newhash so we are returning the right data and peer is not corrupt
		if oldhash == hash {
			_, _, err = storage.HardLinkSha512Path(oldfile, filename)
			if err != nil {
				log.Printf("%s", err.Error())
				return
			}
			filename, reader, _, modTime, err = storage.Seeker(hash)
			if err == nil {
				found = true
			}
		}
	}
	return
}

func getFromPeers(oldhash string) (found bool, filename string, reader io.ReadSeeker, modTime time.Time, err error) {
	var file *os.File
	var req *http.Request
	var resp *http.Response
	var currentpeer string
	var pgrepoutput []byte
	var curlrunning string
	fnre := regexp.MustCompile("filename=\".*\"")
	found = false
	// from golang example
	/*
			const rootPEM = `
		-----BEGIN CERTIFICATE-----
		-----END CERTIFICATE-----`
			roots := x509.NewCertPool()
			ok := roots.AppendCertsFromPEM([]byte(rootPEM))
			if !ok {
				panic("failed to parse root certificate")
			}
			tr := &http.Transport{
				TLSClientConfig: &tls.Config{RootCAs: roots},
			}
			client := &http.Client{Transport: tr}
	*/
	// end golang example
	// if already peerloading a hash, wait
	if PEERLOADING[oldhash] == true {
		// TODO return 503 and Retry-After
		err = fmt.Errorf("Already peerloading %s.", oldhash)
		return
	}
	// if some process is using our hash, peerloading is true
	pgrepoutput, err = exec.Command(cmdPGREP, "-l", "-f", "curl*continue*"+oldhash).Output()
	if err != nil {
		log.Printf("pgrepoutput: %s", err.Error())
		// reset err so we don't throwup
		err = nil
	} else {
		curlrunning = strings.TrimSpace(fmt.Sprintf("%s", pgrepoutput))
		log.Printf("curlrunning: %s.", curlrunning)
		if curlrunning != "" {
			PEERLOADING[oldhash] = true
			err = fmt.Errorf("Already peerloading %s.", oldhash)
			return
		}
	}
	// track hashes being peerloaded
	PEERLOADING[oldhash] = true
	client := &http.Client{}
	tmphash := filepath.Join(config.Temp, oldhash)
	for i := range config.PEERS {
		currentpeer = strings.TrimSpace(config.PEERS[i])
		if (currentpeer != config.ME) && (currentpeer != "") && (found == false) {
			var url = currentpeer + oldhash + "/nopeerload"
			log.Printf("trying to get from peer %s", url)
			// if tmp file exists, means last download was incomplete
			if _, err = os.Lstat(tmphash); err == nil {
				// tmpfile found, continue download with curl
				cmd := exec.Command(cmdCURL, "--continue-at", "-", "--output", tmphash, url)
				err = cmd.Run()
				if err == nil {
					// no error then curl is done
					found, filename, reader, modTime, err = foundHardLinkSha512Path(oldhash, tmphash)
					if err == nil {
						if found == false {
							// no error, but not found, then the tmp is corrupt
							os.Remove(tmphash)
						}
					}
				}
			}
			// this direct fetch happens here after the curl in case the hash mismatch and remove tmphash
			if found == false {
				req, err = http.NewRequest("GET", url, nil)
				if err != nil {
					log.Printf("%s", err.Error())
				} else {
					// set user agent
					req.Header.Set("User-Agent", SERVER_INFO+"/"+SERVER_VERSION)
					resp, err = client.Do(req)
					if err == nil {
						defer resp.Body.Close()
						// get filename
						contentDisposition := resp.Header.Get("Content-Disposition")
						if fnre.MatchString(contentDisposition) {
							log.Printf("Got Content-Disposition: %s.", contentDisposition)
							filename = strings.Replace(
								strings.Replace(
									fnre.FindString(contentDisposition),
									"filename=",
									"",
									-1),
								"\"",
								"",
								-1)
							// save filename early
							storage.saveFilename(oldhash, filename)
						}
						if resp.StatusCode == 200 {
							file, err = os.OpenFile(tmphash, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
							if err != nil {
								log.Printf("%s", err.Error())
							} else {
								defer file.Close()
								// save file
								_, err = io.Copy(file, resp.Body)
								if err != nil {
									// download interrupted
									log.Printf("%s", err.Error())
								} else {
									// go through hash and hardlink process
									found, filename, reader, modTime, err = foundHardLinkSha512Path(oldhash, file.Name())
								}
							}
						}
					}
				}
			}
		}
	}
	PEERLOADING[oldhash] = false
	return
}

func refreshPeersHandler(w http.ResponseWriter, r *http.Request) {
	refreshPeerList()
}
