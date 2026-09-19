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
		fmt.Printf("[ERROR EXECUTE] Error: %s\n", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *Admin) sendJson(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (a *Admin) handlerLogin(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		if _, ok := a.sessionStore.Get(c.Value); ok {
			http.Redirect(w, r, "/home", http.StatusSeeOther)
			return
		}
	}
	a.render(w, "login.html", map[string]string{"Error": ""})
}

func (a *Admin) apiLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method must be post", http.StatusBadRequest)
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad form input", http.StatusBadRequest)
	}

	user := r.FormValue("login")
	password := r.FormValue("password")

	if !a.checkUser(user, password) {
		w.WriteHeader(http.StatusUnauthorized)
		a.render(w, "login.html", map[string]string{"Error": "Incorrect login or password"})
		return
	}

	token, err := a.sessionStore.Create()
	if err != nil {
		http.Error(w, "Internal error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int((24 * time.Hour).Seconds()),
		Secure:   false,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/home", http.StatusSeeOther)
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
	data := &LogData{
		Logs: a.component.GetLogs(),
	}
	a.render(w, "logs.html", data)
}
