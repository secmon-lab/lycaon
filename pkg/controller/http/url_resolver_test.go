package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-mizutani/gt"
	ctrlhttp "github.com/secmon-lab/lycaon/pkg/controller/http"
)

func TestGetFrontendURL(t *testing.T) {
	t.Run("returns configured URL when provided", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "example.com"

		result := ctrlhttp.GetFrontendURL(req, "https://configured.example.com")
		gt.Equal(t, result, "https://configured.example.com")
	})

	t.Run("constructs URL from Host header when configured URL is empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "example.com"

		result := ctrlhttp.GetFrontendURL(req, "")
		gt.Equal(t, result, "https://example.com")
	})

	t.Run("uses Alt-Used header when present (Cloud Run)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "internal.example.com"
		req.Header.Set("Alt-Used", "backstream-lycaon-507354148656.asia-northeast1.run.app")

		result := ctrlhttp.GetFrontendURL(req, "")
		gt.Equal(t, result, "https://backstream-lycaon-507354148656.asia-northeast1.run.app")
	})

	t.Run("Alt-Used takes precedence over X-Forwarded-Host", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "internal.example.com"
		req.Header.Set("X-Forwarded-Host", "public.example.com")
		req.Header.Set("Alt-Used", "backstream-lycaon-507354148656.asia-northeast1.run.app")

		result := ctrlhttp.GetFrontendURL(req, "")
		gt.Equal(t, result, "https://backstream-lycaon-507354148656.asia-northeast1.run.app")
	})

	t.Run("uses X-Forwarded-Host when present", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "internal.example.com"
		req.Header.Set("X-Forwarded-Host", "public.example.com")

		result := ctrlhttp.GetFrontendURL(req, "")
		gt.Equal(t, result, "https://public.example.com")
	})

	t.Run("handles multiple X-Forwarded-Host values", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "internal.example.com"
		req.Header.Set("X-Forwarded-Host", "public.example.com, proxy1.example.com, proxy2.example.com")

		result := ctrlhttp.GetFrontendURL(req, "")
		gt.Equal(t, result, "https://public.example.com")
	})

	t.Run("handles X-Forwarded-Host with spaces", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "internal.example.com"
		req.Header.Set("X-Forwarded-Host", "  public.example.com  , proxy.example.com")

		result := ctrlhttp.GetFrontendURL(req, "")
		gt.Equal(t, result, "https://public.example.com")
	})

	t.Run("always uses HTTPS protocol", func(t *testing.T) {
		// Even with HTTP request, should return HTTPS
		req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
		req.Host = "example.com"

		result := ctrlhttp.GetFrontendURL(req, "")
		gt.Equal(t, result, "https://example.com")
	})

	t.Run("falls back to localhost when no host is available", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = ""

		result := ctrlhttp.GetFrontendURL(req, "")
		gt.Equal(t, result, "https://localhost")
	})

	t.Run("handles Host header with port", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "example.com:8080"

		result := ctrlhttp.GetFrontendURL(req, "")
		gt.Equal(t, result, "https://example.com:8080")
	})

	t.Run("handles X-Forwarded-Host with port", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "internal.example.com"
		req.Header.Set("X-Forwarded-Host", "public.example.com:443")

		result := ctrlhttp.GetFrontendURL(req, "")
		gt.Equal(t, result, "https://public.example.com:443")
	})

	t.Run("empty X-Forwarded-Host is ignored", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "example.com"
		req.Header.Set("X-Forwarded-Host", "")

		result := ctrlhttp.GetFrontendURL(req, "")
		gt.Equal(t, result, "https://example.com")
	})

	t.Run("empty Alt-Used is ignored", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "example.com"
		req.Header.Set("Alt-Used", "")
		req.Header.Set("X-Forwarded-Host", "forwarded.example.com")

		result := ctrlhttp.GetFrontendURL(req, "")
		gt.Equal(t, result, "https://forwarded.example.com")
	})

	t.Run("handles IPv6 addresses", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "[2001:db8::1]"

		result := ctrlhttp.GetFrontendURL(req, "")
		gt.Equal(t, result, "https://[2001:db8::1]")
	})

	t.Run("handles IPv6 addresses with port", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "[2001:db8::1]:8080"

		result := ctrlhttp.GetFrontendURL(req, "")
		gt.Equal(t, result, "https://[2001:db8::1]:8080")
	})
}
