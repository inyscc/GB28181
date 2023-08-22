package gb

import (
	"encoding/xml"
	"fmt"

	"github.com/ghettovoice/gosip/sip"
	"github.com/inysc/GB28181/internal/pkg/gbsip"
	"github.com/inysc/GB28181/internal/pkg/logger"
	"github.com/inysc/GB28181/internal/pkg/parser"
	"github.com/inysc/GB28181/internal/pkg/syn"
)

func deviceConfigQueryHandler(req sip.Request, tx sip.ServerTransaction) {
	logger.Debugf("获取到的configDownload消息：\n%s", req.Body())
	defer func() {
		_ = responseAck(tx, req)
	}()

	cfg := &gbsip.DeviceBasicConfigResp{}

	if err := xml.Unmarshal([]byte(req.Body()), cfg); err != nil {
		b, err := gbkToUtf8([]byte(req.Body()))
		if err != nil {
			logger.Error(err)
			return
		}
		err = xml.Unmarshal(b, cfg)
		if err != nil {
			logger.Error(err)
			return
		}
	}

	if cfg.R.Result != "OK" {
		return
	}

	syn.HasSyncTask(fmt.Sprintf("%s_%s", syn.KeyControlDeviceConfigQuery, cfg.DeviceID.DeviceID), func(e *syn.Entity) {
		e.Ok(cfg)
	})

	_ = storage.updateDeviceBasicConfig(*cfg)
}

func deviceConfigResponseHandler(req sip.Request, tx sip.ServerTransaction) {
	r := parser.GetResultFromXML(req.Body())
	if r == "" {
		logger.Error("获取不到响应信息中的Result字段")
		return
	}

	if r == "ERROR" {
		logger.Error("发送修改配置请求失败，请检查")
	} else {
		logger.Debug("发送修改配置请求成功")
	}
}
