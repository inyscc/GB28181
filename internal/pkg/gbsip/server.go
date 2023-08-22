package gbsip

import (
	"github.com/ghettovoice/gosip"
	l "github.com/ghettovoice/gosip/log"
	"github.com/ghettovoice/gosip/sip"
	"github.com/inysc/GB28181/internal/pkg/logger"
	"github.com/inysc/GB28181/internal/pkg/option"
)

type Server struct {
	host    string
	network string
	s       gosip.Server
	c       *SipConfig
}

type RequestHandlerMap map[sip.RequestMethod]func(req sip.Request, tx sip.ServerTransaction)

type SipConfig struct {
	SipOption   *option.SIPOptions
	MysqlOption *option.MySQLOptions
	HandlerMap  RequestHandlerMap
}

func NewServer(c *SipConfig) *Server {
	s := &Server{
		host: c.SipOption.Ip + ":" + c.SipOption.Port,
		s: gosip.NewServer(
			gosip.ServerConfig{
				UserAgent: c.SipOption.UserAgent,
			},
			nil,
			nil,
			l.NewDefaultLogrusLogger(),
		),
		c: c,
	}
	s.registerHandler()
	mustSetupCommand(s)
	return s
}

func (s *Server) ListenTCP() error {
	logger.Infof("gb server listen tcp: %s", s.host)
	return s.s.Listen("tcp", s.host, nil)
}

func (s *Server) ListenUDP() error {
	logger.Infof("gb server listen udp: %s", s.host)
	return s.s.Listen("udp", s.host, nil)
}

func (s *Server) Shutdown() error {
	s.s.Shutdown()
	logger.Info("gb server shutdown...")
	return nil
}

func (s *Server) sendRequest(request sip.Request) (sip.ClientTransaction, error) {
	return s.s.Request(request)
}

func (s *Server) registerHandler() {
	for method, f := range s.c.HandlerMap {
		_ = s.s.OnRequest(method, f)
	}
}

func (s *Server) Register(method sip.RequestMethod, handler gosip.RequestHandler) {
	_ = s.s.OnRequest(method, handler)
}
