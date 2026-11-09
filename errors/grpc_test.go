package errors

import (
	"context"
	"fmt"
	"testing"

	base "github.com/felix-186/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGRPCStructuredErrorRoundTrip(t *testing.T) {
	for _, httpStatus := range []int{400, 401, 403, 404, 408, 409, 422, 429, 500, 503, 504} {
		t.Run(fmt.Sprint(httpStatus), func(t *testing.T) {
			original := &base.ResponseError{StatusCode: httpStatus, Code: 120010003, Message: "name occupied", Field: "name", Data: map[string]interface{}{"name": "camera"}}
			wire := ToGRPC(fmt.Errorf("wrapped: %w", original))
			var decoded *base.ResponseError
			if !base.As(FromGRPC(wire), &decoded) || decoded.StatusCode != httpStatus || decoded.Code != original.Code || decoded.Field != "name" || decoded.Message != original.Message {
				t.Fatalf("lost business fields: %v", FromGRPC(wire))
			}
			if decoded.Data.(map[string]interface{})["name"] != "camera" {
				t.Fatal("lost template data")
			}
			if original.ERR != nil || original.Message != "name occupied" {
				t.Fatal("mutated source")
			}
		})
	}
}

func TestGRPCNativeAndUnknownErrors(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want codes.Code
	}{
		{status.Error(codes.Unavailable, "offline"), codes.Unavailable},
		{status.Error(codes.PermissionDenied, "denied"), codes.PermissionDenied},
		{context.Canceled, codes.Canceled}, {context.DeadlineExceeded, codes.DeadlineExceeded},
		{fmt.Errorf("database failed"), codes.Unknown},
	} {
		wire := ToGRPC(tc.err)
		if got := status.Code(wire); got != tc.want {
			t.Fatalf("got %v, want %v", got, tc.want)
		}
		if FromGRPC(wire) != wire {
			t.Fatal("plain status incorrectly decoded")
		}
	}
	if ToGRPC(nil) != nil || FromGRPC(nil) != nil {
		t.Fatal("nil changed")
	}
}
