package util

import (
	"net/http"
	"time"

	"github.com/inysc/GB28181/internal/pkg/log"
	"github.com/parnurzeal/gorequest"
)

var client = gorequest.New()

func SendPost(url string, params map[string]interface{}) (b string, err error) {
	client.Post(url).SendMap(params).
		Timeout(3 * time.Second).
		End(func(response gorequest.Response, body string, errs []error) {

			if response.StatusCode != http.StatusOK || errs != nil {
				log.Error(errs)
				err = errs[0]
				return
			}
			b = body
		})
	return
}
