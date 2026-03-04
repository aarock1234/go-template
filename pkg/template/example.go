package template

import (
	"context"
	"net/http"
)

type exampleRequest struct {
	Example string `json:"example"`
}

type exampleResponse struct {
	ResponseBytes []byte
}

// Example demonstrates a POST request with JSON payload and raw byte response.
func (t *Template) Example(ctx context.Context) (*exampleResponse, error) {
	payload := exampleRequest{
		Example: "example",
	}

	var resp []byte
	if err := t.doRequest(ctx, http.MethodPost, "/api/all", payload, &resp); err != nil {
		return nil, err
	}

	return &exampleResponse{
		ResponseBytes: resp,
	}, nil
}
