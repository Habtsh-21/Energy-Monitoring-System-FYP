package middleware

import (
	"net/http"
	"os"
)

func AdminPathPermission(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != os.Getenv("ADMIN") || pass != os.Getenv("PERMISSION_CODE") {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized Access", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
