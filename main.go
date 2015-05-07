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
	"flag"
	"fmt"
	"github.com/PuerkitoBio/ghost/handlers"
	"github.com/gorilla/mux"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

const SERVER_INFO = "dtfc"
const SERVER_VERSION = "0.0.1"

// we use these commands to reduce the amount of garbage collection golang needs to do
const cmdSHASUMFreeBSD = "/usr/local/bin/shasum"

// or on Apple using Macports
const cmdSHASUMApple = "/opt/local/bin/shasum"
const cmdSHA512 = "/sbin/sha512"
const cmdTAIL = "/usr/bin/tail"

const timeLayout = "2006-01-02 15:04:05 MST"
const timeHTTPLayout = "Mon, 2 Jan 2006 15:04:05 MST"

// parse request with maximum memory of _24Kilobits
const _24K = (1 << 20) * 24

var config struct {
	ALLOWDELETE string
	ALLOWGET    string
	ALLOWPUT    string
	Temp        string
	ME          string
	PEERS       []string
}

var storage Storage

var cmdSHASUM string

func init() {
	config.Temp = os.TempDir()
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	var err error
	if _, err = os.Lstat(cmdSHASUMFreeBSD); err == nil {
		cmdSHASUM = cmdSHASUMFreeBSD
	}
	if _, err = os.Lstat(cmdSHASUMApple); err == nil {
		cmdSHASUM = cmdSHASUMApple
	}
	if _, err = os.Lstat(cmdSHASUM); err != nil {
		log.Panic("Error while looking for shasum executable.")
	}
	if _, err = os.Lstat(cmdTAIL); err != nil {
		log.Panic("Error while looking for tail executable.")
	}

	port := flag.String("port", "8080", "port number, default: 8080")
	temp := flag.String("temp", config.Temp, "")
	basedir := flag.String("basedir", "", "")
	logpath := flag.String("log", "", "")
	provider := flag.String("provider", "local", "")
	allowdelete := flag.String("allowdelete", "true", "true or false, default: true")
	allowget := flag.String("allowget", "true", "true or false, default: true")
	allowput := flag.String("allowput", "true", "true or false, default: true")
	me := flag.String("me", "", "example http://127.0.0.1:8080")
	peerlist := flag.String("peerlist", "", "text file with one peer per line")

	flag.Parse()

	if *logpath != "" {
		f, err := os.OpenFile(*logpath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Error opening file: %v", err)
		}
		defer f.Close()
		log.SetOutput(f)
	}

	config.Temp = *temp
	config.ALLOWDELETE = *allowdelete
	config.ALLOWGET = *allowget
	config.ALLOWPUT = *allowput
	config.ME = *me
	config.PEERS, err = readLines(*peerlist)
	if err != nil {
		log.Panic("Error while reading peerlist.", err)
	}

	r := mux.NewRouter()

	r.HandleFunc("/health.html", healthHandler).Methods("GET")
	r.HandleFunc("/{hash}", getHandler).Methods("GET")

	//r.HandleFunc("/{hash}", headHandler).Methods("HEAD")

	r.HandleFunc("/{filename}", putHandler).Methods("PUT")

	r.NotFoundHandler = http.HandlerFunc(notFoundHandler)

	switch *provider {
	case "local":
		if *basedir == "" {
			log.Panic("Error basedir not set.")
		}
		storage, err = NewLocalStorage(*basedir)
	}
	if err != nil {
		log.Panic("Error while creating storage.", err)
	}

	log.Printf("%s/%s server started. listening on port: %v",
		SERVER_INFO, SERVER_VERSION, *port)
	log.Printf("using temp folder: %s, using storage provider: %s",
		config.Temp, *provider)
	log.Printf("allow delete: %s, allow get: %s, allow put: %s",
		config.ALLOWDELETE, config.ALLOWGET, config.ALLOWPUT)
	log.Printf("---------------------------")

	s := &http.Server{
		Addr:    fmt.Sprintf(":%s", *port),
		Handler: handlers.PanicHandler(RedirectHandler(handlers.LogHandler(r, handlers.NewLogOptions(log.Printf, "_default_"))), nil),
	}

	log.Panic(s.ListenAndServe())
	log.Printf("Server stopped.")
}
