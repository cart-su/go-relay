package api

import (
	"bytes"
	"cart-su/go-relay/config"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

type incomingRequest struct {
	URL  string `json:"url"`
	Body string `json:"body"` // assume if this is empty, the request is GET
	Type string `json:"content_type"`
}

type Server struct {
	Engine   *gin.Engine
	server   *http.Server
	addr     string
	certFile string
	keyFile  string
	logFile  *os.File
}

func proxyRequest(c *gin.Context) {
	if c.Request.URL.Path != "/" {
		http.NotFound(c.Writer, c.Request)
		return
	}

	request, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Writer.WriteHeader(http.StatusBadRequest)
		c.Writer.Write([]byte(fmt.Sprintf("Error encountered while reading request to proxy: %s", err.Error())))
		return
	}

	var incomingJson incomingRequest
	err = json.Unmarshal(request, &incomingJson)
	if err != nil {
		c.Writer.WriteHeader(http.StatusBadRequest)
		c.Writer.Write([]byte(fmt.Sprintf("Error encountered while deserializing request to JSON: %s", err.Error())))
		return
	}

	reqBody := bytes.NewReader([]byte(incomingJson.Body))

	client := http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest(http.MethodPost, incomingJson.URL, reqBody)
	if err != nil {
		c.Writer.WriteHeader(http.StatusBadRequest)
		c.Writer.Write([]byte(fmt.Sprintf("Error from server: %s", err.Error())))
		return
	}

	if incomingJson.Body == "" {
		req.Method = http.MethodGet
	}

	for key, val := range c.Request.Header {
		if key == "Content-Type" {
			req.Header.Add("Content-Type", incomingJson.Type)
			continue
		}
		req.Header.Add(key, val[0])
	}

	resp, err := client.Do(req)
	if err != nil {
		c.Writer.WriteHeader(http.StatusBadRequest)
		c.Writer.Write([]byte(fmt.Sprintf("Error retrieving response from server: %s", err.Error())))
		return
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.Writer.WriteHeader(http.StatusBadRequest)
		c.Writer.Write([]byte(fmt.Sprintf("Error retrieving response from server: %s", err.Error())))
		return
	}

	c.Writer.Write(responseBody)
}

func NewServer() *Server {
	file, err := os.OpenFile("relay.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}

	gin.DisableConsoleColor()
	gin.DefaultWriter = file
	gin.DefaultErrorWriter = file
	gin.SetMode(gin.ReleaseMode)

	engine := gin.Default()
	engine.SetTrustedProxies(nil)

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		NextProtos: []string{"h2", "http/1.1"},
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
		},
		PreferServerCipherSuites: true,
	}

	srv := &http.Server{
		Handler:      engine,
		TLSConfig:    tlsConfig,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return &Server{
		Engine:   engine,
		server:   srv,
		addr:     fmt.Sprintf("%s:%d", config.Config.ListenRange, config.Config.Port),
		certFile: "cert.pem",
		keyFile:  "key.pem",
		logFile:  file,
	}
}

func SetRoutes(r *gin.Engine) {
	r.POST("/", func(c *gin.Context) {
		proxyRequest(c)
	})
}

func (s *Server) Run() error {
	s.server.Addr = s.addr

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		if err := s.server.ListenAndServeTLS(s.certFile, s.keyFile); err != nil && err != http.ErrServerClosed {
			log.Printf("Error starting server: %v", err)
			quit <- syscall.SIGTERM
		}
	}()

	<-quit
	log.Println("Shutting down server...")

	defer s.logFile.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}
	log.Println("Server exiting")
	return nil
}
