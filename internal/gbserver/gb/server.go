package gb

import (
	"github.com/ghettovoice/gosip/sip"
	"github.com/inysc/GB28181/internal/gbserver/storage/mysql"
	"github.com/inysc/GB28181/internal/pkg/gbsip"
	"github.com/inysc/GB28181/internal/pkg/logger"
	"github.com/inysc/GB28181/internal/pkg/option"
)

type Server struct {
	server *gbsip.Server
}

type SipConfig struct {
	SipOption   *option.SIPOptions
	MysqlOption *option.MySQLOptions
}

func NewServer(c *SipConfig) *Server {
	s := &Server{
		gbsip.NewServer(
			&gbsip.SipConfig{
				SipOption:   c.SipOption,
				MysqlOption: c.MysqlOption,
				HandlerMap:  createHandlerMap(),
			}),
	}
	storage.s = mysql.GetMySQLFactory()
	return s
}

func (s *Server) ListenTCP() error {
	return s.server.ListenTCP()
}

func (s *Server) ListenUDP() error {
	return s.server.ListenUDP()
}

func (s *Server) Close() error {
	_ = s.server.Shutdown()
	logger.Info("gb server shutdown...")
	return nil
}

func createHandlerMap() gbsip.RequestHandlerMap {
	m := make(map[sip.RequestMethod]func(req sip.Request, tx sip.ServerTransaction))
	m[sip.REGISTER] = RegisterHandler
	m[sip.MESSAGE] = MessageHandler
	return m
}
