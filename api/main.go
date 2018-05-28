package main

import (
	"net/http"
	"sync"

	"gopkg.in/mgo.v2"
)

var (
	vars     map[*http.Request]map[string]interface{}
	varsLock sync.RWMutex
)

func OpenVars(r *http.Request) {
	varsLock.Lock()
	if vars == nil {
		vars = map[*http.Request]map[string]interface{}{}
	}
	vars[r] = map[string]interface{}{}
	varsLock.Unlock()
}

func CloseVars(r *http.Request) {
	varsLock.Lock()
	delete(vars, r)
	varsLock.Unlock()
}

func GetVars(r *http.Request, key string) interface{} {
	varsLock.RLock()
	value := vars[r][key]
	varsLock.RUnlock()
	return value
}

func SetVars(r *http.Request, key string, value interface{}) {
	varsLock.Lock()
	vars[r][key] = value
	varsLock.Unlock()
}

/* Handlers */
// Demo code (dont use it in production)
func withApiKey(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isValidApiKey(r.URL.Query().Get("key")) {
			respondErr(w, r, http.StatusUnauthorized, "invalid API key")
			return
		}
		fn(w, r)
	}
}

func isValidApiKey(key string) bool {
	return key == "demo_api_key"
}

func withData(d *mgo.Session, fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		thisDB := d.Copy()
		defer thisDB.Close()
		SetVars(r, "db", thisDB.DB("twittervotes"))
		fn(w, r)
	}
}

func withVars(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		OpenVars(r)
		defer CloseVars(r)
		fn(w, r)
	}
}

func withCORS(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Expose-Headers", "Location")
		fn(w, r)
	}
}
