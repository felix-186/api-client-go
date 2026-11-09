package local_grpc

import (
	"context"
	"fmt"
	"testing"

	"github.com/felix-186/api-client-go/api"
	pb "github.com/felix-186/api-client-go/core"
	"github.com/felix-186/errors"
)

type errorTableServer struct {
	pb.UnimplementedTableDataServiceServer
}

func (errorTableServer) Get(context.Context, *pb.GetOrDeleteDataRequest) (*api.Response, error) {
	return nil, fmt.Errorf("wrapped: %w", errors.New400Response(120010003, "name conflict"))
}

func TestLocalInvokeRetainsBusinessError(t *testing.T) {
	s := NewServer()
	pb.RegisterTableDataServiceServer(s, errorTableServer{})
	cli := pb.NewTableDataServiceClient(NewClient(s, nil))
	_, err := cli.Get(context.Background(), &pb.GetOrDeleteDataRequest{})
	var business *errors.ResponseError
	if !errors.As(err, &business) || business.Code != 120010003 || business.StatusCode != 400 {
		t.Fatalf("business error lost: %v", err)
	}
}
