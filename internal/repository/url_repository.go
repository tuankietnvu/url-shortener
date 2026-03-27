package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"url-shortener/internal/model"
)

type URLRepository interface {
	Create(ctx context.Context, url *model.URL) error
	FindByShortID(ctx context.Context, shortID string) (*model.URL, error)
	UpdateLongURLByShortID(ctx context.Context, shortID string, longURL string) (*model.URL, error)
	IncrementClick(ctx context.Context, shortID string) error
}

type gormURLRepository struct {
	db *gorm.DB
}

func NewURLRepository(db *gorm.DB) URLRepository {
	return &gormURLRepository{
		db: db,
	}
}

func (r *gormURLRepository) Create(ctx context.Context, url *model.URL) error {
	if err := r.db.WithContext(ctx).Create(url).Error; err != nil {
		return fmt.Errorf("repository: create url: %w", err)
	}
	return nil
}

func (r *gormURLRepository) FindByShortID(ctx context.Context, shortID string) (*model.URL, error) {
	var url model.URL

	if err := r.db.WithContext(ctx).Where("short_id = ?", shortID).First(&url).Error; err != nil {
		return nil, fmt.Errorf("repository: find url by short_id %q: %w", shortID, err)
	}

	return &url, nil
}

func (r *gormURLRepository) UpdateLongURLByShortID(ctx context.Context, shortID string, longURL string) (*model.URL, error) {
	var url model.URL

	if err := r.db.WithContext(ctx).Where("short_id = ?", shortID).First(&url).Error; err != nil {
		return nil, fmt.Errorf("repository: find url by short_id %q for update: %w", shortID, err)
	}

	url.LongURL = longURL
	if err := r.db.WithContext(ctx).Save(&url).Error; err != nil {
		return nil, fmt.Errorf("repository: update long_url by short_id %q: %w", shortID, err)
	}

	return &url, nil
}

func (r *gormURLRepository) IncrementClick(ctx context.Context, shortID string) error {
	if err := r.db.WithContext(ctx).
		Model(&model.URL{}).
		Where("short_id = ?", shortID).
		UpdateColumn("clicks", gorm.Expr("clicks + ?", 1)).Error; err != nil {
		return fmt.Errorf("repository: increment clicks for short_id %q: %w", shortID, err)
	}

	return nil
}

