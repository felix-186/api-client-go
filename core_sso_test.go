package api_client_go

import (
	"context"
	"testing"

	"github.com/felix-186/api-client-go/api"
	"github.com/felix-186/api-client-go/auth"
	"github.com/felix-186/api-client-go/config"
	"github.com/felix-186/api-client-go/core"
	"github.com/felix-186/api-client-go/local_grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type ssoExchangeTestServer struct {
	core.UnimplementedAppServiceServer
	t       *testing.T
	project string
	err     error
	tokens  int
}

func (s *ssoExchangeTestServer) GetToken(_ context.Context, req *api.TokenRequest) (*api.Response, error) {
	if req.GetAk() != "client-ak" || req.GetSk() != "client-sk" {
		s.t.Fatalf("expected configured client credentials")
	}
	s.tokens++
	return &api.Response{Status: true, Code: 200, Result: []byte(`{"accessToken":"platform-token","expiresAt":4102444800}`)}, nil
}

func (s *ssoExchangeTestServer) ExchangeSSO(ctx context.Context, req *core.SSOExchangeRequest) (*api.Response, error) {
	s.t.Helper()
	md, _ := metadata.FromIncomingContext(ctx)
	if got := md.Get(config.XRequestHeaderAuthorization); len(got) != 1 || got[0] != "platform-token" {
		s.t.Fatalf("unexpected token metadata: %v", got)
	}
	if got := md.Get(config.XRequestProject); len(got) != 1 || got[0] != s.project {
		s.t.Fatalf("unexpected project metadata: %v", got)
	}
	if req.GetUsername() != "zhangsan" {
		s.t.Fatalf("username = %q", req.GetUsername())
	}
	if s.err != nil {
		return nil, s.err
	}
	return &api.Response{Status: true, Code: 200, Info: "ok", Result: []byte(`{"accessToken":"user-token","tokenType":"Bearer","userId":"u1"}`)}, nil
}

func TestExchangeSSO(t *testing.T) {
	for _, project := range []string{"project1", ""} {
		t.Run("project="+project, func(t *testing.T) {
			expected := project
			if expected == "" {
				expected = config.XRequestProjectDefault
			}
			server := local_grpc.NewServer()
			impl := &ssoExchangeTestServer{t: t, project: expected}
			core.RegisterAppServiceServer(server, impl)
			authClient := auth.NewClient(config.Config{Type: config.Project, ProjectId: expected, AK: "client-ak", SK: "client-sk"})
			lowLevel, cleanup, err := core.NewLocalClient(config.Config{}, local_grpc.NewClient(server, func() *auth.Client { return authClient }))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(cleanup)
			authClient.SetClient(lowLevel)
			client := &Client{CoreClient: lowLevel}
			var result map[string]interface{}
			if err := client.ExchangeSSO(context.Background(), project, "zhangsan", &result); err != nil {
				t.Fatal(err)
			}
			if result["accessToken"] != "user-token" || result["userId"] != "u1" {
				t.Fatalf("unexpected login result: %#v", result)
			}
			impl.err = status.Error(codes.Unauthenticated, "expired token")
			if err := client.ExchangeSSO(context.Background(), project, "zhangsan", &result); err == nil {
				t.Fatal("RPC failure must reach the caller")
			}
			if impl.tokens != 1 {
				t.Fatalf("expected client token caching, got %d token requests", impl.tokens)
			}
		})
	}
}

func TestExchangeSSORejectsEmptyUsername(t *testing.T) {
	client := &Client{}
	for _, username := range []string{"", "  "} {
		if err := client.ExchangeSSO(context.Background(), "project1", username, nil); err == nil {
			t.Fatalf("expected input validation for %q", username)
		}
	}
}
