package parser

import (
	"crypto/rand"
	"encoding/xml"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"strings"

	"github.com/beevik/etree"
	"github.com/inysc/GB28181/internal/pkg/logger"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type (
	Name = xml.Name

	QueryType   string
	ControlType string

	WithKeyValue func(element *etree.Element)
)

const (
	DeviceConfig    ControlType = "DeviceConfig"
	DeviceControl   ControlType = "DeviceControl"
	DeviceUpgrade   ControlType = "DeviceUpgrade"
	RecordDownload  ControlType = "RecordDownload"
	DeviceCapture   ControlType = "DeviceCapture"
	DeviceRestart   ControlType = "DeviceRestart"
	DeviceRestore   ControlType = "DeviceRestore"
	DeviceDelete    ControlType = "DeviceDelete"
	DeviceKeepalive ControlType = "DeviceKeepalive"
	DeviceStatus    ControlType = "DeviceStatus"

	DeviceStatusCmdType   QueryType = "DeviceStatus"
	CatalogCmdType        QueryType = "Catalog"
	DeviceInfoCmdType     QueryType = "DeviceInfo"
	RecordInfoCmdType     QueryType = "RecordInfo"
	AlarmCmdType          QueryType = "Alarm"
	ConfigDownloadCmdType QueryType = "ConfigDownload"
	PresetQueryCmdType    QueryType = "PresetQuery"
	MobilePositionCmdType QueryType = "MobilePosition"
)

// CreateQueryXML create catalog query request xml of sip message and return
func CreateQueryXML(cmd QueryType, deviceId string, kvs ...WithKeyValue) (string, error) {
	document := etree.NewDocument()
	document.CreateProcInst("xml", "version=\"1.0\" encoding=\"GB2312\"")
	query := document.CreateElement("Query")
	query.CreateElement("CmdType").CreateText(string(cmd))
	query.CreateElement("SN").CreateText(getSN())
	query.CreateElement("DeviceID").CreateText(deviceId)

	for _, kv := range kvs {
		kv(query)
	}

	document.Indent(2)
	body, err := document.WriteToString()
	if err != nil {
		logger.Error(err)
		return "", errors.Wrap(err, "encoding catalog query request xml fail")
	}
	return body, nil
}

func CreateControlXml(cmd ControlType, deviceId string, kvs ...WithKeyValue) (string, error) {
	document := etree.NewDocument()
	document.CreateProcInst("xml", "version=\"1.0\" encoding=\"GB2312\"")
	query := document.CreateElement("Control")
	query.CreateElement("CmdType").CreateText(string(cmd))
	query.CreateElement("SN").CreateText(getSN())
	query.CreateElement("DeviceID").CreateText(deviceId)

	for _, kv := range kvs {
		kv(query)
	}

	document.Indent(2)
	body, err := document.WriteToString()
	if err != nil {
		logger.Error(err)
		return "", errors.Wrap(err, "encoding device control request xml fail")
	}
	return body, nil

}

// WithFilePath create 'FilePath' item of xml by value
func WithFilePath(value string) WithKeyValue {
	return func(element *etree.Element) {
		element.CreateElement("FilePath").CreateText(value)
	}
}

// WithPTZCmd create 'PTZCmd' item of xml by value
func WithPTZCmd(ptz string) WithKeyValue {
	return func(element *etree.Element) {
		element.CreateElement("PTZCmd").CreateText(ptz)
	}
}

// WithBasicParams create 'BasicParam' item of xml by value
func WithBasicParams(name string, expiration, heartBeatInterval, heartBeatCount int) WithKeyValue {
	return func(element *etree.Element) {
		p := element.CreateElement("BasicParam")
		p.CreateElement("Name").CreateText(name)
		p.CreateElement("Expiration").CreateText(cast.ToString(expiration))
		p.CreateElement("HeartBeatInterval").CreateText(cast.ToString(heartBeatCount))
		p.CreateElement("HeartBeatCount").CreateText(cast.ToString(heartBeatInterval))
	}
}

func WithAlarmQuery() WithKeyValue {
	return func(element *etree.Element) {
		element.CreateElement("StartAlarmPriority").CreateText("0")
		element.CreateElement("EndAlarmPriority").CreateText("0")
		element.CreateElement("AlarmMethod").CreateText("0")
	}
}

// WithCustomKV create 'k' item of xml by 'v'
func WithCustomKV(k, v string) WithKeyValue {
	return func(element *etree.Element) {
		element.CreateElement(k).CreateText(v)
	}
}

func getSN() string {
	i, err := rand.Int(rand.Reader, big.NewInt(10))
	if err != nil {
		return "9876"
	}
	return strconv.FormatInt((i.Int64()+1)*9876, 10)
}

// GetCmdTypeFromXML 根据body获取XML配置文件中的根元素
func GetCmdTypeFromXML(body string) (key string, err error) {
	decoder := xml.NewDecoder(strings.NewReader(body))
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		if charset == "GB2312" {
			return transform.NewReader(input, simplifiedchinese.GB18030.NewDecoder()), nil
		}
		return input, nil
	}
	var (
		isRoot, isCmdType = false, false
		root, cmdType     string
	)

re:
	for t, err := decoder.Token(); err == nil || err == io.EOF; t, err = decoder.Token() {
		switch token := t.(type) {
		case xml.StartElement:
			if !isRoot {
				root = token.Name.Local
				isRoot = true
			}
			if token.Name.Local == "CmdType" {
				isCmdType = true
			}
		case xml.CharData:
			if isCmdType {
				cmdType = string(token)
				break re
			}
		default:
		}
	}

	key = fmt.Sprintf("%s:%s", root, cmdType)
	return
}

func GetResultFromXML(body string) string {
	_, v, err := getSpecificFromXML(body, "Result")
	if err != nil {
		logger.Error(err)
		return ""
	}
	return v
}

// 在body查询指定key的value，然后返回
func getSpecificFromXML(body, key string) (k, v string, err error) {
	decoder := xml.NewDecoder(strings.NewReader(body))
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		if charset == "GB2312" {
			return transform.NewReader(input, simplifiedchinese.GB18030.NewDecoder()), nil
		}
		return input, nil
	}

	isSpecific := false

re:
	for t, err := decoder.Token(); err == nil || err == io.EOF; t, err = decoder.Token() {
		switch token := t.(type) {
		case xml.StartElement:
			if token.Name.Local == key {
				isSpecific = true
			}
		case xml.CharData:
			if isSpecific {
				v = string(token)
				break re
			}
		default:
		}
	}
	return key, v, nil
}

func XmlStringDecode(body string, x any) error {
	decoder := xml.NewDecoder(strings.NewReader(body))
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		switch charset {
		case "GB2312":
			return transform.NewReader(input, simplifiedchinese.GB18030.NewDecoder()), nil
		default:
			return input, nil
		}
	}
	return decoder.Decode(x)
}
