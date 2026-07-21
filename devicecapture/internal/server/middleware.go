package server

import (
	"net/http"
)

type Middleware func(http.HandlerFunc) http.HandlerFunc

// Chain chain middleware funcs
//
//	func MyHandler(w http.ResponseWriter, r *http.Request) {
//	    ...
//	}
//
// jsonCors := Chain(Middleware, JsonResponseMiddleware)
// http.HandleFunc("/my-url", jsonCors(MyHandler))
func Chain(mws ...Middleware) Middleware {
	return func(f http.HandlerFunc) http.HandlerFunc {
		for i := len(mws) - 1; i >= 0; i-- {
			f = mws[i](f)
		}
		return f
	}
}

func EnableCors(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*") // Or specific origin
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func CorsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		EnableCors(w, r)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}

func JsonResponseMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next(w, r)
	}
}
