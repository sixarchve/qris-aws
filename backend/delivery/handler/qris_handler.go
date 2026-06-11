package handler

import (
	"net/http"
	"qris-latency-optimizer/usecase"

	"github.com/gin-gonic/gin"
)

type QRISHandler struct {
	usecase usecase.QRISUsecase
}

func NewQRISHandler(u usecase.QRISUsecase) *QRISHandler {
	return &QRISHandler{usecase: u}
}

func (h *QRISHandler) GenerateDynamic(c *gin.Context) {
	merchantIDStr := c.Query("merchant_id")
	amountStr := c.Query("amount")

	payload, merchant, amount, err := h.usecase.GenerateQRIS(merchantIDStr, amountStr)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "merchant not found" {
			status = http.StatusNotFound
		} else if err.Error() != "invalid amount" && err.Error() != "merchant_id is required" && err.Error() != "invalid merchant_id" {
			status = http.StatusInternalServerError
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"qris_payload": payload,
		"merchant_id":  merchant.ID,
		"amount":       amount,
	})
}
