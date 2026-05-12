package handlers

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

type ExtensionReleaseHandler struct {
	ReleasesDir string
}

func (h ExtensionReleaseHandler) ServeUpdateManifest(c *gin.Context) {
	slug := c.Param("slug")
	releasePath, err := h.resolvePath(slug, "update.xml")
	if err != nil {
		notFound(c, "extension release not found")
		return
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.File(releasePath)
}

func (h ExtensionReleaseHandler) ServeAsset(c *gin.Context) {
	slug := c.Param("slug")
	filename := c.Param("filename")

	releasePath, err := h.resolvePath(slug, filename)
	if err != nil {
		notFound(c, "extension asset not found")
		return
	}

	switch strings.ToLower(filepath.Ext(releasePath)) {
	case ".crx":
		c.Header("Content-Type", "application/x-chrome-extension")
	case ".xml":
		c.Header("Content-Type", "application/xml; charset=utf-8")
	}

	c.File(releasePath)
}

func (h ExtensionReleaseHandler) resolvePath(slug, filename string) (string, error) {
	if slug == "" || filename == "" {
		return "", errors.New("missing path segment")
	}

	if slug != filepath.Base(slug) || filename != filepath.Base(filename) {
		return "", errors.New("invalid release path")
	}

	baseDir, err := filepath.Abs(h.ReleasesDir)
	if err != nil {
		return "", err
	}

	targetPath := filepath.Join(baseDir, slug, filename)
	cleanTarget := filepath.Clean(targetPath)

	relativePath, err := filepath.Rel(baseDir, cleanTarget)
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(relativePath, "..") {
		return "", errors.New("path traversal blocked")
	}

	info, err := os.Stat(cleanTarget)
	if err != nil {
		return "", err
	}

	if info.IsDir() {
		return "", errors.New("directories are not served")
	}

	return cleanTarget, nil
}
