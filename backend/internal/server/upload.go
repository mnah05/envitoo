package server

import (
	"net/http"
	"path"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/mnah05/envitoo/internal/dto"
	"github.com/mnah05/envitoo/internal/middleware"
)

// GeneratePresignedURL handles POST /api/upload/presigned-url.
// The frontend sends file metadata; we return a time-limited PUT URL
// that the browser uses to upload directly to R2.
func (s *Server) GeneratePresignedURL(c echo.Context) error {
	if s.storage == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "image uploads are not configured")
	}

	var req dto.PresignedURLRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := s.validate.Struct(req); err != nil {
		return validationError(c, err)
	}

	ext := strings.ToLower(path.Ext(req.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		return echo.NewHTTPError(http.StatusBadRequest, "unsupported file extension")
	}

	ctx := c.Request().Context()
	uploadURL, publicURL, err := s.storage.GeneratePresignedURL(ctx, req.Filename, req.ContentType, req.Size)
	if err != nil {
		middleware.LoggerFromContext(ctx).ErrorContext(ctx, "generate presigned URL", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate upload URL")
	}

	return c.JSON(http.StatusOK, dto.PresignedURLResponse{
		UploadURL: uploadURL,
		PublicURL: publicURL,
	})
}
