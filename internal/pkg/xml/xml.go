package xml

import (
	"encoding/xml"
	"io"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type Name = xml.Name

func StringDecode(body string, x any) error {
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
