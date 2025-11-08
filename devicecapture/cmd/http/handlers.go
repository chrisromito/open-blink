package main

import (
	"devicecapture/internal/app"
	"fmt"
	"net/http"
)

func DeviceListHandler(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, e := fmt.Fprintf(w, "Hello, deviceListHandler")
		if e != nil {
			panic(e)
		}
		return
	}
}

func StreamProxyHandler(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, e := fmt.Fprintf(w, "Hello, StreamProxyHandler")
		if e != nil {
			panic(e)
		}
		return
	}
}

func HeartBeatListHandler(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, e := fmt.Fprintf(w, "Hello, HeartBeatListHandler")
		if e != nil {
			panic(e)
		}
		return
	}
}
