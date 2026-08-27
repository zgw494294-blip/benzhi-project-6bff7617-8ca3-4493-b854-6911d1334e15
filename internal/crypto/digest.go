package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"fieldlingua/internal/domain"
)

func Digest(p *domain.CorpusProject) (string, error) {
	b, e := p.Canonical()
	if e != nil {
		return "", e
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
