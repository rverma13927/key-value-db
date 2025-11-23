package main

import (
	"fmt"
	"kvdb/kv" // Importing the local package
	"net/http"
)

var db *kv.KeyValueDb

func HandleSet(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	value := r.URL.Query().Get("value")

	if key == "" || value == "" {
		http.Error(w, "Missing key or value", http.StatusBadRequest)
		return
	}

	msg, err := db.Set(key, value)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "%s", msg)
}

func HandleGet(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")

	if key == "" {
		http.Error(w, "Missing key", http.StatusBadRequest)
		return
	}

	val, err := db.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	fmt.Fprintf(w, "%s", val)
}

func main() {
	db = kv.NewKeyValueDb()
	db.Load()

	http.HandleFunc("/set", HandleSet)
	http.HandleFunc("/get", HandleGet)

	fmt.Println("Server started at :8080")
	http.ListenAndServe(":8080", nil)
}
