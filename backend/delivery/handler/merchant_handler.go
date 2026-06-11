package handler

import (
	"net/http"
	"qris-latency-optimizer/usecase"

	"github.com/gin-gonic/gin"
)

type MerchantHandler struct {
	usecase usecase.MerchantUsecase
}

func NewMerchantHandler(u usecase.MerchantUsecase) *MerchantHandler {
	return &MerchantHandler{usecase: u}
}

func (h *MerchantHandler) GetMerchants(c *gin.Context) {
	merchants, err := h.usecase.GetActiveMerchants()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch merchants"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"merchants": merchants})
}
