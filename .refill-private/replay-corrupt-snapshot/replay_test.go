package replay_corrupt_snapshot

import (
	"fieldlingua/internal/domain"
	"fieldlingua/internal/persistence"
	"os"
	"path/filepath"
	"testing"
)

func TestReplayCorruptSnapshotPreservesProjection(t *testing.T) {
	dir := t.TempDir()
	store := persistence.New(dir)
	project, err := domain.NewProject("p-replay", "重放测试项目", "变体", "owner", []string{"地点"}, domain.EthicsApproved)
	if err != nil {
		t.Fatal(err)
	}
	store.PutProject(project)
	if err = os.WriteFile(filepath.Join(dir, "snapshot.json"), []byte("{损坏快照"), 0644); err != nil {
		t.Fatal(err)
	}

	if err = store.Replay(); err == nil {
		t.Errorf("损坏快照重放必须返回可追踪错误")
	}
	if _, err = store.GetProject(project.ProjectID); err != nil {
		t.Fatalf("重放解析失败时不应清空已有项目投影: %v", err)
	}
}
