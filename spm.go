package api_client_go

import (
	"context"
	stdjson "encoding/json"
	"time"

	"github.com/felix-186/errors"
	"github.com/felix-186/json"
)

func (c *Client) QueryProject(ctx context.Context, query, result interface{}) error {
	_, err := c.QueryTableData(ctx, "base", "project", query, result)
	return err
}

// QueryProjectAvailable returns enabled projects whose grants are currently valid.
func (c *Client) QueryProjectAvailable(ctx context.Context, result interface{}) error {
	query := map[string]interface{}{
		"filter": map[string]interface{}{"status": true},
		"project": map[string]interface{}{
			"id":     1,
			"name":   1,
			"grant":  1,
			"status": 1,
			"hasSSO": 1,
		},
	}
	// Keep raw rows so callers retain all projected fields and may use their
	// own project types without introducing a dependency on service/entitys.
	var rows []stdjson.RawMessage
	if _, err := c.QueryTableData(ctx, "base", "project", query, &rows); err != nil {
		return err
	}
	available := make([]stdjson.RawMessage, 0, len(rows))
	now := time.Now()
	for _, row := range rows {
		if projectAvailable(row, now) {
			available = append(available, row)
		}
	}
	if result == nil {
		return nil
	}
	bts, err := json.Marshal(available)
	if err != nil {
		return err
	}
	return json.Unmarshal(bts, result)
}

func projectAvailable(row stdjson.RawMessage, now time.Time) bool {
	var project struct {
		ID     string `json:"id"`
		Status bool   `json:"status"`
		Grant  *struct {
			StartTime      string `json:"startTime"`
			ValidityPeriod int    `json:"validityPeriod"`
		} `json:"grant"`
	}
	if err := json.Unmarshal(row, &project); err != nil {
		return false
	}
	if project.ID == "" || !project.Status || project.Grant == nil || project.Grant.ValidityPeriod <= 0 {
		return false
	}
	// Match Core's license validation: local midnight and calendar days.
	start, err := time.ParseInLocation(time.DateOnly, project.Grant.StartTime, time.Local)
	if err != nil || now.Before(start) {
		return false
	}
	deadline := start.AddDate(0, 0, project.Grant.ValidityPeriod)
	return !deadline.Before(start) && !now.After(deadline)
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
