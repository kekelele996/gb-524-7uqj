package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"spectrum-interference-triangulation/backend/internal/dto"
	"spectrum-interference-triangulation/backend/internal/repository"
	"spectrum-interference-triangulation/backend/internal/service"
	"spectrum-interference-triangulation/backend/pkg/api"
)

type ObservationHandler struct {
	service *service.ObservationService
}

func NewObservationHandler(observationService *service.ObservationService) *ObservationHandler {
	return &ObservationHandler{service: observationService}
}

func (h *ObservationHandler) List(c *gin.Context) {
	page, pageSize := pagination(c)
	caseID, _ := strconv.ParseUint(c.Query("case_id"), 10, 32)
	stationID, _ := strconv.ParseUint(c.Query("station_id"), 10, 32)
	filter := repository.ObservationFilter{
		CaseID: uint(caseID), StationID: uint(stationID), Quality: c.Query("quality"),
		Page: page, PageSize: pageSize,
	}
	observations, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Page(c, observations, page, pageSize, total)
}

func (h *ObservationHandler) Get(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	observation, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Success(c, http.StatusOK, observation)
}

func (h *ObservationHandler) Create(c *gin.Context) {
	actor, ok := actorFromContext(c)
	if !ok {
		return
	}
	var request dto.CreateObservationRequest
	if !bindJSON(c, &request) {
		return
	}
	observation, err := h.service.Create(c.Request.Context(), request, actor)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Success(c, http.StatusCreated, observation)
}

func (h *ObservationHandler) Exclude(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	actor, ok := actorFromContext(c)
	if !ok {
		return
	}
	var request dto.ExcludeObservationRequest
	if !bindJSON(c, &request) {
		return
	}
	observation, err := h.service.Exclude(c.Request.Context(), id, request, actor)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Success(c, http.StatusOK, observation)
}

func (h *ObservationHandler) ValidateCase(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	result, err := h.service.ValidateCase(c.Request.Context(), id)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Success(c, http.StatusOK, result)
}
