// Package model implements the data model of the tiny URL service.
package model

import (
	"time"

	"gorm.io/gorm"
)

// CreateRequest is the request of create API
type CreateRequest struct {
	// LongURL is the original long URL
	LongURL string `binding:"required,http_url" json:"long_url" form:"long_url" xml:"long_url"`
	// TTLSeconds is an optional time-to-live in seconds; 0 (or omitted) means the
	// short link never expires.
	TTLSeconds int `binding:"omitempty,gt=0" json:"ttl_seconds" form:"ttl_seconds" xml:"ttl_seconds"`
}

// ShortenRequest is the request of shorten API with short URL
type ShortenRequest struct {
	// ShortURL is the shortened URL
	ShortURL string `binding:"required" json:"short_url" form:"short_url" xml:"short_url"`
}

// ShortenResponse is the response of shorten API
type ShortenResponse struct {
	TinyURL
	// Error is the error message if any error occurs
	Error string `json:"error"`
}

// TinyURL is the tiny URL model, which is used to store the short URL and its original long URL
type TinyURL struct {
	// ShortURL is the shortened URL
	ShortURL string `json:"short_url"`
	// LongURL is the original long URL
	LongURL string `json:"long_url"`
	// CreatedAt is the creation time of the short URL
	CreatedAt time.Time `json:"created_at"`
	// DeletedAt is the deletion time of the short URL
	DeletedAt gorm.DeletedAt `json:"deleted_at"`
	// ExpiresAt is when the short URL expires; null means it never expires
	ExpiresAt *time.Time `json:"expires_at"`
}
