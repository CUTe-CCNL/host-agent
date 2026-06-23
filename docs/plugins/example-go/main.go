package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"host-agent/pkg/pluginipc"
)

type exampleHandler struct{}

func (exampleHandler) Health(ctx context.Context) error {
	return nil
}

func (exampleHandler) HandleHTTP(ctx context.Context, req pluginipc.HTTPRequest) (pluginipc.HTTPResponse, error) {
	switch {
	case req.Method == http.MethodGet && req.Path == "/status":
		return jsonResponse(http.StatusOK, map[string]interface{}{
			"status": "ok",
			"plugin": "example-go",
		})
	case req.Method == http.MethodGet && req.Path == "/echo":
		return jsonResponse(http.StatusOK, map[string]interface{}{
			"method":    req.Method,
			"path":      req.Path,
			"raw_query": req.RawQuery,
			"headers":   req.Headers,
		})
	case req.Method == http.MethodPost && req.Path == "/echo":
		return jsonResponse(http.StatusCreated, map[string]interface{}{
			"method":    req.Method,
			"path":      req.Path,
			"raw_query": req.RawQuery,
			"body":      string(req.Body),
		})
	case req.Method == http.MethodGet && req.Path == "/items":
		return jsonResponse(http.StatusOK, map[string]interface{}{
			"resource": "items",
			"items": []map[string]string{
				{"id": "alpha", "name": "Alpha"},
				{"id": "beta", "name": "Beta"},
			},
		})
	case req.Method == http.MethodPost && req.Path == "/items":
		return jsonResponse(http.StatusCreated, map[string]interface{}{
			"resource": "items",
			"body":     string(req.Body),
		})
	case req.Method == http.MethodGet && strings.HasPrefix(req.Path, "/items/"):
		id := strings.TrimPrefix(req.Path, "/items/")
		return jsonResponse(http.StatusOK, map[string]interface{}{
			"resource": "item",
			"id":       id,
			"name":     "Item " + id,
		})
	case req.Method == http.MethodDelete && strings.HasPrefix(req.Path, "/items/"):
		id := strings.TrimPrefix(req.Path, "/items/")
		return jsonResponse(http.StatusOK, map[string]interface{}{
			"resource": "item",
			"id":       id,
			"deleted":  true,
		})
	default:
		return jsonResponse(http.StatusNotFound, map[string]string{
			"error": "route not found",
		})
	}
}

func jsonResponse(statusCode int, value interface{}) (pluginipc.HTTPResponse, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return pluginipc.HTTPResponse{}, err
	}

	return pluginipc.HTTPResponse{
		StatusCode: statusCode,
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
		},
		Body: body,
	}, nil
}

func main() {
	if err := pluginipc.Serve(context.Background(), exampleHandler{}); err != nil {
		log.New(os.Stderr, "example-plugin: ", log.LstdFlags).Fatal(err)
	}
}
