package errors

import (
	"context"
	"encoding/json"
	"strconv"

	base "github.com/felix-186/errors"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const errorDomain = "gts.ResponseError"

// ToGRPC preserves structured application errors for both generic and typed
// protobuf RPCs. It never converts unknown/internal errors into client errors.
func ToGRPC(err error) error {
	if err == nil {
		return nil
	}
	var response *base.ResponseError
	if !base.As(err, &response) {
		if _, ok := status.FromError(err); ok {
			return err
		}
		return status.FromContextError(err).Err()
	}
	code := codes.Unknown
	switch response.StatusCode {
	case 400, 422:
		code = codes.InvalidArgument
	case 401:
		code = codes.Unauthenticated
	case 403:
		code = codes.PermissionDenied
	case 404:
		code = codes.NotFound
	case 408, 504:
		code = codes.DeadlineExceeded
	case 409:
		code = codes.AlreadyExists
	case 429:
		code = codes.ResourceExhausted
	case 503:
		code = codes.Unavailable
	default:
		if response.StatusCode >= 500 {
			code = codes.Internal
		} else if response.StatusCode >= 400 {
			code = codes.FailedPrecondition
		}
	}
	metadata := map[string]string{
		"code": strconv.Itoa(response.Code), "status": strconv.Itoa(response.StatusCode),
		"message": response.Message, "field": response.Field,
	}
	if response.Data != nil {
		if data, e := json.Marshal(response.Data); e == nil {
			metadata["data"] = string(data)
		}
	}
	st := status.New(code, err.Error())
	withDetails, e := st.WithDetails(&errdetails.ErrorInfo{Domain: errorDomain, Reason: "APPLICATION_ERROR", Metadata: metadata})
	if e != nil {
		return st.Err()
	}
	return withDetails.Err()
}

// FromGRPC decodes only our own tagged details; plain gRPC status errors retain
// their original meaning and status. The cause keeps the transport status.
func FromGRPC(err error) error {
	if err == nil {
		return nil
	}
	var existing *base.ResponseError
	if base.As(err, &existing) {
		return err
	}
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	for _, detail := range st.Details() {
		info, ok := detail.(*errdetails.ErrorInfo)
		if !ok || info.Domain != errorDomain || info.Reason != "APPLICATION_ERROR" {
			continue
		}
		code, e1 := strconv.Atoi(info.Metadata["code"])
		httpStatus, e2 := strconv.Atoi(info.Metadata["status"])
		if e1 != nil || e2 != nil {
			return err
		}
		response := &base.ResponseError{Code: code, StatusCode: httpStatus, Message: info.Metadata["message"], Field: info.Metadata["field"], ERR: err}
		if data := info.Metadata["data"]; data != "" {
			_ = json.Unmarshal([]byte(data), &response.Data)
		}
		return response
	}
	return err
}

// UnaryClientErrorDecoder applies to every service created through CreateConn.
func UnaryClientErrorDecoder(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	return FromGRPC(invoke(ctx, method, req, reply, cc, opts...))
}
