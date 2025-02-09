package pages

import (
	"fmt"
	"net/http"
)

// DefaultPage provides a basic HTML page with links to API endpoints
func DefaultPage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `
		<!DOCTYPE html>
		<html>
		<head><title>Cert-Monitor</title></head>
		<body>
			<h1>Cert-Monitor</h1>
			<ul>
				<li><a href="/metrics">Metrics</a></li>
				<li><a href="/healthz">Health Check</a></li>
				<li><a href="/version">Version</a></li>
			</ul>
			<h2>Checks</h2>
			<ul>
				<li><a href="/check/secrets">Check for expiring certs</a></li>
				<li><a href="/check/cert-manager">Check cert-manager certificates</a></li>
				<li><a href="/check/ingress">Check Ingress SSL status</a></li>
			</ul>
			<h2>Status Pages</h2>
			<ul>
				<li><a href="/status/secrets">View TLS secret status</a></li>
				<li><a href="/status/cert-manager">View cert-manager certificate status</a></li>
				<li><a href="/status/ingress">View Ingress SSL status</a></li>
			</ul>
		</body>
		</html>
	`)
}
