package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound        = errors.New("未找到资源")
	ErrVersionConflict = errors.New("版本冲突")
	ErrInvalid         = errors.New("领域校验失败")
	ErrForbidden       = errors.New("当前角色无权执行")
	ErrConflict        = errors.New("资源冲突")
)

type ValidationError struct {
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func (e *ValidationError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return ErrInvalid.Error()
}
func (e *ValidationError) Unwrap() error { return ErrInvalid }

func InvalidField(field, message string) error {
	return &ValidationError{Message: "请求字段校验失败", Fields: map[string]string{field: message}}
}

type VersionConflictError struct {
	ExpectedVersion int    `json:"expectedVersion"`
	CurrentVersion  int    `json:"currentVersion"`
	ProjectID       string `json:"projectID"`
}

func (e *VersionConflictError) Error() string {
	return fmt.Sprintf("版本冲突：期望版本 %d，当前版本 %d，请重新加载项目", e.ExpectedVersion, e.CurrentVersion)
}
func (e *VersionConflictError) Unwrap() error { return ErrVersionConflict }
