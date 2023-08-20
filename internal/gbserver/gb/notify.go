package gb

import (
	"encoding/xml"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chenjianhao66/go-GB28181/internal/pkg/cron"
	"github.com/chenjianhao66/go-GB28181/internal/pkg/log"
	"github.com/chenjianhao66/go-GB28181/internal/pkg/model"
	"github.com/chenjianhao66/go-GB28181/internal/pkg/parser"
	"github.com/ghettovoice/gosip/sip"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// xml解析心跳包结构
type keepalive struct {
	CmdType  string `xml:"CmdType"`
	SN       int    `xml:"SN"`
	DeviceID string `xml:"DeviceID"`
	Status   string `xml:"Status"`
	Info     string `xml:"Info"`
}

func keepaliveNotifyHandler(req sip.Request, tx sip.ServerTransaction) {
	keepalive := &keepalive{}
	decoder := xml.NewDecoder(strings.NewReader(req.Body()))
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		if charset == "GB2312" {
			return transform.NewReader(input, simplifiedchinese.GB18030.NewDecoder()), nil
		}
		return input, nil
	}
	if err := decoder.Decode(&keepalive); err != nil {
		log.Debugf("keepalive 消息解析xml失败：%s", err)
		return
	}
	device, ok := parser.DeviceFromRequest(req)
	if !ok {
		return
	}
	device, ok = storage.getDeviceById(device.DeviceId)
	if !ok {
		resp := sip.NewResponseFromRequest("", req, http.StatusNotFound, "device "+device.DeviceId+"not found", "")
		log.Debugf("{%s}设备不存在\n%s", device.DeviceId, resp)
		_ = tx.Respond(resp)
		return
	}

	// 更新心跳时间
	if err := storage.deviceKeepalive(device.ID); err != nil {
		log.Debugf("{%d,%s}更新心跳失败：%v", device.ID, device.DeviceId, err.Error())
	}
	err := cron.ResetTime(device.DeviceId, cron.TaskKeepLive)
	switch err {
	case nil:
	case cron.ErrNotFoud:
		err = cron.StartTask(device.DeviceId, cron.TaskKeepLive, 10*time.Second, func() {
			storage.s.Devices().Update(model.Device{DeviceId: device.DeviceId, Offline: 0})
		})
		if err != nil {
			log.Errorf("启动定时任务失败：%s", err)
		}
	default:
		log.Errorf("{%d,%s}更新心跳失败：%v", device.ID, device.DeviceId, err.Error())
	}

	resp := sip.NewResponseFromRequest("", req, http.StatusOK, http.StatusText(http.StatusOK), "")
	log.Debugf("{%d,%s}收到心跳包\n%s", device.ID, device.DeviceId, resp)
	_ = tx.Respond(resp)
}

func alarmNotifyHandler(req sip.Request, tx sip.ServerTransaction) {
	// 使用 gbsip.AlarmNotify 结构体解析
	// 自行扩展

	_ = responseAck(tx, req)
}

func mobilePositionNotifyHandler(req sip.Request, tx sip.ServerTransaction) {
	// 自行扩展

	_ = responseAck(tx, req)
}

func subscribeAlarmResponseHandler(req sip.Request, tx sip.ServerTransaction) {
	r := parser.GetResultFromXML(req.Body())
	if r == "" {
		log.Error("获取不到响应信息中的Result字段")
		return
	}

	if r == "ERROR" {
		log.Error("订阅报警信息失败，请检查")
	} else {
		log.Debug("订阅报警信息成功")
	}
	_ = responseAck(tx, req)
}

func subscribeMobilePositionResponseHandler(req sip.Request, tx sip.ServerTransaction) {
	r := parser.GetResultFromXML(req.Body())
	if r == "" {
		log.Error("获取不到响应信息中的Result字段")
		return
	}

	if r == "ERROR" {
		log.Error("订阅设备移动位置信息失败，请检查")
	} else {
		log.Debug("订阅设备移动位置信息信息成功")
	}
	_ = responseAck(tx, req)
}
