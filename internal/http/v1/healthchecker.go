package v1

import (
	"net/http"
	"pasteAPI/internal/config"
	"pasteAPI/pkg/helpers"
)

func (h *Handler) HealthcheckHandler(w http.ResponseWriter, r *http.Request) {
	env := helpers.Envelope{
		"status": "development",
		"system_info": map[string]string{
			"environment": h.service.Config.Env,
			"version":     config.Version,
		},
	}
	err := helpers.WriteJSON(w, http.StatusOK, env, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}
