package service

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"go.uber.org/zap"
)

// OCRService defines the contract for extracting raw text from an image file.
type OCRService interface {
	// ExtractText runs OCR on the supplied image file and returns the extracted plain text.
	// The method respects the provided context for cancellation and timeout.
	ExtractText(ctx context.Context, imagePath string) (string, error)
}

// ocrService is the concrete implementation that wraps the Tesseract binary.
type ocrService struct {
	logger        *zap.Logger
	tesseractBin  string        // path to the tesseract executable (e.g., "tesseract")
	defaultLang   string        // fallback language if auto‑detect fails (e.g., "eng")
	execTimeout   time.Duration // max time allowed for a single tesseract execution
	scriptLangMap map[string]string
}

// NewOCRService creates a new OCRService.
// tesseractBin should be the absolute or relative path to the tesseract binary.
// defaultLang is the language code used when auto‑detect cannot determine a language.
func NewOCRService(logger *zap.Logger, tesseractBin string, defaultLang string) OCRService {
	if tesseractBin == "" {
		tesseractBin = "tesseract"
	}
	if defaultLang == "" {
		defaultLang = "eng"
	}
	// Mapping from Tesseract script names to language codes.
	scriptLangMap := map[string]string{
		"Latin":    "eng",
		"Arabic":   "ara",
		"Cyrillic": "rus",
		"Greek":    "ell",
		"Han":      "chi_sim",
		"Hangul":   "kor",
		"Hiragana": "jpn",
		"Katakana": "jpn",
		"Devanagari": "hin",
	}
	return &ocrService{
		logger:        logger,
		tesseractBin:  tesseractBin,
		defaultLang:   defaultLang,
		execTimeout:   30 * time.Second,
		scriptLangMap: scriptLangMap,
	}
}

// ExtractText performs language auto‑detection (via OSD) and then runs OCR.
// It returns the raw extracted text or an error.
func (s *ocrService) ExtractText(ctx context.Context, imagePath string) (string, error) {
	if imagePath == "" {
		return "", errors.New("imagePath cannot be empty")
	}

	// Step 1: Detect language using OSD (Orientation and Script Detection)
	lang, err := s.detectLanguage(ctx, imagePath)
	if err != nil {
		s.logger.Warn("language detection failed, falling back to default",
			zap.Error(err), zap.String("fallback", s.defaultLang))
		lang = s.defaultLang
	}

	// Step 2: Run OCR with the detected (or fallback) language
	text, err := s.runTesseract(ctx, imagePath, lang)
	if err != nil {
		s.logger.Error("tesseract OCR execution failed",
			zap.Error(err), zap.String("image", imagePath), zap.String("lang", lang))
		return "", fmt.Errorf("ocr execution failed: %w", err)
	}
	return text, nil
}

// detectLanguage uses Tesseract's OSD mode (psm 0) to infer the script of the document.
// It returns a language code compatible with Tesseract (e.g., "eng").
// If detection fails, an error is returned and the caller should fallback.
func (s *ocrService) detectLanguage(parentCtx context.Context, imagePath string) (string, error) {
	ctx, cancel := context.WithTimeout(parentCtx, s.execTimeout)
	defer cancel()

	// Tesseract OSD: output to stdout, page segmentation mode 0 (OSD only)
	cmd := exec.CommandContext(ctx, s.tesseractBin, imagePath, "stdout", "--psm", "0")
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	s.logger.Debug("running tesseract OSD for language detection",
		zap.String("image", imagePath), zap.String("cmd", strings.Join(cmd.Args, " ")))

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("osd command failed: %w, stderr: %s", err, errBuf.String())
	}

	script, parseErr := parseScriptFromOSD(outBuf.Bytes())
	if parseErr != nil {
		return "", fmt.Errorf("failed to parse script from OSD output: %w", parseErr)
	}

	lang, ok := s.scriptLangMap[script]
	if !ok {
		return "", fmt.Errorf("script %q not mapped to a language", script)
	}
	s.logger.Info("detected script for OCR", zap.String("script", script), zap.String("lang", lang))
	return lang, nil
}

// runTesseract executes the tesseract binary with the given language and returns the extracted text.
func (s *ocrService) runTesseract(parentCtx context.Context, imagePath string, lang string) (string, error) {
	ctx, cancel := context.WithTimeout(parentCtx, s.execTimeout)
	defer cancel()

	// tesseract <image> stdout -l <lang> --psm 3 (default fully automatic)
	cmd := exec.CommandContext(ctx, s.tesseractBin, imagePath, "stdout", "-l", lang, "--psm", "3")
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	s.logger.Debug("executing tesseract OCR",
		zap.String("image", imagePath), zap.String("lang", lang), zap.String("cmd", strings.Join(cmd.Args, " ")))

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tesseract command failed: %w, stderr: %s", err, errBuf.String())
	}
	return outBuf.String(), nil
}

// parseScriptFromOSD extracts the "Script:" line from OSD output.
// Example OSD output:
//
//	Orientation: 0
//	Orientation in degrees: 0
//	Rotate: 0
//	Script: Latin
//	Confidence: 9.123456
func parseScriptFromOSD(data []byte) (string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "Script:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				return "", fmt.Errorf("malformed script line: %s", line)
			}
			script := strings.TrimSpace(parts[1])
			if script == "" {
				return "", errors.New("empty script value")
			}
			return script, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scanner error: %w", err)
	}
	return "", errors.New("script not found in OSD output")
}