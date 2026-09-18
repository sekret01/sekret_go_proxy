package webadmin

import (
	"embed"
	"html/template"
	"net/http"
)

//go:embed templates/*.html
var templateFS embed.FS

type Admin struct {
	component Controllable
	tmpl      *template.Template
}

func (a *Admin) Start(addr string) error {
	mux := http.NewServeMux()
	// Сделать кластруктуру ServerRouter для auth и rout
	mux.HandleFunc("/home", a.auth(a.handlerHome))
	mux.HandleFunc("/logs", a.auth(a.handlerLogs))
	mux.HandleFunc("/api/start", a.auth(a.apiStart))
	mux.HandleFunc("/api/stop", a.auth(a.apiStop))
	return http.ListenAndServe(addr, mux)
}

func NewAdmin(component Controllable) *Admin {
	return &Admin{
		component: component,
		tmpl:      template.Must(template.ParseFS(templateFS, "templates/*.html")),
	}
}
