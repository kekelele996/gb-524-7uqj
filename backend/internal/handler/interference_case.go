package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"spectrum-interference-triangulation/backend/internal/dto"
	"spectrum-interference-triangulation/backend/internal/service"
	"spectrum-interference-triangulation/backend/pkg/api"
)

type CaseHandler struct {
	service *service.CaseService
}

func NewCaseHandler(caseService *service.CaseService) *CaseHandler {
	return &CaseHandler{service: caseService}
}

func (h *CaseHandler) List(c *gin.Context) {
	page, pageSize := pagination(c)
	cases, total, err := h.service.List(c.Request.Context(), page, pageSize, c.Query("status"))
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Page(c, cases, page, pageSize, total)
}

func (h *CaseHandler) Get(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	item, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Success(c, http.StatusOK, item)
}

func (h *CaseHandler) Create(c *gin.Context) {
	actor, ok := actorFromContext(c)
	if !ok {
		return
	}
	var request dto.CreateCaseRequest
	if !bindJSON(c, &request) {
		return
	}
	item, err := h.service.Create(c.Request.Context(), request, actor)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Success(c, http.StatusCreated, item)
}

func (h *CaseHandler) Transition(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	actor, ok := actorFromContext(c)
	if !ok {
		return
	}
	var request dto.TransitionCaseRequest
	if !bindJSON(c, &request) {
		return
	}
	item, err := h.service.Transition(c.Request.Context(), id, request, actor)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Success(c, http.StatusOK, item)
}
