package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

func helloHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL.Path)
	fmt.Fprintf(w, "Hello, World!")
}

func getPhotoHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path parameter is required", http.StatusBadRequest)
		return
	}

	// Intentionally vulnerable - allows directory traversal
	content, err := ioutil.ReadFile(path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading file: %v", err), http.StatusInternalServerError)
		return
	}

	w.Write(content)
}

func main() {
	http.HandleFunc("/", helloHandler)
	http.HandleFunc("/getPhoto", getPhotoHandler)

	fmt.Println("Server starting on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}
