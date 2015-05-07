#!/bin/sh

export GOPATH=/opt/go/

go get "github.com/PuerkitoBio/ghost/handlers"
go get "github.com/gorilla/mux"
go get "github.com/kennygrant/sanitize"
