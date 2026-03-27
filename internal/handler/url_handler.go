package handler

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"gorm.io/gorm"

	"url-shortener/internal/model"
	"url-shortener/internal/repository"
)

type URLHandler struct {
	repo repository.URLRepository
}

func NewURLHandler(repo repository.URLRepository) *URLHandler {
	return &URLHandler{repo: repo}
}

type CreateShortURLRequest struct {
	URL string `json:"url"`
}

type CreateShortURLResponse struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	ShortCode string    `json:"shortCode"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

const (
	shortCodeLength = 8
	maxAttempts     = 5
)

var base62Alphabet = []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")

func (h *URLHandler) RegisterRoutes(r *gin.Engine) {
	r.POST("/shorten", h.CreateShortURL)
	r.GET("/shorten/:shortCode", h.GetOriginalURL)
	r.PUT("/shorten/:shortCode", h.UpdateShortURL)
	r.DELETE("/shorten/:shortCode", h.DeleteShortURL)
}

func (h *URLHandler) CreateShortURL(c *gin.Context) {
	var req CreateShortURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"errors": []string{"invalid JSON body"},
		})
		return
	}

	if err := validateLongURL(req.URL); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"errors": []string{err.Error()},
		})
		return
	}

	ctx := c.Request.Context()
	var created *model.URL

	for attempt := 0; attempt < maxAttempts; attempt++ {
		shortCode, err := generateShortCode(shortCodeLength)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"errors": []string{"failed to generate short code"},
			})
			return
		}

		u := &model.URL{
			ShortID: shortCode,
			LongURL: req.URL,
		}

		if err := h.repo.Create(ctx, u); err != nil {
			if isUniqueViolation(err) {
				continue // collision, retry with another short code
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"errors": []string{"failed to create short url"},
			})
			return
		}

		created = u
		break
	}

	if created == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"errors": []string{"could not generate a unique short code"},
		})
		return
	}

	createdAt := created.CreatedAt.UTC()
	updatedAt := created.UpdatedAt.UTC()

	c.JSON(http.StatusCreated, CreateShortURLResponse{
		ID:        fmt.Sprintf("%d", created.ID),
		URL:       created.LongURL,
		ShortCode: created.ShortID,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	})
}

func (h *URLHandler) GetOriginalURL(c *gin.Context) {
	shortCode := c.Param("shortCode")
	if shortCode == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"errors": []string{"short code not found"},
		})
		return
	}

	ctx := c.Request.Context()
	url, err := h.repo.FindByShortID(ctx, shortCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"errors": []string{"short code not found"},
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"errors": []string{"failed to retrieve url"},
		})
		return
	}

	createdAt := url.CreatedAt.UTC()
	updatedAt := url.UpdatedAt.UTC()

	c.JSON(http.StatusOK, CreateShortURLResponse{
		ID:        fmt.Sprintf("%d", url.ID),
		URL:       url.LongURL,
		ShortCode: url.ShortID,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	})
}

func (h *URLHandler) UpdateShortURL(c *gin.Context) {
	shortCode := c.Param("shortCode")
	if shortCode == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"errors": []string{"short code not found"},
		})
		return
	}

	var req CreateShortURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"errors": []string{"invalid JSON body"},
		})
		return
	}

	if err := validateLongURL(req.URL); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"errors": []string{err.Error()},
		})
		return
	}

	updatedURL, err := h.repo.UpdateLongURLByShortID(c.Request.Context(), shortCode, req.URL)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"errors": []string{"short code not found"},
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"errors": []string{"failed to update short url"},
		})
		return
	}

	c.JSON(http.StatusOK, CreateShortURLResponse{
		ID:        fmt.Sprintf("%d", updatedURL.ID),
		URL:       updatedURL.LongURL,
		ShortCode: updatedURL.ShortID,
		CreatedAt: updatedURL.CreatedAt.UTC(),
		UpdatedAt: updatedURL.UpdatedAt.UTC(),
	})
}

func (h *URLHandler) DeleteShortURL(c *gin.Context) {
	shortCode := c.Param("shortCode")
	if shortCode == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"errors": []string{"short code not found"},
		})
		return
	}

	err := h.repo.DeleteByShortID(c.Request.Context(), shortCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"errors": []string{"short code not found"},
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"errors": []string{"failed to delete short url"},
		})
		return
	}

	c.Status(http.StatusNoContent)
}

func validateLongURL(raw string) error {
	if raw == "" {
		return errors.New("url is required")
	}

	parsed, err := url.ParseRequestURI(raw)
	if err != nil {
		return errors.New("url must be a valid URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("url must start with http:// or https://")
	}
	if parsed.Host == "" {
		return errors.New("url must include a host")
	}
	return nil
}

func generateShortCode(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("short code length must be positive")
	}

	max := big.NewInt(int64(len(base62Alphabet)))
	out := make([]byte, length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = base62Alphabet[n.Int64()]
	}

	return string(out), nil
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		// PostgreSQL unique_violation
		return string(pqErr.Code) == "23505"
	}
	return false
}

