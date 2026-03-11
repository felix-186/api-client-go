package api_client_go

import (
	"context"
	"github.com/zhgqiang/api-client-go/live"

	"github.com/zhgqiang/api-client-go/api"
	"github.com/zhgqiang/api-client-go/apicontext"
	"github.com/zhgqiang/api-client-go/config"
	"github.com/zhgqiang/errors"
	"github.com/zhgqiang/json"
)

func (c *Client) RtspPull(ctx context.Context, projectId string, createData interface{}) (string, error) {
	if projectId == "" {
		projectId = config.XRequestProjectDefault
	}
	if createData == nil {
		return "", errors.New("插入数据为空")
	}
	cli, err := c.LiveClient.GetLiveServiceClient()
	if err != nil {
		return "", err
	}
	bts, err := json.Marshal(createData)
	if err != nil {
		return "", errors.Wrap(err, "序列化插入数据错误")
	}
	res, err := cli.Create(
		apicontext.GetGrpcContext(ctx, map[string]string{config.XRequestProject: projectId}),
		&api.CreateRequest{Data: bts})
	result, err := parseRes(err, res, nil)
	if err != nil {
		return "", err
	}
	if result != nil || string(result) == "" {
		return string(result), nil
	} else {
		return "", errors.New("result中path为空")
	}
}

func (c *Client) LivePull2Ws(ctx context.Context, projectId string, createData interface{}) (string, error) {
	return c.RtspPull(ctx, projectId, createData)
}

func (c *Client) LiveStreamsInfo(ctx context.Context, projectId string, result interface{}) error {
	if projectId == "" {
		projectId = config.XRequestProjectDefault
	}
	cli, err := c.LiveClient.GetLiveServiceClient()
	if err != nil {
		return err
	}
	res, err := cli.QueryStreamsInfo(
		apicontext.GetGrpcContext(ctx, map[string]string{config.XRequestProject: projectId}),
		&api.QueryRequest{Query: nil})
	if _, err = parseRes(err, res, result); err != nil {
		return err
	}
	return nil
}

func (c *Client) LiveStreamStopByStreamPath(ctx context.Context, projectId, streamPath string) error {
	if projectId == "" {
		projectId = config.XRequestProjectDefault
	}
	cli, err := c.LiveClient.GetLiveServiceClient()
	if err != nil {
		return err
	}
	res, err := cli.StopStreamByStreamPath(
		apicontext.GetGrpcContext(ctx, map[string]string{config.XRequestProject: projectId}),
		&live.LiveStopStreamByStreamPathRequest{StreamPath: streamPath})
	if _, err = parseRes(err, res, nil); err != nil {
		return err
	}
	return nil
}
