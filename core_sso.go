package api_client_go

import (
	"context"
	"strings"

	"github.com/felix-186/api-client-go/apicontext"
	"github.com/felix-186/api-client-go/config"
	"github.com/felix-186/api-client-go/core"
	"github.com/felix-186/errors"
)

// ExchangeSSO 使用客户端现有认证，换取指定项目用户的完整登录结果。
// 外部身份验证由调用方完成；result 的结构与 HTTP 登录响应数据一致。
func (c *Client) ExchangeSSO(ctx context.Context, projectId, username string, result interface{}) error {
	if projectId == "" {
		projectId = config.XRequestProjectDefault
	}
	if strings.TrimSpace(username) == "" {
		return errors.New("username is empty")
	}
	cli, err := c.CoreClient.GetAppServiceClient()
	if err != nil {
		return err
	}
	res, err := cli.ExchangeSSO(apicontext.GetGrpcContext(ctx, map[string]string{
		config.XRequestProject: projectId,
	}), &core.SSOExchangeRequest{Username: username})
	_, err = parseRes(err, res, result)
	return err
}
