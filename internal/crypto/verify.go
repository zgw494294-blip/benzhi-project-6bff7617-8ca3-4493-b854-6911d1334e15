package crypto

import (
	"encoding/hex"
	"fmt"
	"strings"
)

func ValidFormat(c Credential) bool {
	return ValidateFormat(c) == nil
}

func ValidateFormat(c Credential) error {
	if strings.TrimSpace(c.CredentialID) == "" {
		return fmt.Errorf("credentialID 格式错误")
	}
	if strings.TrimSpace(c.ProjectID) == "" {
		return fmt.Errorf("项目 ID 格式错误")
	}
	if len(c.RevisionIDs) == 0 {
		return fmt.Errorf("修订 ID 列表不能为空")
	}
	for _, id := range c.RevisionIDs {
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("修订 ID 格式错误")
		}
	}
	if len(c.ContentDigest) != 64 {
		return fmt.Errorf("contentDigest 格式错误")
	}
	if _, err := hex.DecodeString(c.ContentDigest); err != nil {
		return fmt.Errorf("contentDigest 格式错误")
	}
	if len(strings.TrimSpace(c.Signature)) != 64 {
		return fmt.Errorf("签名格式错误")
	}
	if _, err := hex.DecodeString(c.Signature); err != nil {
		return fmt.Errorf("签名格式错误")
	}
	return nil
}
