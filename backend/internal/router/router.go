package router

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"spectrum-interference-triangulation/backend/internal/constants"
	"spectrum-interference-triangulation/backend/internal/handler"
	"spectrum-interference-triangulation/backend/internal/middleware"
	"spectrum-interference-triangulation/backend/internal/service"
)

type Handlers struct {
	Support     *handler.SupportHandler
	Station     *handler.StationHandler
	Observation *handler.ObservationHandler
	Case        *handler.CaseHandler
	Estimate    *handler.EstimateHandler
}

func New(log *slog.Logger, authService *service.AuthService, handlers Handlers, corsOrigins []string) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(middleware.RequestID())
	engine.Use(middleware.AccessLog(log))
	engine.Use(middleware.Recovery(log))
	engine.Use(cors(corsOrigins))

	engine.GET("/healthz", handlers.Support.Health)
	engine.GET("/readyz", handlers.Support.Ready)

	loginLimiter := middleware.NewRateLimiter(8, 0.5)
	localizationLimiter := middleware.NewRateLimiter(5, 0.25)
	apiV1 := engine.Group("/api/v1")
	apiV1.POST("/auth/login", loginLimiter.Middleware("login"), handlers.Support.Login)

	protected := apiV1.Group("")
	protected.Use(middleware.Auth(authService))
	protected.GET("/auth/me", handlers.Support.Me)

	protected.GET("/stations", handlers.Station.List)
	protected.GET("/stations/:id", handlers.Station.Get)
	protected.GET("/stations/:id/coverage", handlers.Station.Coverage)
	protected.POST("/stations", middleware.RBAC(constants.RoleAnalyst, constants.RoleAdmin), handlers.Station.Create)
	protected.PUT("/stations/:id", middleware.RBAC(constants.RoleAnalyst, constants.RoleAdmin), handlers.Station.Update)

	protected.GET("/observations", handlers.Observation.List)
	protected.GET("/observations/:id", handlers.Observation.Get)
	protected.POST("/observations", middleware.RBAC(constants.RoleObserver, constants.RoleAnalyst, constants.RoleAdmin), handlers.Observation.Create)
	protected.POST("/observations/:id/exclude", middleware.RBAC(constants.RoleAnalyst, constants.RoleAdmin), handlers.Observation.Exclude)
	protected.GET("/cases/:id/validate-observations", handlers.Observation.ValidateCase)

	protected.GET("/cases", handlers.Case.List)
	protected.GET("/cases/:id", handlers.Case.Get)
	protected.POST("/cases", middleware.RBAC(constants.RoleObserver, constants.RoleAnalyst, constants.RoleAdmin), handlers.Case.Create)
	protected.POST("/cases/:id/transition", handlers.Case.Transition)

	protected.GET("/localizations", handlers.Estimate.List)
	protected.GET("/localizations/:id", handlers.Estimate.Get)
	protected.POST("/localizations/run", localizationLimiter.Middleware("localization"), middleware.RBAC(constants.RoleAnalyst, constants.RoleAdmin), handlers.Estimate.Run)

	protected.GET("/audits", middleware.RBAC(constants.RoleReviewer, constants.RoleAdmin), handlers.Support.ListAudits)
	return engine
}

func cors(origins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		allowed[strings.TrimSpace(origin)] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if _, ok := allowed[origin]; ok && origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
