package ingest

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name       string
		httpClient *http.Client
		url        string
		token      string
		wantErr    bool
	}{
		{
			name:       "valid",
			httpClient: http.DefaultClient,
			url:        "https://example.com/ingest",
			token:      "test-token",
			wantErr:    false,
		},
		{
			name:       "nil httpClient",
			httpClient: nil,
			url:        "https://example.com/ingest",
			token:      "test-token",
			wantErr:    true,
		},
		{
			name:       "empty url",
			httpClient: http.DefaultClient,
			url:        "",
			token:      "test-token",
			wantErr:    true,
		},
		{
			name:       "empty token",
			httpClient: http.DefaultClient,
			url:        "https://example.com/ingest",
			token:      "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := New(tt.httpClient, tt.url, tt.token)
			if (err != nil) != tt.wantErr {
				t.Fatalf("New() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && c == nil {
				t.Error("New() == nil, want non-nil Client")
			}
		})
	}
}

func TestSend(t *testing.T) {
	var gotBody string

	serv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if got, want := r.Method, http.MethodPost; got != want {
				t.Errorf("method = %s, want %s", got, want)
			}

			if got, want := r.Header.Get("Content-Type"), "application/x-ndjson"; got != want {
				t.Errorf("Content-Type = %q, want %q", got, want)
			}

			if got, want := r.Header.Get("Authorization"), "Bearer test-token"; got != want {
				t.Errorf("Authorization = %q, want %q", got, want)
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read body: %v", err)
			}

			gotBody = string(body)
		}))

	defer serv.Close()

	c, err := New(serv.Client(), serv.URL, "test-token")
	if err != nil {
		t.Fatal(err)
	}

	rows := []any{
		map[string]string{"k": "a1"},
		map[string]string{"k": "a2"},
	}

	if err := c.Send(context.Background(), rows); err != nil {
		t.Fatalf("Send() err = %v", err)
	}

	want := "{\"k\":\"a1\"}\n{\"k\":\"a2\"}\n"
	if gotBody != want {
		t.Errorf("body = %q, want %q", gotBody, want)
	}
}

func TestSend_SplitsPayload(t *testing.T) {
	var gotBodies []string

	serv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read body: %v", err)
			}

			gotBodies = append(gotBodies, string(body))
		}))

	defer serv.Close()

	c, err := New(serv.Client(), serv.URL, "test-token")
	if err != nil {
		t.Fatal(err)
	}
	// 各行は {"k":"a1"}+改行 = 11 バイト。22 = ちょうど 2 行分
	c.maxPayload = 22

	rows := []any{
		map[string]string{"k": "a1"},
		map[string]string{"k": "a2"},
		map[string]string{"k": "a3"},
	}
	if err := c.Send(context.Background(), rows); err != nil {
		t.Fatalf("Send() err = %v", err)
	}

	// ちょうど上限の 2 行は同一チャンク、3 行目が次チャンクへ
	want := []string{
		"{\"k\":\"a1\"}\n{\"k\":\"a2\"}\n",
		"{\"k\":\"a3\"}\n",
	}
	if !reflect.DeepEqual(gotBodies, want) {
		t.Errorf("bodies = %q, want %q", gotBodies, want)
	}
}

func TestSend_Empty(t *testing.T) {
	posts := 0

	serv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			posts++
		}))
	defer serv.Close()

	c, err := New(serv.Client(), serv.URL, "test-token")
	if err != nil {
		t.Fatal(err)
	}

	if err := c.Send(context.Background(), []any{}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if posts != 0 {
		t.Errorf("posts = %d, want 0", posts)
	}
}

func TestSend_RowTooLarge(t *testing.T) {
	posts := 0

	serv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			posts++
		}))
	defer serv.Close()

	c, err := New(serv.Client(), serv.URL, "test-token")
	if err != nil {
		t.Fatal(err)
	}

	c.maxPayload = 10 // 1 行(11 バイト)が単独で超過する
	if err := c.Send(context.Background(), []any{map[string]string{"k": "a1"}}); err == nil {
		t.Error("want error, got nil")
	}

	if posts != 0 {
		t.Errorf("posts = %d, want 0", posts)
	}
}

func TestSend_MarshalError(t *testing.T) {
	posts := 0

	serv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			posts++
		}))
	defer serv.Close()

	c, err := New(serv.Client(), serv.URL, "test-token")
	if err != nil {
		t.Fatal(err)
	}

	// chan は json.Masrhal 不能
	if err := c.Send(context.Background(), []any{make(chan int)}); err == nil {
		t.Error("want error, got nil")
	}

	if posts != 0 {
		t.Errorf("posts = %d, want 0", posts)
	}
}

func TestSend_ErrorStatus(t *testing.T) {
	serv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
	defer serv.Close()

	c, err := New(serv.Client(), serv.URL, "test-token")
	if err != nil {
		t.Fatal(err)
	}

	if err := c.Send(context.Background(), []any{map[string]string{"k": "a1"}}); err == nil {
		t.Error("want error, got nil")
	}
}
