package handlers

import (
	"SimpleGit/models"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

type SSHKeyRequest struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
}

func (s *Server) handleAddSSHKey(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	user, err := s.getUserFromRequest(r)
	if err != nil {
		models.HandleError(w, r, models.NewUnauthorizedError("Not authenticated"))
		return
	}

	var req SSHKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		models.HandleError(w, r, models.NewBadRequestError("Invalid request body"))
		return
	}

	// Validate input
	req.Name = strings.TrimSpace(req.Name)
	req.PublicKey = strings.TrimSpace(req.PublicKey)

	if req.Name == "" || req.PublicKey == "" {
		models.HandleError(w, r, models.NewBadRequestError("Name and public key are required"))
		return
	}

	// Add SSH key
	key, err := s.userService.AddSSHKey(user.ID, req.Name, req.PublicKey)
	if err != nil {
		models.HandleError(w, r, models.NewInternalError("Failed to add SSH key").WithError(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(key)
}

func (s *Server) handleListSSHKeys(w http.ResponseWriter, r *http.Request) {
	user, err := s.getUserFromRequest(r)
	if err != nil {
		models.HandleError(w, r, models.NewUnauthorizedError("Not authenticated"))
		return
	}

	keys, err := s.userService.GetUserSSHKeys(user.ID)
	if err != nil {
		models.HandleError(w, r, models.NewInternalError("Failed to get SSH keys").WithError(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(keys)
}

func (s *Server) handleDeleteSSHKey(w http.ResponseWriter, r *http.Request) {
    user, err := s.getUserFromRequest(r)
    if err != nil {
        models.HandleError(w, r, models.NewUnauthorizedError("Not authenticated"))
        return
    }

    keyID := mux.Vars(r)["id"]
    if keyID == "" {
        models.HandleError(w, r, models.NewBadRequestError("Key ID is required"))
        return
    }

    if err := s.userService.DeleteSSHKey(user.ID, keyID); err != nil {
        models.HandleError(w, r, models.NewInternalError("Failed to delete SSH key").WithError(err))
        return
    }

    w.WriteHeader(http.StatusNoContent)
}
