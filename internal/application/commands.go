package application

import (
	"fieldlingua/internal/crypto"
	"fieldlingua/internal/quality"
)

type Result struct {
	Project          any                    `json:"project,omitempty"`
	Issues           any                    `json:"issues,omitempty"`
	IssueSummary     []quality.IssueSummary `json:"issueSummary,omitempty"`
	Blocking         *bool                  `json:"blocking,omitempty"`
	Credential       *crypto.Credential     `json:"credential,omitempty"`
	Verification     *VerificationResult    `json:"verification,omitempty"`
	Message          string                 `json:"message,omitempty"`
	IdempotentReplay bool                   `json:"idempotentReplay,omitempty"`
}
