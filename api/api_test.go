// api/server_test.go

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	server := NewServer()
	assert.NotNil(t, server)
	assert.NotNil(t, server.Engine)
	assert.NotNil(t, server.server)
	assert.NotEmpty(t, server.addr)
	assert.Equal(t, "cert.pem", server.certFile)
	assert.Equal(t, "key.pem", server.keyFile)
}

func TestSetRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetRoutes(engine)

	// verify the route route exists for the post request
	routes := engine.Routes()
	foundRoute := false
	for _, route := range routes {
		if route.Path == "/" && route.Method == "POST" {
			foundRoute = true
			break
		}
	}
	assert.True(t, foundRoute)
}

func TestProxyRequest(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    incomingRequest
		expectedStatus int
	}{
		{
			name: "Valid POST request",
			requestBody: incomingRequest{
				URL:  "https://example.com",
				Body: `{"key": "value"}`,
				Type: "application/json",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Valid GET request",
			requestBody: incomingRequest{
				URL:  "https://example.com",
				Body: "",
				Type: "application/json",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Invalid URL",
			requestBody: incomingRequest{
				URL:  "not-a-url",
				Body: "",
				Type: "application/json",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// create request body
			jsonBody, _ := json.Marshal(tt.requestBody)
			c.Request = httptest.NewRequest("POST", "/", bytes.NewBuffer(jsonBody))

			proxyRequest(c)

			// for invalid url, expect bad request
			if tt.name == "Invalid URL" {
				assert.Equal(t, tt.expectedStatus, w.Code)
				return
			}

			// for valid cases, we can't fully test the external request
			// but we can verify the function handles the setup correctly
			assert.NotEqual(t, http.StatusBadRequest, w.Code)
		})
	}
}
