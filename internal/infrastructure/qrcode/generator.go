package qrcode

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/skip2/go-qrcode"
)

// ErrorCorrectionLevel defines the QR code error correction capability.
// Higher levels can recover from more damage but result in larger QR codes.
type ErrorCorrectionLevel int

const (
	// ErrorCorrectionLow recovers 7% of data.
	ErrorCorrectionLow ErrorCorrectionLevel = iota
	// ErrorCorrectionMedium recovers 15% of data (recommended for most use cases).
	ErrorCorrectionMedium
	// ErrorCorrectionHigh recovers 25% of data.
	ErrorCorrectionHigh
	// ErrorCorrectionHighest recovers 30% of data (recommended for critical applications).
	ErrorCorrectionHighest
)

const (
	// DEFAULT_SIZE is the default QR code size in pixels.
	DEFAULT_SIZE = 256

	// MIN_SIZE is the minimum allowed QR code size.
	MIN_SIZE = 64

	// MAX_SIZE is the maximum allowed QR code size.
	MAX_SIZE = 2048

	// DEFAULT_ERROR_CORRECTION is the default error correction level.
	// Medium provides good balance between data capacity and error recovery.
	DEFAULT_ERROR_CORRECTION = ErrorCorrectionMedium
)

// QR code generation errors
var (
	// ErrEmptyToken indicates the token string is empty.
	ErrEmptyToken = errors.New("token cannot be empty")

	// ErrInvalidSize indicates the QR code size is outside valid range.
	ErrInvalidSize = errors.New("size must be between 64 and 2048 pixels")

	// ErrGenerationFailed indicates QR code generation failed.
	ErrGenerationFailed = errors.New("failed to generate QR code")
)

// Generator provides QR code generation functionality.
// It supports multiple output formats (PNG, SVG, Base64) with configurable
// size and error correction levels.
type Generator struct {
	errorCorrection qrcode.RecoveryLevel
}

// NewGenerator creates a new QR code generator with default settings.
// The generator uses medium error correction level by default.
func NewGenerator() *Generator {
	return &Generator{
		errorCorrection: qrcode.Medium,
	}
}

// NewGeneratorWithErrorCorrection creates a new QR code generator with custom error correction.
func NewGeneratorWithErrorCorrection(level ErrorCorrectionLevel) *Generator {
	return &Generator{
		errorCorrection: mapErrorCorrectionLevel(level),
	}
}

// GeneratePNG generates a QR code as PNG binary data.
// The context parameter allows for cancellation support in future implementations.
//
// Parameters:
//   - ctx: Context for cancellation (reserved for future async support)
//   - token: The token string to encode in the QR code
//   - size: The QR code size in pixels (must be between 64 and 2048)
//
// Returns PNG binary data or an error if generation fails.
func (g *Generator) GeneratePNG(ctx context.Context, token string, size int) ([]byte, error) {
	if token == "" {
		return nil, ErrEmptyToken
	}

	if err := validateSize(size); err != nil {
		return nil, err
	}

	// Generate QR code
	qr, err := qrcode.New(token, g.errorCorrection)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGenerationFailed, err)
	}

	// Generate PNG with specified size
	png, err := qr.PNG(size)
	if err != nil {
		return nil, fmt.Errorf("%w: PNG encoding failed: %w", ErrGenerationFailed, err)
	}

	return png, nil
}

// GeneratePNGBase64 generates a QR code as a base64-encoded PNG string.
// The output can be directly used in HTML img src attributes (data:image/png;base64,...)
//
// Parameters:
//   - ctx: Context for cancellation (reserved for future async support)
//   - token: The token string to encode in the QR code
//   - size: The QR code size in pixels (must be between 64 and 2048)
//
// Returns a base64-encoded PNG string (without data URI prefix) or an error.
func (g *Generator) GeneratePNGBase64(ctx context.Context, token string, size int) (string, error) {
	// Generate PNG
	png, err := g.GeneratePNG(ctx, token, size)
	if err != nil {
		return "", err
	}

	// Encode to base64
	encoded := base64.StdEncoding.EncodeToString(png)

	return encoded, nil
}

// GenerateSVG generates a QR code as a valid SVG document.
// The QR code PNG is generated via GeneratePNG (which handles validation),
// then embedded as a base64-encoded image within an SVG <image> element,
// producing a valid scalable vector graphic that can be used in web pages.
//
// Parameters:
//   - ctx: Context for cancellation (reserved for future async support)
//   - token: The token string to encode in the QR code
//   - size: The QR code size in pixels (must be between 64 and 2048)
//
// Returns a valid SVG XML string or an error if generation fails.
func (g *Generator) GenerateSVG(ctx context.Context, token string, size int) (string, error) {
	// Reuse GeneratePNG which handles validation and QR code generation
	pngBytes, err := g.GeneratePNG(ctx, token, size)
	if err != nil {
		return "", err
	}

	// Encode PNG as base64 and embed in SVG <image> element
	b64 := base64.StdEncoding.EncodeToString(pngBytes)
	svg := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="%d" height="%d" viewBox="0 0 %d %d">
  <image href="data:image/png;base64,%s" width="%d" height="%d"/>
</svg>`, size, size, size, size, b64, size, size)

	return svg, nil
}

// SetErrorCorrection sets the error correction level for the generator.
// This affects all subsequent QR code generations.
func (g *Generator) SetErrorCorrection(level ErrorCorrectionLevel) {
	g.errorCorrection = mapErrorCorrectionLevel(level)
}

// validateSize validates the QR code size is within acceptable range.
func validateSize(size int) error {
	if size < MIN_SIZE || size > MAX_SIZE {
		return fmt.Errorf("%w: got %d, expected between %d and %d", ErrInvalidSize, size, MIN_SIZE, MAX_SIZE)
	}
	return nil
}

// mapErrorCorrectionLevel maps our error correction enum to the library's type.
func mapErrorCorrectionLevel(level ErrorCorrectionLevel) qrcode.RecoveryLevel {
	switch level {
	case ErrorCorrectionLow:
		return qrcode.Low
	case ErrorCorrectionMedium:
		return qrcode.Medium
	case ErrorCorrectionHigh:
		return qrcode.High
	case ErrorCorrectionHighest:
		return qrcode.Highest
	default:
		return qrcode.Medium
	}
}
