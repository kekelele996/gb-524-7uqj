package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"spectrum-interference-triangulation/backend/internal/dto"
	"spectrum-interference-triangulation/backend/internal/service"
	"spectrum-interference-triangulation/backend/pkg/api"
)

type EstimateHandler struct {
	service *service.EstimateService
}

func NewEstimateHandler(estimateService *service.EstimateService) *EstimateHandler {
	return &EstimateHandler{service: estimateService}
}

func (h *EstimateHandler) List(c *gin.Context) {
	caseID, _ := strconv.ParseUint(c.Query("case_id"), 10, 32)
	estimates, err := h.service.List(c.Request.Context(), uint(caseID))
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Success(c, http.StatusOK, estimates)
}

func (h *EstimateHandler) Get(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	estimate, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Success(c, http.StatusOK, estimate)
}

func (h *EstimateHandler) Run(c *gin.Context) {
	actor, ok := actorFromContext(c)
	if !ok {
		return
	}
	var request dto.RunLocalizationRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := h.service.Run(c.Request.Context(), request, actor)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Success(c, http.StatusCreated, result)
}
