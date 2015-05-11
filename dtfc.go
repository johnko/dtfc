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
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

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
	fnre := regexp.MustCompile("filename=\".*\"")
	found = false
	client := &http.Client{}
	tmphash := filepath.Join(config.Temp, oldhash)
	for i := range config.PEERS {
		currentpeer = strings.Trim(config.PEERS[i], "")
		if (currentpeer != config.ME) && (currentpeer != "") && (found == false) {
			var url = currentpeer + oldhash
			log.Printf("trying to get from peer %s", url)
			// if tmp file exists, means last download was incomplete
			if _, err = os.Lstat(tmphash); err == nil {
				// file found, continue download with curl
				cmd := exec.Command(cmdCURL, "--continue-at", "-", "--output", tmphash, url)
				err = cmd.Run()
				if err == nil {
					found, filename, reader, modTime, err = foundHardLinkSha512Path(oldhash, tmphash)
				}
			} else {
				file, err = os.OpenFile(tmphash, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
				if err != nil {
					log.Printf("%s", err.Error())
				} else {
					defer file.Close()
					req, err = http.NewRequest("GET", url, nil)
					if err != nil {
						log.Printf("%s", err.Error())
					} else {
						// set user agent
						req.Header.Set("User-Agent", SERVER_INFO+"/"+SERVER_VERSION)
						resp, err = client.Do(req)
						if err == nil {
							if resp.StatusCode == 200 {
								// get filename
								if fnre.MatchString(resp.Header.Get("Content-Disposition")) {
									filename = strings.Replace(
										strings.Replace(
											fnre.FindString(
												resp.Header.Get("Content-Disposition")),
											"filename=",
											"",
											-1),
										"\"",
										"",
										-1)
									// ssave filename early
									saveFilename(oldhash, filename)
								}
								defer resp.Body.Close()
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
	return
}
