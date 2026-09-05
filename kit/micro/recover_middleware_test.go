package micro

import (
	"context"
	"net"
	"testing"
	"time"

	kratosgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type panickingHealthServer struct {
	grpc_health_v1.UnimplementedHealthServer
}

func (*panickingHealthServer) Watch(*grpc_health_v1.HealthCheckRequest, grpc.ServerStreamingServer[grpc_health_v1.HealthCheckResponse]) error {
	panic("private server details")
}

func TestNewRecoversStreamHandlerPanic(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := New(listener.Addr().String(), func(server *kratosgrpc.Server) {
		grpc_health_v1.RegisterHealthServer(server, &panickingHealthServer{})
	}, kratosgrpc.Listener(listener), kratosgrpc.CustomHealth())

	go func() {
		_ = server.Start(context.Background())
	}()

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_ = server.Stop(ctx)
	})

	var conn *grpc.ClientConn
	conn, err = grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	client := grpc_health_v1.NewHealthClient(conn)
	var stream grpc.ServerStreamingClient[grpc_health_v1.HealthCheckResponse]
	stream, err = client.Watch(ctx, &grpc_health_v1.HealthCheckRequest{})
	require.NoError(t, err)

	_, err = stream.Recv()
	require.Equal(t, codes.Internal, status.Code(err))
	require.NotContains(t, err.Error(), "private server details")

	_, err = client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	require.Equal(t, codes.Unimplemented, status.Code(err))
}
