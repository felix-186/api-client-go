package api_client_go

import (
	"context"
	stdjson "encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/felix-186/api-client-go/api"
	"github.com/felix-186/api-client-go/auth"
	"github.com/felix-186/api-client-go/config"
	"github.com/felix-186/api-client-go/core"
	"github.com/felix-186/api-client-go/local_grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestProjectAvailable(t *testing.T) {
	start := time.Date(2026, 9, 23, 0, 0, 0, 0, time.Local)
	deadline := start.AddDate(0, 0, 10)
	row := stdjson.RawMessage(`{"id":"baktest","status":true,"grant":{"startTime":"2026-09-23","validityPeriod":10}}`)
	for _, tc := range []struct {
		name string
		now  time.Time
		want bool
	}{
		{"before_start", start.Add(-time.Nanosecond), false},
		{"at_start", start, true},
		{"before_deadline", deadline.Add(-time.Nanosecond), true},
		{"at_deadline", deadline, true},
		{"after_deadline", deadline.Add(time.Nanosecond), false},
		{"reported_expired_project", time.Date(2026, 11, 4, 14, 50, 0, 0, time.Local), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := projectAvailable(row, tc.now); got != tc.want {
				t.Fatalf("available = %v, want %v", got, tc.want)
			}
		})
	}
	for _, invalid := range []string{
		`{"id":"p","status":false,"grant":{"startTime":"2026-09-23","validityPeriod":10}}`,
		`{"id":"p","status":true}`,
		`{"id":"p","status":true,"grant":null}`,
		`{"id":"p","status":true,"grant":{"startTime":"","validityPeriod":10}}`,
		`{"id":"p","status":true,"grant":{"startTime":"2026-02-30","validityPeriod":10}}`,
		`{"id":"p","status":true,"grant":{"startTime":"2026-09-23","validityPeriod":0}}`,
		`{"id":"p","status":true,"grant":{"startTime":"2026-09-23","validityPeriod":-1}}`,
		`{"id":"p","status":true,"grant":{"startTime":"2026-09-23","validityPeriod":"10"}}`,
		`{"id":"p","status":true,"grant":"invalid"}`,
	} {
		if projectAvailable(stdjson.RawMessage(invalid), start) {
			t.Errorf("invalid project accepted: %s", invalid)
		}
	}
}

type availableProjectsTestServer struct {
	core.UnimplementedTableDataServiceServer
	t       *testing.T
	payload []byte
	err     error
}

type availableProjectsAuthServer struct {
	core.UnimplementedAppServiceServer
}

func (s *availableProjectsAuthServer) GetToken(context.Context, *api.TokenRequest) (*api.Response, error) {
	return &api.Response{Status: true, Code: 200, Result: []byte(`{"accessToken":"platform-token","expiresAt":4102444800}`)}, nil
}

func (s *availableProjectsTestServer) Query(_ context.Context, req *core.QueryDataRequest) (*api.Response, error) {
	if req.GetTable() != "project" {
		s.t.Fatalf("unexpected table %s", req.GetTable())
	}
	var query struct {
		Filter struct{ Status bool } `json:"filter"`
	}
	if err := stdjson.Unmarshal(req.GetQuery(), &query); err != nil || !query.Filter.Status {
		s.t.Fatalf("enabled-project filter missing: %s", req.GetQuery())
	}
	if s.err != nil {
		return nil, s.err
	}
	return &api.Response{Status: true, Code: 200, Result: s.payload}, nil
}

func TestQueryProjectAvailable(t *testing.T) {
	server := local_grpc.NewServer()
	impl := &availableProjectsTestServer{t: t}
	core.RegisterAppServiceServer(server, &availableProjectsAuthServer{})
	core.RegisterTableDataServiceServer(server, impl)
	authClient := auth.NewClient(config.Config{Type: config.Project, ProjectId: "base", AK: "ak", SK: "sk"})
	lowLevel, cleanup, err := core.NewLocalClient(config.Config{}, local_grpc.NewClient(server, func() *auth.Client { return authClient }))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	authClient.SetClient(lowLevel)
	client := &Client{CoreClient: lowLevel}
	start := time.Now().AddDate(0, 0, -1).Format(time.DateOnly)
	impl.payload = []byte(fmt.Sprintf(`[
		{"id":"active","name":"Active","status":true,"hasSSO":true,"extra":{"keep":1},"grant":{"startTime":%q,"validityPeriod":10}},
		{"id":"baktest","status":true,"grant":{"startTime":"2000-09-23","validityPeriod":10}},
		{"id":"missing-grant","status":true},
		{"id":"malformed-grant","status":true,"grant":"bad"},
		{"id":"future","status":true,"grant":{"startTime":"9999-01-01","validityPeriod":10}}
	]`, start))
	var rows []map[string]interface{}
	if err := client.QueryProjectAvailable(context.Background(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["id"] != "active" || rows[0]["name"] != "Active" || rows[0]["hasSSO"] != true || rows[0]["extra"] == nil {
		t.Fatalf("unexpected available projects: %#v", rows)
	}
	// Preserve the caller's typed result contract.
	var typed []struct {
		ID    string `json:"id"`
		Grant *struct {
			StartTime string `json:"startTime"`
		} `json:"grant"`
	}
	if err := client.QueryProjectAvailable(context.Background(), &typed); err != nil || len(typed) != 1 || typed[0].Grant.StartTime != start {
		t.Fatalf("typed result: %#v, error: %v", typed, err)
	}
	if err := client.QueryProjectAvailable(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	impl.payload = []byte(`[{"id":"expired","status":true,"grant":{"startTime":"2000-01-01","validityPeriod":1}}]`)
	if err := client.QueryProjectAvailable(context.Background(), &rows); err != nil || rows == nil || len(rows) != 0 {
		t.Fatalf("stale result was not cleared: %#v, error: %v", rows, err)
	}
	impl.err = status.Error(codes.Unavailable, "core unavailable")
	if err := client.QueryProjectAvailable(context.Background(), &rows); err == nil {
		t.Fatal("RPC failure must reach the caller")
	}
}
