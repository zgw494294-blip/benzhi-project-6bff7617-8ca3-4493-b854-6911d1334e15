package persistence_save_failure

import (
	"errors"
	"fieldlingua/internal/application"
	"fieldlingua/internal/domain"
	"fieldlingua/internal/persistence"
	"os"
	"testing"
)

func TestPersistenceFailureDoesNotCommit(t *testing.T) {
	dir := t.TempDir()
	storePath := dir + "/store-file"
	if err := os.WriteFile(storePath, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	service := application.New(persistence.New(storePath))
	_, err := service.Create(application.CreateProjectInput{
		ProjectID:       "p-save-failure",
		Title:           "持久化失败项目",
		LanguageVariant: "方言",
		OwnerID:         "owner",
		CollectionSites: []string{"地点"},
		EthicsStatus:    domain.EthicsApproved,
	})
	if err == nil {
		t.Fatal("存储目录不可写时创建必须失败")
	}
	if _, getErr := service.Store.GetProject("p-save-failure"); !errors.Is(getErr, domain.ErrNotFound) {
		t.Fatalf("保存失败不应提交内存状态，得到错误=%v", getErr)
	}
}
