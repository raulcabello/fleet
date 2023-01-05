package helm

import (
	"crypto/subtle"
	"github.com/rancher/fleet/integrationtests/cli/testenv"
	"net/http"
	"strings"
)

const (
	username = "user"
	password = "pass"
)

type repository struct {
	server *http.Server
}

func (r *repository) startRepository(authEnabled bool) {
	r.server = &http.Server{Addr: ":3000"}
	r.server.Handler = getHandler(authEnabled)
	go func() {
		err := r.server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			panic("error creating helm repository: " + err.Error())
		}
	}()
}

func getHandler(authEnabled bool) http.Handler {
	fs := http.FileServer(http.Dir(testenv.AssetsPath + "helmrepository"))
	if !authEnabled {
		return fs
	}
	return &authHandler{fs: fs}

}

func (r *repository) stopRepository() error {
	return r.server.Close()
}

type authHandler struct {
	fs http.Handler
}

func (h *authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok || subtle.ConstantTimeCompare([]byte(strings.TrimSpace(user)), []byte(username)) != 1 || subtle.ConstantTimeCompare([]byte(strings.TrimSpace(pass)), []byte(password)) != 1 {
		w.WriteHeader(401)
		w.Write([]byte("Unauthorised."))
		return
	}
	h.fs.ServeHTTP(w, r)
}
