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
	"strings"
	"time"
)

const SERVER_INFO = "dtfc"
const SERVER_VERSION = "0.0.4"

// we use these commands to reduce the amount of garbage collection golang needs to do
const cmdSHASUMFreeBSD = "/usr/local/bin/shasum"
const cmdSHASUMApple = "/opt/local/bin/shasum"

const cmdSHA512 = "/sbin/sha512"
const cmdTAIL = "/usr/bin/tail"

const cmdCURLFreeBSD = "/usr/local/bin/curl"
const cmdCURLApple = "/opt/local/bin/curl"

const cmdPGREPFreeBSD = "/bin/pgrep"
const cmdPGREPApple = "/usr/bin/pgrep"

const timeLayout = "2006-01-02 15:04:05 MST"
const timeHTTPLayout = "Mon, 2 Jan 2006 15:04:05 MST"

// parse request with maximum memory of _24Kilobits
const _24K = (1 << 20) * 24

var config struct {
	DENY         string
	Temp         string
	ME           string
	PEERS        []string
	PEERLIST     string
	PEERLISTHASH string
}

var PEERLOADING map[string]bool

var storage Storage

var cmdSHASUM string
var cmdCURL string
var cmdPGREP string

func init() {
	config.Temp = os.TempDir()
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	PEERLOADING = make(map[string]bool)

	var err error

	// cmdTAIL
	if _, err = os.Lstat(cmdTAIL); err != nil {
		log.Panic("Error while looking for tail executable.")
	}

	// cmdSHASUM
	if _, err = os.Lstat(cmdSHASUMFreeBSD); err == nil {
		cmdSHASUM = cmdSHASUMFreeBSD
	}
	if _, err = os.Lstat(cmdSHASUMApple); err == nil {
		cmdSHASUM = cmdSHASUMApple
	}
	if _, err = os.Lstat(cmdSHASUM); err != nil {
		log.Panic("Error while looking for shasum executable.")
	}

	// cmdCURL
	if _, err = os.Lstat(cmdCURLFreeBSD); err == nil {
		cmdCURL = cmdCURLFreeBSD
	}
	if _, err = os.Lstat(cmdCURLApple); err == nil {
		cmdCURL = cmdCURLApple
	}
	if _, err = os.Lstat(cmdCURL); err != nil {
		log.Panic("Error while looking for curl executable.")
	}

	// cmdPGREP
	if _, err = os.Lstat(cmdPGREPFreeBSD); err == nil {
		cmdPGREP = cmdPGREPFreeBSD
	}
	if _, err = os.Lstat(cmdPGREPApple); err == nil {
		cmdPGREP = cmdPGREPApple
	}
	if _, err = os.Lstat(cmdPGREP); err != nil {
		log.Panic("Error while looking for pgrep executable.")
	}

	port := flag.String("port", "8080", "port number, default: 8080")
	temp := flag.String("temp", config.Temp, "")
	basedir := flag.String("basedir", "", "")
	logpath := flag.String("log", "", "")
	provider := flag.String("provider", "local", "")
	deny := flag.String("deny", "", "path to deny files")
	me := flag.String("me", "", "example http://127.0.0.1:8080/")
	melist := flag.String("melist", "", "text file with first line as me")
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
	config.DENY = *deny
	config.PEERLIST = *peerlist
	// usually string empty test is == ""
	// the following test can be nil because *string can be nil
	if me != nil {
		config.ME = *me
	}
	if (config.ME == "") && (melist != nil) {
		var arraystring []string
		arraystring, err = readLines(*melist)
		if err != nil {
			log.Panic("Error while reading melist.", err)
		} else {
			config.ME = strings.TrimSpace(arraystring[0])
		}
	}
	if strings.TrimSpace(config.ME) == "" {
		log.Panic("Error while trying to figure out me.")
	}
	log.Printf("config.ME: %s", config.ME)
	config.PEERS, err = readLines(config.PEERLIST)
	if err != nil {
		log.Panic("Error while reading peerlist.", err)
	}
	log.Printf("config.PEERS: %s", config.PEERS)
	config.PEERLISTHASH, err = Sha512(config.PEERLIST, "")
	if err != nil {
		log.Panic("Error while hashing peerlist.", err)
	}

	r := mux.NewRouter()

	r.HandleFunc("/health.html", healthHandler).Methods("GET")
	r.HandleFunc("/refreshpeers.html", refreshPeersHandler).Methods("PUT")
	r.HandleFunc("/{hash}/{option}", getHandler).Methods("GET")
	r.HandleFunc("/{hash}", getHandler).Methods("GET")

	r.HandleFunc("/{hash}", headHandler).Methods("HEAD")

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
	log.Printf("---------------------------")

	s := &http.Server{
		Addr:    fmt.Sprintf(":%s", *port),
		Handler: handlers.PanicHandler(RedirectHandler(handlers.LogHandler(r, handlers.NewLogOptions(log.Printf, "_default_"))), nil),
	}

	log.Panic(s.ListenAndServe())
	log.Printf("Server stopped.")
}
