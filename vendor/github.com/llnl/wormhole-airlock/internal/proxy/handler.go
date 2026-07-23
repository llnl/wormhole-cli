package proxy

import (
	"fmt"
	"net/http"

	"github.com/llnl/wormhole-airlock/internal/version"
)

func (d *data) authJSONHandler(w http.ResponseWriter, r *http.Request) {
	if len(d.authJson) == 0 {
		http.Error(w, "auth.json not configured", http.StatusNotFound)
		return
	}

	jsonContentHeader(w)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(d.authJson)
}

func generic404Handler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write(fmt.Append(nil, "404 Not Found"))
}

func versionHandler(w http.ResponseWriter, r *http.Request) {
	jsonContentHeader(w)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(fmt.Appendf(nil, "{\"version\": \"%s\"}\n", version.GetVersion()))
}

//

func jsonContentHeader(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
}
