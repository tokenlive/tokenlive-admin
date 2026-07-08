package schema

import "github.com/tokenlive/tokenlive-admin/pkg/errors"

// PolicyEnabledForm toggles the enabled status of a policy.
type PolicyEnabledForm struct {
	Enabled int `json:"enabled"` // Enable status: 0-disabled, 1-enabled
}

func (p *PolicyEnabledForm) Validate() error {
	if p.Enabled != 0 && p.Enabled != 1 {
		return errors.BadRequest("", "enabled 字段必须为 0 或 1")
	}
	return nil
}
