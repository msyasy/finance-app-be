package controllers

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// DebugIP membantu memeriksa IP apa yang diterima server di belakang proxy Render.
// Hanya aktif kalau environment DEBUG_IP=1, selain itu membalas 404.
// SEMENTARA: hapus file ini dan route-nya setelah hasilnya dicatat.
func DebugIP(c *gin.Context) {
	if os.Getenv("DEBUG_IP") != "1" {
		c.Status(http.StatusNotFound)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"remote_addr":      c.Request.RemoteAddr,
		"client_ip_gin":    c.ClientIP(),
		"x_forwarded_for":  c.GetHeader("X-Forwarded-For"),
		"x_real_ip":        c.GetHeader("X-Real-IP"),
		"cf_connecting_ip": c.GetHeader("CF-Connecting-IP"),
		"true_client_ip":   c.GetHeader("True-Client-IP"),
	})
}