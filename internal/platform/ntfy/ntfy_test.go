package ntfy

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

type MockClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func TestSend(t *testing.T) {
	t.Run("Successful notification", func(t *testing.T) {
		mockClient := &MockClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(""))),
				}, nil
			},
		}

		ntfyClient := New(mockClient, "https://ntfy.sh", "test-topic")

		err := ntfyClient.Send("Test Title", "Test Message", "test,tags")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Failed notification", func(t *testing.T) {
		mockClient := &MockClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewReader([]byte("error"))),
				}, nil
			},
		}

		ntfyClient := New(mockClient, "https://ntfy.sh", "test-topic")

		err := ntfyClient.Send("Test Title", "Test Message", "test,tags")
		if err == nil {
			t.Error("expected an error, but got nil")
		}
	})
}
