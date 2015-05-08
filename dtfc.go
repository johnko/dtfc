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

func getFromPeers(oldhash string) (found bool, filename string, reader io.ReadSeeker, contentLength uint64, modTime time.Time, err error) {
    var file *os.File
    var resp *http.Response
    fnre := regexp.MustCompile("filename=\".*\"")
    found = false
    for i := range config.PEERS {
        if (config.PEERS[i] != config.ME) && (found == false) {
            var url = config.PEERS[i] + oldhash
            log.Printf("trying to get from peer %s", url)
            file, err = ioutil.TempFile(config.Temp, "peer-")
            if err != nil {
                log.Printf("%s", err.Error())
                return
            }
            defer file.Close()
            resp, err = http.Get(url)
            if err == nil {
                if resp.StatusCode == 200 {
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
                    _, err = io.Copy(file, resp.Body)
                    if err != nil {
                        os.Remove(file.Name())
                        log.Printf("%s", err.Error())
                        return
                    }
                    var hash string
                    if hash, err = storage.HardLinkSha512Path(file.Name(), filename); err != nil {
                        log.Printf("%s", err.Error())
                    } else if err == nil {
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
    return
}
