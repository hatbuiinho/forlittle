package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type errorResponse struct {
	Error string `json:"error"`
}

func badRequest(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
}

func unauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, errorResponse{Error: message})
}

func notFound(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, errorResponse{Error: message})
}

func internalServerError(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, errorResponse{Error: message})
}

func internalServerErrorWithCause(c *gin.Context, message string, err error) {
	log.Printf("%s: %v", message, err)
	internalServerError(c, message)
}
