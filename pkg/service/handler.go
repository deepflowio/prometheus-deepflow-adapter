package service

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

var (
	httpclient = http.DefaultClient
	ctx        = context.TODO()
)

func ReceiveHandler(remoteUrl string) func(c *gin.Context) {
	return func(c *gin.Context) {
		req, err := http.NewRequest("POST", remoteUrl, c.Request.Body)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		req.Header.Set("Content-Type", "application/x-protobuf")
		req.Header.Set("Content-Encoding", "snappy")
		req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

		resp, err := httpclient.Do(req.WithContext(ctx))
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		defer resp.Body.Close()

		if resp.StatusCode/100 != 2 {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				c.AbortWithError(resp.StatusCode, err)
				return
			}

			c.AbortWithError(resp.StatusCode, fmt.Errorf("body=%s", body))
			return
		}

	}

}
