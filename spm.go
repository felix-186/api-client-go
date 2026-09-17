package api_client_go

import (
	"context"

	"github.com/felix-186/errors"
	"github.com/felix-186/json"
)

func (c *Client) QueryProject(ctx context.Context, query, result interface{}) error {
	_, err := c.QueryTableData(ctx, "base", "project", query, result)
	return err
}

func (c *Client) QueryProjectAvailable(ctx context.Context, result interface{}) error {
	query := map[string]interface{}{
		"filter": map[string]interface{}{"status": true},
		"project": map[string]interface{}{
			"id":     1,
			"name":   1,
			"grant":  1,
			"status": 1,
		},
	}
	_, err := c.QueryTableData(ctx, "base", "project", query, result)
	return err
}

func (c *Client) RestQueryProject(ctx context.Context, query, result interface{}) error {
	return c.QueryProject(ctx, query, result)
}

func (c *Client) GetProject(ctx context.Context, id string, result interface{}) ([]byte, error) {
	if id == "" {
		return nil, errors.New("id为空")
	}
	query := map[string]interface{}{"filter": map[string]interface{}{"id": id}}
	rows := make([]map[string]interface{}, 0, 1)
	_, err := c.QueryTableData(ctx, "base", "project", query, &rows)
	if err != nil {
		return nil, err
	}
	// QueryTableData only returns a reliable count when withCount is requested;
	// GetProject only needs the row, so use the actual result length here.
	if len(rows) == 0 {
		return nil, errors.New("项目不存在")
	}
	bts, err := json.Marshal(rows[0])
	if err != nil {
		return nil, err
	}
	if result != nil {
		if err := json.Unmarshal(bts, result); err != nil {
			return nil, err
		}
	}
	return bts, err
}

func (c *Client) DeleteProject(ctx context.Context, id string, result interface{}) error {
	if id == "" {
		return errors.New("id为空")
	}
	return c.DeleteTableData(ctx, "base", "project", id, result)
}

func (c *Client) UpdateProject(ctx context.Context, id string, updateData, result interface{}) error {
	if id == "" {
		return errors.New("id为空")
	}
	if updateData == nil {
		return errors.New("更新数据为空")
	}
	return c.UpdateTableData(ctx, "base", "project", id, false, updateData, result)
}

func (c *Client) ReplaceProject(ctx context.Context, id string, updateData, result interface{}) error {
	if id == "" {
		return errors.New("id为空")
	}
	if updateData == nil {
		return errors.New("更新数据为空")
	}
	return c.ReplaceTableData(ctx, "base", "project", id, false, updateData, result)
}

func (c *Client) CreateProject(ctx context.Context, createData, result interface{}) error {
	if createData == nil {
		return errors.New("插入数据为空")
	}
	return c.CreateTableData(ctx, "base", "project", false, createData, result)
}

func (c *Client) QueryPmSetting(ctx context.Context, query, result interface{}) error {
	_, err := c.QueryTableData(ctx, "base", "setting", query, result)
	return err
}
