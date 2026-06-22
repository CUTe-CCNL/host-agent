package main

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"host-agent/pkg/pluginipc"
)

func TestExampleHandlerRoutesMultiplePaths(t *testing.T) {
	handler := exampleHandler{}

	tests := []struct {
		name       string
		request    pluginipc.HTTPRequest
		wantStatus int
		wantFields map[string]interface{}
	}{
		{
			name: "status",
			request: pluginipc.HTTPRequest{
				Method: http.MethodGet,
				Path:   "/status",
			},
			wantStatus: http.StatusOK,
			wantFields: map[string]interface{}{
				"status": "ok",
				"plugin": "example-go",
			},
		},
		{
			name: "echo",
			request: pluginipc.HTTPRequest{
				Method: http.MethodPost,
				Path:   "/echo",
				Body:   []byte("hello"),
			},
			wantStatus: http.StatusCreated,
			wantFields: map[string]interface{}{
				"method": http.MethodPost,
				"path":   "/echo",
				"body":   "hello",
			},
		},
		{
			name: "items collection",
			request: pluginipc.HTTPRequest{
				Method: http.MethodGet,
				Path:   "/items",
			},
			wantStatus: http.StatusOK,
			wantFields: map[string]interface{}{
				"resource": "items",
			},
		},
		{
			name: "items create",
			request: pluginipc.HTTPRequest{
				Method: http.MethodPost,
				Path:   "/items",
				Body:   []byte(`{"name":"Gamma"}`),
			},
			wantStatus: http.StatusCreated,
			wantFields: map[string]interface{}{
				"resource": "items",
				"body":     `{"name":"Gamma"}`,
			},
		},
		{
			name: "item detail",
			request: pluginipc.HTTPRequest{
				Method: http.MethodGet,
				Path:   "/items/alpha",
			},
			wantStatus: http.StatusOK,
			wantFields: map[string]interface{}{
				"resource": "item",
				"id":       "alpha",
			},
		},
		{
			name: "item delete",
			request: pluginipc.HTTPRequest{
				Method: http.MethodDelete,
				Path:   "/items/alpha",
			},
			wantStatus: http.StatusOK,
			wantFields: map[string]interface{}{
				"resource": "item",
				"id":       "alpha",
				"deleted":  true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := handler.HandleHTTP(context.Background(), tt.request)
			if err != nil {
				t.Fatalf("HandleHTTP() error = %v", err)
			}
			if response.StatusCode != tt.wantStatus {
				t.Fatalf("StatusCode = %d, want %d", response.StatusCode, tt.wantStatus)
			}

			var body map[string]interface{}
			if err := json.Unmarshal(response.Body, &body); err != nil {
				t.Fatalf("decode response body: %v", err)
			}
			for key, want := range tt.wantFields {
				if got := body[key]; got != want {
					t.Fatalf("body[%s] = %v, want %v", key, got, want)
				}
			}
		})
	}
}
