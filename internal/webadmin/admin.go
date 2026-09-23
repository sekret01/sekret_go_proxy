package webadmin

import (
	"crypto/subtle"
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"time"

	"github.com/sekret01/sekret_go_proxy/internal/config"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

type Admin struct {
	component    Controllable
	tmpl         *template.Template
	sessionStore *SessionStore
	cfg          *config.Config
}

func (a *Admin) Start(addr string) error {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(sub))))
	// Сделать кластруктуру ServerRouter для auth и rout
	mux.HandleFunc("/login", a.handlerLogin)
	mux.HandleFunc("/api/login", a.apiLogin)

	mux.HandleFunc("/home", a.auth(a.handlerHome))
	mux.HandleFunc("/logs", a.auth(a.handlerLogs))
	mux.HandleFunc("/api/start", a.auth(a.apiStart))
	mux.HandleFunc("/api/stop", a.auth(a.apiStop))
	mux.HandleFunc("/users", a.handlerUsers)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	return srv.ListenAndServe() // http.ListenAndServe(addr, mux)
}

func (a *Admin) checkUser(user, password string) bool {
	if subtle.ConstantTimeCompare([]byte(user), []byte(a.cfg.User)) != 1 {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(password), []byte(a.cfg.Password)) == 1
}

func NewAdmin(component Controllable, cfg *config.Config, ttl time.Duration) *Admin {
	return &Admin{
		component:    component,
		tmpl:         template.Must(template.ParseFS(templateFS, "templates/*.html")),
		sessionStore: NewSessionStore(ttl),
		cfg:          cfg,
	}
}
