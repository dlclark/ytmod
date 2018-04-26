package main

import (
	"flag"
	"net/http"

	"github.com/gorilla/mux"

	youtube "google.golang.org/api/youtube/v3"
)

var oauthBaseURL = "localhost:8080"
var viewBaseURL = "localhost:8081"

var chanID = flag.String("chan", "", "Youtube channel ID to monitor")

var yt *youtube.Service

func main() {
	flag.Parse()

	if chanID == nil || len(*chanID) == 0 {
		flag.Usage()
		return
	}

	c := getClient(youtube.YoutubeForceSslScope)
	svc, err := youtube.New(c)
	if err != nil {
		panic(err)
	}
	yt = svc

	r := mux.NewRouter()
	r.HandleFunc("/", getLatestComments).Methods(http.MethodGet)
	r.HandleFunc("/remove/{id}", removeComment).Methods(http.MethodPost)
	http.ListenAndServe(viewBaseURL, r)
}
