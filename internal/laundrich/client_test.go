package laundrich

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name       string
		httpClient *http.Client
		userAgent  string
		wantErr    bool
	}{
		{
			name:       "valid",
			httpClient: http.DefaultClient,
			userAgent:  "test-agent",
			wantErr:    false,
		},
		{
			name:       "nil httpClient",
			httpClient: nil,
			userAgent:  "test-agent",
			wantErr:    true,
		},
		{
			name:       "empty userAgent",
			httpClient: http.DefaultClient,
			userAgent:  "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := New(tt.httpClient, tt.userAgent)
			if (err != nil) != tt.wantErr {
				t.Fatalf("New() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && c == nil {
				t.Error("New() == nil, want non-nil Client")
			}
		})
	}
}

func TestFetchShopsRaw(t *testing.T) {
	serv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if got, want := r.Method, http.MethodPost; got != want {
				t.Errorf("method = %s, want %s", got, want)
			}

			if got, want := r.Header.Get("Content-Type"), "application/x-www-form-urlencoded"; got != want {
				t.Errorf("Content-type = %q, want %q", got, want)
			}

			if got, want := r.Header.Get("Referer"), refererURL; got != want {
				t.Errorf("Referer = %q, want %q", got, want)
			}

			if got, want := r.Header.Get("User-Agent"), "test-agent"; got != want {
				t.Errorf("User-Agent = %q, want %q", got, want)
			}

			if err := r.ParseForm(); err != nil {
				t.Errorf("ParseForm() error = %v", err)
			}

			if got, want := r.PostForm.Get("className"), "AilsGoogleMap"; got != want {
				t.Errorf("className = %q, want %q", got, want)
			}

			if _, ok := r.PostForm["data[PREFECTURES]"]; !ok {
				t.Error("form key data[PREFECTURES] is missing")
			}

			if got, want := r.PostForm.Get("data[PREFECTURES]"), ""; got != want {
				t.Errorf("data[PREFECTURES] = %q, want %q", got, want)
			}

			// dummy response
			w.Header().Set("Content-Type", "text/html; charset=UTF-8")
			_, _ = w.Write([]byte(`[{"ANKSHOPID":"1","NUMFACILITYLIST":"1"}]`))
		}))

	defer serv.Close()

	c, err := New(serv.Client(), "test-agent")
	if err != nil {
		t.Fatal(err)
	}

	c.baseURL = serv.URL

	raw, err := c.FetchShopsRaw(context.Background())
	if err != nil {
		t.Fatalf("FetchShopsRaw() error = %v", err)
	}

	want := `[{"ANKSHOPID":"1","NUMFACILITYLIST":"1"}]`
	if got := string(raw); got != want {
		t.Errorf("raw = %q, want %q", got, want)
	}
}

func TestFetchShopsRaw_ErrorResponses(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{
			name:   "non-200 status",
			status: http.StatusServiceUnavailable,
			body:   "[]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serv := httptest.NewServer(
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(tt.status)
						_, _ = w.Write([]byte(tt.body))
					},
				),
			)
			defer serv.Close()

			c, err := New(serv.Client(), "test-agent")
			if err != nil {
				t.Fatal(err)
			}

			c.baseURL = serv.URL

			if _, err := c.FetchShopsRaw(context.Background()); err == nil {
				t.Error("want err, got nil")
			}
		})
	}
}

func TestFetchStatusesRaw(t *testing.T) {
	serv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if got, want := r.Method, http.MethodGet; got != want {
				t.Errorf("method = %s, want %s", got, want)
			}

			if got, want := r.Header.Get("User-Agent"), "test-agent"; got != want {
				t.Errorf("User-Agent = %q, want %q", got, want)
			}

			q := r.URL.Query()

			if got, want := q.Get("className"), "AilsShopDetailOperationalStatus"; got != want {
				t.Errorf("className = %q, want %q", got, want)
			}

			if got, want := q.Get("shopId"), "11001646"; got != want {
				t.Errorf("shopId = %q, want %q", got, want)
			}

			// dummy response
			w.Header().Set("Content-Type", "text/html; charset=UTF-8")
			_, _ = w.Write([]byte(`[{"ANKSHOPID":"11001646","ANKK_GU":"T"}]`))
		}))

	defer serv.Close()

	c, err := New(serv.Client(), "test-agent")
	if err != nil {
		t.Fatal(err)
	}

	c.baseURL = serv.URL

	raw, err := c.FetchStatusesRaw(context.Background(), "11001646")
	if err != nil {
		t.Fatalf("FetchStatusesRaw() error = %v", err)
	}

	want := `[{"ANKSHOPID":"11001646","ANKK_GU":"T"}]`
	if got := string(raw); got != want {
		t.Errorf("raw = %q, want %q", got, want)
	}
}
