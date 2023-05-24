package service

import (
	"fmt"
	"io"
	"net/http"
	"prometheus-deepflow-adapter/pkg/log"

	"github.com/gin-gonic/gin"
)

var (
	httpclient = http.DefaultClient
)

func sendSamples(remoteUrl string) gin.HandlerFunc {
	return func(c *gin.Context) {
		req, err := http.NewRequest("POST", remoteUrl, c.Request.Body)
		if err != nil {
			log.Logger.Error("msg", "build http request error", "err", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		req.Header.Set("Content-Type", "application/x-protobuf")
		req.Header.Set("Content-Encoding", "snappy")
		req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

		resp, err := httpclient.Do(req.WithContext(c.Request.Context()))
		if err != nil {
			log.Logger.Error("msg", "remote write error", "err", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Logger.Error("msg", "read remote write response error", "err", err)
				c.AbortWithError(resp.StatusCode, err)
				return
			}

			c.AbortWithError(resp.StatusCode, fmt.Errorf("body=%s", body))
			return
		}

		log.Logger.Debug("msg", fmt.Sprintf("remote write to %s success", remoteUrl))
	}
}
