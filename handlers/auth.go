//handlers/auth.go

package handlers

import (
	"fmt"
	"net/http"
	"os"
)

// AuthMiddleware wraps handlers requiring authentication
func (s *Server) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("auth_token")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		user, err := s.userService.VerifyToken(cookie.Value)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Add user to request context
		r.Header.Set("User-ID", user.ID)
		r.Header.Set("User-Admin", fmt.Sprintf("%v", user.IsAdmin))
		next.ServeHTTP(w, r)
	}
}

// AdminMiddleware ensures the user is an admin
func (s *Server) AdminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Admin") != "true" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		s.tmpl.ExecuteTemplate(w, "login.html", nil)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	user, token, err := s.userService.AuthenticateUser(email, password)
	if err != nil {
		data := map[string]interface{}{
			"Error": "Invalid credentials",
		}
		w.WriteHeader(http.StatusUnauthorized)
		s.tmpl.ExecuteTemplate(w, "login.html", data)
		return
	}

	// Set auth cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400, // 24 hours
	})

	// Redirect based on user role
	if user.IsAdmin {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) handleAdminSetup(w http.ResponseWriter, r *http.Request) {
	// Check if admin already exists
	adminCount, err := s.userService.GetAdminCount()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if adminCount > 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == "GET" {
		s.tmpl.ExecuteTemplate(w, "admin-setup.html", nil)
		return
	}

	setupToken := r.FormValue("setup_token")
	storedToken, err := os.ReadFile("admin_setup_token.txt")
	if err != nil || setupToken != string(storedToken) {
		data := map[string]interface{}{
			"Error": "Invalid setup token",
		}
		w.WriteHeader(http.StatusBadRequest)
		s.tmpl.ExecuteTemplate(w, "admin-setup.html", data)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	_, err = s.userService.CreateUser(email, password, true)
	if err != nil {
		data := map[string]interface{}{
			"Error": "Failed to create admin user",
		}
		w.WriteHeader(http.StatusInternalServerError)
		s.tmpl.ExecuteTemplate(w, "admin-setup.html", data)
		return
	}

	// Delete setup token file
	os.Remove("admin_setup_token.txt")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return s.AuthMiddleware(next)
}

func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return s.AuthMiddleware(s.AdminMiddleware(next))
}
