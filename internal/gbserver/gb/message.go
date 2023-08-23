package gb

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/ghettovoice/gosip"
	"github.com/ghettovoice/gosip/sip"
	"github.com/inysc/GB28181/internal/pkg/gbsip"
	"github.com/inysc/GB28181/internal/pkg/logger"
	"github.com/inysc/GB28181/internal/pkg/model"
	"github.com/inysc/GB28181/internal/pkg/parser"
	"github.com/inysc/GB28181/internal/pkg/syn"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

var (
	messageHandler = map[string]gosip.RequestHandler{
		// 通知
		"Notify:Keepalive":      keepaliveNotifyHandler,
		"Notify:Alarm":          alarmNotifyHandler,
		"Notify:MobilePosition": mobilePositionNotifyHandler,

		// 响应
		// 查询设备信息响应
		"Response:DeviceInfo": deviceInfoHandler,

		// 设备配置请求应答
		"Response:DeviceConfig": deviceConfigResponseHandler,

		// 查询设备目录信息响应
		"Response:Catalog": catalogHandler,

		// 查询设备状态信息响应
		"Response:DeviceStatus": deviceStatusHandler,

		// 查询设备配置信息响应
		"Response:ConfigDownload": deviceConfigQueryHandler,

		// 发起报警订阅信息响应
		"Response:Alarm": subscribeAlarmResponseHandler,

		// 发起设备移动位置订阅响应
		"Response:MobilePosition": subscribeMobilePositionResponseHandler,
	}
)

func MessageHandler(req sip.Request, tx sip.ServerTransaction) {
	logger.Debugf("收到MESSAGE请求\n%s", printRequest(req))
	if l, ok := req.ContentLength(); !ok || l.Equals(0) {
		resp := sip.NewResponseFromRequest("", req, http.StatusOK, http.StatusText(http.StatusOK), "")
		logger.Debug("该MESSAGE消息的消息体长度为0，返回OK\n%s", resp)
		_ = tx.Respond(resp)
	}
	body := req.Body()
	cmdType, err := parser.GetCmdTypeFromXML(body)
	logger.Debug("解析出的命令：", cmdType)
	if err != nil {
		return
	}
	handler, ok := messageHandler[cmdType]
	if !ok {
		logger.Warn("不支持的Message方法实现")
		return
	}
	handler(req, tx)
}

const (
	resultOK = "OK"
)

type (
	CmdType struct {
		CmdType string `xml:"CmdType"`
	}

	DeviceID struct {
		DeviceID string `xml:"DeviceID"`
	}

	SN struct {
		SN string `xml:"SN"`
	}
)

type deviceInfo struct {
	CmdType      string `xml:"CmdType"`
	SN           string `xml:"SN"`
	DeviceID     string `xml:"DeviceID"`
	Result       string `xml:"Result"`
	DeviceName   string `xml:"DeviceName"`
	Manufacturer string `xml:"Manufacturer"`
	Model        string `xml:"Model"`
	Firmware     string `xml:"Firmware"`
}

func deviceInfoHandler(req sip.Request, tx sip.ServerTransaction) {
	d := &deviceInfo{}
	if err := parser.XmlStringDecode(req.Body(), d); err != nil {
		logger.Error("解析deviceInfo响应包出错", err)
		return
	}

	if d.Result != resultOK {
		logger.Errorf("查询设备信息请求结果为%s，请检查", d.Result)
		return
	}

	dev := model.Device{
		Name:         d.DeviceName,
		Manufacturer: d.Manufacturer,
		Model:        d.Model,
		Firmware:     d.Firmware,
		DeviceId:     d.DeviceID,
	}

	if err := storage.updateDeviceInfo(dev); err != nil {
		logger.Error("更新设备信息失败", err)
		return
	}
	err := responseAck(tx, req)
	if err != nil {
		return
	}
}

type DeviceCatalogResponse struct {
	Name xml.Name `xml:"Response"`
	CmdType
	SN
	DeviceID
	SunNum        string     `xml:"SunNum"`
	DeviceListNum string     `xml:"Num,attr"`
	DeviceList    DeviceList `xml:"DeviceList"`
}

type DeviceList struct {
	Name  xml.Name      `xml:"DeviceList"`
	Items []CatalogItem `xml:"Item"`
}

type CatalogItem struct {
	XmlName xml.Name `xml:"Item"`
	DeviceID
	Name         string `xml:"Name"`
	Manufacturer string `xml:"Manufacturer"`
	Model        string `xml:"Model"`
	Owner        string `xml:"Owner"`
	CivilCode    string `xml:"CivilCode"`
	Address      string `xml:"Address"`
	Parental     string `xml:"Parental"`
	ParentID     string `xml:"ParentID"`
	SafetyWay    string `xml:"SafetyWay"`
	RegisterWay  string `xml:"RegisterWay"`
	Secrecy      string `xml:"Secrecy"`
	Status       string `xml:"Status"`
}

func (i CatalogItem) ConvertToChannel() model.Channel {
	c := model.NewChannelMust(i.DeviceID.DeviceID)
	c.Name = i.Name
	c.Manufacturer = i.Manufacturer
	c.Model = i.Model
	c.Owner = i.Owner
	c.CivilCode = i.CivilCode
	c.Address = i.Address
	c.Parental = i.Parental
	c.ParentID = i.ParentID
	c.SafetyWay = i.SafetyWay
	c.RegisterWay = i.RegisterWay
	c.Secrecy = i.Secrecy
	c.Status = i.Status
	return c
}

func gbkToUtf8(s []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewDecoder())
	buffer := bytes.Buffer{}
	_, err := buffer.ReadFrom(reader)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func catalogHandler(req sip.Request, tx sip.ServerTransaction) {
	defer func() {
		_ = responseAck(tx, req)
	}()

	catalog := DeviceCatalogResponse{}

	err := parser.XmlStringDecode(req.Body(), &catalog)

	if err != nil {
		// maybe charset is GBK, so take request body convert utf8
		b, err := gbkToUtf8([]byte(req.Body()))

		if err != nil {
			logger.Error(err)
			return
		}
		err = parser.XmlStringDecode(string(b), &catalog)
		if err != nil {
			logger.Error(err)
			return
		}
	}
	// save catalog object to database
	storage.syncChannel(catalog)
}

func deviceStatusHandler(req sip.Request, tx sip.ServerTransaction) {
	defer func() {
		_ = responseAck(tx, req)
	}()
	status := &gbsip.DeviceStatus{}

	if err := parser.XmlStringDecode(req.Body(), status); err != nil {
		logger.Error(err)
		return
	}
	syn.HasSyncTask(fmt.Sprintf("%s_%s", syn.KeyQueryDeviceStatus, status.DeviceID.DeviceID), func(e *syn.Entity) {
		e.Ok(*status)
	})
}

func printRequest(req sip.Request) string {
	var buffer bytes.Buffer

	buffer.WriteString(req.StartLine() + "\r\n")
	for _, v := range req.Headers() {
		buffer.WriteString(v.String() + "\r\n")
	}
	buffer.WriteString("\r\n")
	body := req.Body()
	text, err := simplifiedchinese.GB18030.NewDecoder().String(body)
	if err != nil {
		buffer.WriteString(body)
	} else {
		buffer.WriteString(text)
	}

	return buffer.String()
}
