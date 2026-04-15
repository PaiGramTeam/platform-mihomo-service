package server

import (
	"time"

	mihomov1 "github.com/PaiGramTeam/proto-contracts/mihomo/v1"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	kratosgrpc "github.com/go-kratos/kratos/v2/transport/grpc"

	v1 "platform-mihomo-service/api/mihomo/v1"
	"platform-mihomo-service/internal/conf"
	"platform-mihomo-service/internal/service"
)

func NewGRPCServer(bc *conf.Bootstrap, hoyoSvc *service.MihomoAccountService, sharedSvc *service.MihomoCredentialService) *kratosgrpc.Server {
	grpcConf := bc.GetServer().GetGrpc()

	srv := kratosgrpc.NewServer(
		kratosgrpc.Network(grpcConf.GetNetwork()),
		kratosgrpc.Address(grpcConf.GetAddr()),
		kratosgrpc.Timeout(time.Duration(grpcConf.GetTimeoutSeconds())*time.Second),
		kratosgrpc.Middleware(recovery.Recovery()),
	)

	if hoyoSvc != nil {
		v1.RegisterMihomoAccountServiceServer(srv, hoyoSvc)
	}
	if sharedSvc != nil {
		mihomov1.RegisterMihomoCredentialServiceServer(srv, sharedSvc)
	}

	return srv
}
