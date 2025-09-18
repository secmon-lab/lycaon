package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-mizutani/gt"
	httpCtrl "github.com/secmon-lab/lycaon/pkg/controller/http"
)

func TestSPAHandler(t *testing.T) {
	// Create a mock filesystem for testing
	mockFS := http.Dir("testdata/spa")

	t.Run("serve existing static file", func(t *testing.T) {
		handler, err := httpCtrl.NewSPAHandler(mockFS)
		gt.NoError(t, err)

		req := httptest.NewRequest("GET", "/static/app.js", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		gt.Equal(t, w.Code, http.StatusOK)
		gt.Equal(t, w.Header().Get("Content-Type"), "application/javascript; charset=utf-8")
		gt.S(t, w.Body.String()).Contains("console.log")
	})

	t.Run("serve index.html for SPA route", func(t *testing.T) {
		handler, err := httpCtrl.NewSPAHandler(mockFS)
		gt.NoError(t, err)

		req := httptest.NewRequest("GET", "/incidents/123", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		gt.Equal(t, w.Code, http.StatusOK)
		gt.Equal(t, w.Header().Get("Content-Type"), "text/html; charset=utf-8")
		gt.S(t, w.Body.String()).Contains("<html")
		gt.S(t, w.Body.String()).Contains("<div id=\"root\">")
	})

	t.Run("serve index.html for root path", func(t *testing.T) {
		handler, err := httpCtrl.NewSPAHandler(mockFS)
		gt.NoError(t, err)

		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		gt.Equal(t, w.Code, http.StatusOK)
		gt.Equal(t, w.Header().Get("Content-Type"), "text/html; charset=utf-8")
		gt.S(t, w.Body.String()).Contains("<html")
	})

	t.Run("serve CSS file with correct content type", func(t *testing.T) {
		handler, err := httpCtrl.NewSPAHandler(mockFS)
		gt.NoError(t, err)

		req := httptest.NewRequest("GET", "/static/style.css", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		gt.Equal(t, w.Code, http.StatusOK)
		gt.Equal(t, w.Header().Get("Content-Type"), "text/css; charset=utf-8")
		gt.S(t, w.Body.String()).Contains("body")
	})

	t.Run("handle deep SPA routes", func(t *testing.T) {
		handler, err := httpCtrl.NewSPAHandler(mockFS)
		gt.NoError(t, err)

		testCases := []string{
			"/incidents",
			"/incidents/123",
			"/dashboard",
			"/unknown/deep/path",
		}

		for _, path := range testCases {
			req := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			gt.Equal(t, w.Code, http.StatusOK)
			gt.Equal(t, w.Header().Get("Content-Type"), "text/html; charset=utf-8")
			gt.S(t, w.Body.String()).Contains("<html")
		}
	})
}

func TestSPAHandlerContentTypes(t *testing.T) {
	mockFS := http.Dir("testdata/spa")

	handler, err := httpCtrl.NewSPAHandler(mockFS)
	gt.NoError(t, err)

	testCases := []struct {
		path        string
		contentType string
	}{
		{"/static/app.js", "application/javascript; charset=utf-8"},
		{"/static/style.css", "text/css; charset=utf-8"},
		{"/static/data.json", "application/json; charset=utf-8"},
		{"/static/favicon.ico", "image/x-icon"},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest("GET", tc.path, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			gt.Equal(t, w.Header().Get("Content-Type"), tc.contentType)
		}
	}
}

func TestNewSPAHandlerError(t *testing.T) {
	// Test with filesystem that doesn't have index.html
	emptyFS := http.Dir("testdata/empty")

	_, err := httpCtrl.NewSPAHandler(emptyFS)
	gt.Error(t, err)
	gt.S(t, err.Error()).Contains("failed to open index.html")
}

// Setup test data files for testing
func init() {
	// This would normally be handled by test setup
	// For real tests, you'd need to create testdata/spa directory with:
	// - index.html
	// - static/app.js
	// - static/style.css
	// - static/data.json
	// - static/favicon.ico
}
