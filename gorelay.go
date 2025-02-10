package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

var config configuration

type configuration struct {
	ListenRange string `json:"listen_ip"`
	Port        int    `json:"port"`
}

type incomingRequest struct {
	URL  string `json:"url"`
	Body string `json:"body"` // assume if this is empty, the request is GET
	Type string `json:"content_type"`
}

func proxyRequest(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" || r.Method != "POST" {
		http.NotFound(w, r)
		return
	}

	request, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Error encountered while reading request to proxy: %s", err.Error())))
		return
	}

	var incomingJson incomingRequest
	err = json.Unmarshal(request, &incomingJson)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Error encountered while deserializing request to JSON: %s", err.Error())))
		return
	}

	reqBody := bytes.NewReader([]byte(incomingJson.Body))

	client := http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest(http.MethodPost, incomingJson.URL, reqBody)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Error from server: %s", err.Error())))
		return
	}

	if incomingJson.Body == "" {
		req.Method = http.MethodGet
	}

	for key, val := range r.Header {
		if key == "Content-Type" {
			req.Header.Add("Content-Type", incomingJson.Type)
			continue
		}
		req.Header.Add(key, val[0])
	}

	resp, err := client.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Error retrieving response from server: %s", err.Error())))
		return
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Error retrieving response from server: %s", err.Error())))
		return
	}

	w.Write(responseBody)
}

func main() {
	http.HandleFunc("/", proxyRequest)
	http.ListenAndServe(fmt.Sprintf("%s:%d", config.ListenRange, config.Port), nil) // consider tls here
}

func init() {
	file, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatalf("%s", err.Error())
	}

	err = json.Unmarshal(file, &config)
	if err != nil {
		log.Fatalf("%s", err.Error())
	}
}
