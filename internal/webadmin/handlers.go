package webadmin

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"
)

func (a *Admin) render(w http.ResponseWriter, page string, data any) {
	templ, err := template.ParseFS(templateFS, "templates/layout.html", "templates/"+page)
	if err != nil {
		fmt.Printf("[ERROR TEMPLATE RENDER] Template %s not found\n", page)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templ.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *Admin) sendJson(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (a *Admin) apiStart(w http.ResponseWriter, r *http.Request) {
	err := a.component.Start()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	http.Redirect(w, r, "/home", http.StatusSeeOther)
}

func (a *Admin) apiStop(w http.ResponseWriter, r *http.Request) {
	err := a.component.Stop()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	http.Redirect(w, r, "/home", http.StatusSeeOther)
}

func (a *Admin) handlerHome(w http.ResponseWriter, r *http.Request) {
	data := HomeData{
		Status: a.component.Status(),
		Time:   time.Now().Format(time.DateTime),
		Info:   a.component.GetInfo(),
	}
	a.render(w, "home.html", data)
}

func (a *Admin) handlerLogs(w http.ResponseWriter, r *http.Request) {
	data := a.component.GetLogs()
	a.sendJson(w, data)

}
