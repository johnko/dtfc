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
	if _, err = os.Lstat(config.DENY+str); err != nil {
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

func getFromPeers(oldhash string) (found bool, filename string, reader io.ReadSeeker, contentLength uint64, modTime time.Time, err error) {
	var file *os.File
	var req *http.Request
	var resp *http.Response
	fnre := regexp.MustCompile("filename=\".*\"")
	found = false
	client := &http.Client{}
	for i := range config.PEERS {
		if (config.PEERS[i] != config.ME) && (found == false) {
			var url = config.PEERS[i] + oldhash
			log.Printf("trying to get from peer %s", url)
			file, err = ioutil.TempFile(config.Temp, "peer-")
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
							}
							defer resp.Body.Close()
							// save file
							_, err = io.Copy(file, resp.Body)
							if err != nil {
								os.Remove(file.Name())
								log.Printf("%s", err.Error())
							} else {
								// go through hash and hardlink process
								var hash string
								if hash, _, err = storage.HardLinkSha512Path(file.Name(), filename); err != nil {
									log.Printf("%s", err.Error())
								} else if err == nil {
									// compare oldhash to newhash so we are returning the right data and peer is not corrupt
									if oldhash == hash {
										filename, reader, _, modTime, err = storage.Seeker(hash)
										if err == nil {
											found = true
											return
										}
									}
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
