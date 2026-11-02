package api_client_go

import (
	"context"
	"net"
	"testing"

	"github.com/felix-186/api-client-go/config"
	pb "github.com/felix-186/api-client-go/eap"
	"github.com/felix-186/api-client-go/local_grpc"
	"github.com/go-kratos/kratos/v2/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

type eapBackupTestServer struct {
	pb.UnimplementedBackupServiceServer
}

func (eapBackupTestServer) PrepareExport(context.Context, *pb.BackupEmpty) (*pb.BackupTransfer, error) {
	return &pb.BackupTransfer{Id: "called-eap"}, nil
}

func TestEAPBackupClientLocalAndNetwork(t *testing.T) {
	for _, kind := range []string{"local", "network"} {
		t.Run(kind, func(t *testing.T) {
			var eap *pb.Client
			var cleanup func()
			var err error
			if kind == "local" {
				server := local_grpc.NewServer()
				pb.RegisterBackupServiceServer(server, eapBackupTestServer{})
				eap, cleanup, err = pb.NewLocalClient(config.Config{}, local_grpc.NewClient(server, nil))
			} else {
				listener := bufconn.Listen(1024 * 1024)
				server := grpc.NewServer()
				pb.RegisterBackupServiceServer(server, eapBackupTestServer{})
				go func() { _ = server.Serve(listener) }()
				t.Cleanup(server.Stop)
				eap, cleanup, err = pb.NewClient(config.Config{GatewayGrpc: "passthrough:///backup"}, nil, grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
			}
			if err != nil {
				t.Fatal(err)
			}
			defer cleanup()
			api := &Client{EAPClient: eap}
			client, closeClient, err := api.NewBackupClient(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			defer closeClient()
			got, err := client.PrepareExport(context.Background(), &pb.BackupEmpty{})
			if err != nil || got.Id != "called-eap" {
				t.Fatal("wrong backup transport", got, err)
			}
		})
	}
}

func TestEAPBackupEndpointMatchesEnvironment(t *testing.T) {
	instances := []*registry.ServiceInstance{
		{Metadata: map[string]string{"env": "other"}, Endpoints: []string{"grpc://other:9000"}},
		{Metadata: map[string]string{"env": "target"}, Endpoints: []string{"http://target:8000", "grpc://target:9000"}},
		{Metadata: map[string]string{"env": "target"}, Endpoints: []string{"grpc://second:9000"}},
	}
	if got := eapBackupEndpoint(instances, map[string]string{"env": "target"}); got != "target:9000" {
		t.Fatal(got)
	}
	if got := eapBackupEndpoint(instances, map[string]string{"env": "missing"}); got != "" {
		t.Fatal("cross-environment endpoint selected", got)
	}
}
