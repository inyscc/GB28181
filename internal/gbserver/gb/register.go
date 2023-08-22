package gb

import (
	"fmt"
	"net/http"

	"github.com/ghettovoice/gosip/sip"
	"github.com/inysc/GB28181/internal/pkg/cron"
	"github.com/inysc/GB28181/internal/pkg/gbsip"
	"github.com/inysc/GB28181/internal/pkg/logger"
	"github.com/inysc/GB28181/internal/pkg/parser"
)

const (
	DefaultAlgorithm = "MD5"
	WWWHeader        = "WWW-Authenticate"
	ExpiresHeader    = "Expires"
)

func RegisterHandler(req sip.Request, tx sip.ServerTransaction) {
	logger.Debugf("收到register请求\n%s", printRequest(req))
	// 判断是否存在 Authorization 字段
	if headers := req.GetHeaders("Authorization"); len(headers) > 0 {
		// 存在 Authorization 头部字段
		//authHeader := headers[0].(*SipOption.GenericHeader)
		fromRequest, ok := parser.DeviceFromRequest(req)
		if !ok {
			return
		}
		offlineFlag := false
		device, ok := storage.getDeviceById(fromRequest.DeviceId)

		if !ok {
			logger.Debug("not found from device from database")
			device = fromRequest
		}

		h := req.GetHeaders(ExpiresHeader)
		if len(h) != 1 {
			logger.Error("not found expires header from request", req)
			return
		}
		expires := h[0].(*sip.Expires)
		// 如果v=0，则代表该请求是注销请求
		if expires.Equals(new(sip.Expires)) {
			logger.Debug("expires值为0,该请求是注销请求")
			offlineFlag = true
		}
		device.Expires = expires.Value()
		logger.Infof("设备信息:  %+v\n")
		// 发送OK信息
		resp := sip.NewResponseFromRequest("", req, http.StatusOK, "ok", "")
		logger.Debugf("发送OK信息\n%s", resp)
		_ = tx.Respond(resp)

		if offlineFlag {
			// 注销请求
			_ = storage.deviceOffline(device)
			if err := cron.StopTask(device.DeviceId, cron.TaskKeepLive); err != nil {
				logger.Errorf("停止心跳检测任务失败: %s", device.DeviceId)
			}
		} else {
			// 注册请求
			if err := storage.deviceOnline(device); err != nil {
				logger.Errorf("设备上线失败请检查,%s", err)
			}
			go gbsip.DeviceInfoQuery(device)
		}
		return
	}

	// 没有存在 Authorization 头部字段
	resp := sip.NewResponseFromRequest("", req, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized), "")
	// 添加 WWW-Authenticate 头
	wwwHeader := &sip.GenericHeader{
		HeaderName: WWWHeader,
		Contents: fmt.Sprintf("Digest nonce=\"%s\", algorithm=%s, realm=\"%s\", qop=\"auth\"",
			"44010200491118000001",
			DefaultAlgorithm,
			randString(32),
		),
	}
	resp.AppendHeader(wwwHeader)
	logger.Debugf("没有Authorization头部信息，生成WWW-Authenticate头部返回：\n%s", resp)
	_ = tx.Respond(resp)
}
