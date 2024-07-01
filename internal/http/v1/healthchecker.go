package v1

import (
	"net/http"
	"pasteAPI/internal/config"
	"pasteAPI/pkg/helpers"
)

type HealthCheckOutput struct {
	S string `json:"status"`
	I struct {
		E string `json:"environment"`
		V string `json:"version"`
	} `json:"system_info"`
}

// HealthcheckHandler retrieves status of the application
//
// @Summary      Health check
// @Description  Retrieves status of the application
// @Tags         app
// @Produce      json
// @Success      200  {object}  HealthCheckOutput  "Successfully retrieved paste"
// @Failure 429 {object} v1.ErrorResponse "Too many requests, rate limit exceeded"
// @Failure      500  {object}  ErrorResponse "Internal server error"
// @Router       /api/v1/healthcheck [get]
func (h *Handler) HealthcheckHandler(w http.ResponseWriter, r *http.Request) {
	env := helpers.Envelope{
		"status": h.service.Config.Status,
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
