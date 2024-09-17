package opensearch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/opensearch-project/opensearch-go"
	"github.com/opensearch-project/opensearch-go/opensearchapi"
)

type OpenSearchClient struct {
	client *opensearch.Client
}

func NewOpenSearchClient(url string) (*OpenSearchClient, error) {
	cfg := opensearch.Config{
		Addresses: []string{url},
	}
	client, err := opensearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &OpenSearchClient{client: client}, nil
}

func (osc *OpenSearchClient) IndexDocument(index string, docID string, document interface{}) error {
	body, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	req := opensearchapi.IndexRequest{
		Index:      index,
		DocumentID: docID,
		Body:       strings.NewReader(string(body)),
		Refresh:    "true",
	}

	res, err := req.Do(context.Background(), osc.client)
	if err != nil {
		return fmt.Errorf("failed to index document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error indexing document ID=%s: %s", docID, res.String())
	}
	return nil
}

func (osc *OpenSearchClient) Search(index string, query map[string]interface{}) (string, error) {
	body, err := json.Marshal(query)
	if err != nil {
		return "", fmt.Errorf("failed to marshal query: %w", err)
	}

	req := opensearchapi.SearchRequest{
		Index: []string{index},
		Body:  strings.NewReader(string(body)),
	}

	res, err := req.Do(context.Background(), osc.client)
	if err != nil {
		return "", fmt.Errorf("failed to execute search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return "", fmt.Errorf("error searching index=%s: %s", index, res.String())
	}

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(responseBody), nil
}
