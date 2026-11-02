package api_client_go

import (
	"context"
	"fmt"
	"strings"

	"github.com/felix-186/api-client-go/apicontext"
	"github.com/felix-186/api-client-go/auth"
	"github.com/felix-186/api-client-go/config"
	"github.com/felix-186/api-client-go/conn"
	pb "github.com/felix-186/api-client-go/eap"
	"github.com/go-kratos/kratos/v2/registry"
	"google.golang.org/grpc"
)

const EAPBackupChunkSize = 256 * 1024

// NewBackupClient opens one EAP backup session. KESI reuses its in-process
// client. Service-discovered deployments pin the entire transfer to one node.
func (c *Client) NewBackupClient(ctx context.Context) (pb.BackupServiceClient, func(), error) {
	if c.EAPClient == nil {
		return nil, nil, fmt.Errorf("EAP gRPC client unavailable")
	}
	if c.RegistryClient == nil {
		client, err := c.EAPClient.GetBackupServiceClient()
		return client, func() {}, err
	}
	instances, err := c.RegistryClient.GetServiceInstances(ctx, "eap", true)
	if err != nil {
		return nil, nil, err
	}
	cfg := c.Config
	wanted := cfg.Metadata
	if meta := cfg.Services["eap"].Metadata; len(meta) > 0 {
		wanted = meta
	}
	address := eapBackupEndpoint(instances, wanted)
	if address == "" {
		return nil, nil, fmt.Errorf("no matching EAP gRPC backup endpoint")
	}
	services := make(map[string]config.Service, len(cfg.Services)+1)
	for name, service := range cfg.Services {
		services[name] = service
	}
	service := services["eap"]
	service.GrpcAddr = address
	services["eap"] = service
	cfg.Services = services
	if cfg.Timeout < 3600 {
		cfg.Timeout = 3600
	}
	if c.AuthClient == nil {
		return nil, nil, fmt.Errorf("EAP authentication client unavailable")
	}
	credential := auth.NewCustomCredential(func() *auth.Client { return c.AuthClient })
	cc, err := conn.CreateConn("eap", cfg, nil, grpc.WithPerRPCCredentials(credential))
	if err != nil {
		return nil, nil, err
	}
	return pb.NewBackupServiceClient(cc), func() { _ = cc.Close() }, nil
}

func eapBackupEndpoint(instances []*registry.ServiceInstance, wanted map[string]string) string {
	for _, instance := range instances {
		matches := true
		for key, value := range wanted {
			if instance.Metadata[key] != value {
				matches = false
				break
			}
		}
		if !matches {
			continue
		}
		for _, endpoint := range instance.Endpoints {
			if strings.HasPrefix(endpoint, "grpc://") {
				return strings.TrimPrefix(endpoint, "grpc://")
			}
		}
	}
	return ""
}

type EAPTaskAttachment struct {
	URL         string
	Filename    string
	ContentType string
}

type SendEAPTaskMessageRequest struct {
	TaskID       string
	Content      string
	SystemPrompt string
	Attachments  []EAPTaskAttachment
}

type SendEAPTaskMessageResult struct {
	TaskID    string
	RunID     string
	MessageID string
}

func (c *Client) SendEAPTaskMessage(ctx context.Context, projectID string, request *SendEAPTaskMessageRequest) (*SendEAPTaskMessageResult, error) {
	client, err := c.EAPClient.GetTaskServiceClient()
	if err != nil {
		return nil, err
	}
	attachments := make([]*pb.Attachment, 0, len(request.Attachments))
	for _, attachment := range request.Attachments {
		attachments = append(attachments, &pb.Attachment{Url: attachment.URL, Filename: attachment.Filename, ContentType: attachment.ContentType})
	}
	if projectID == "" {
		projectID = config.XRequestProjectDefault
	}
	response, err := client.SendMessage(apicontext.GetGrpcContext(ctx, map[string]string{config.XRequestProject: projectID}), &pb.SendTaskMessageRequest{TaskId: request.TaskID, Content: request.Content, SystemPrompt: request.SystemPrompt, Attachments: attachments})
	if err != nil {
		return nil, err
	}
	return &SendEAPTaskMessageResult{TaskID: response.TaskId, RunID: response.RunId, MessageID: response.MessageId}, nil
}
