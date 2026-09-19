package webadmin

import (
	"net/http"
)

func (a *Admin) auth(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookie)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if _, ok := a.sessionStore.Get(cookie.Value); !ok {
			http.SetCookie(w, &http.Cookie{
				Name:   sessionCookie,
				Value:  "",
				Path:   "/",
				MaxAge: -1,
			})
			http.Redirect(w, r, "/login", http.StatusSeeOther)
		}
		f(w, r)
	}
}
