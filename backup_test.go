package backup

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/treeforest/rollback"
)

func TestBackupPathGeneration(t *testing.T) {
	testCases := []struct {
		input    string
		suffix   string
		expected string
	}{
		{"/data/file.txt", ".bak", "/data/file.txt.bak"},
		{"/var/log/app.log", ".20230101", "/var/log/app.log.20230101"},
		{"/tmp/config", "_backup", "/tmp/config_backup"},
	}

	for _, tc := range testCases {
		actual := generateBackupPath(tc.input, tc.suffix)
		require.Equal(t, tc.expected, actual)
	}
}

func TestBackupErrorHandling(t *testing.T) {
	t.Run("备份恢复失败处理", func(t *testing.T) {
		// 使用错误注入的Backuper
		mock := &errorInjectBackup{
			failAt: "Copy", // 模拟恢复时的拷贝失败
		}

		rb := rollback.New()
		_, _, err := Backup(rb, mock, "/fake/path")
		require.Error(t, err)
	})
}

// 错误注入测试结构体
type errorInjectBackup struct {
	failAt string
}

func (b *errorInjectBackup) PathExists(path string) bool {
	return path == "/fake/path"
}

func (b *errorInjectBackup) Rename(src, dst string) error {
	return nil
}

func (b *errorInjectBackup) Copy(src, dst string) error {
	if b.failAt == "Copy" {
		return fmt.Errorf("模拟拷贝失败")
	}
	return nil
}

func (b *errorInjectBackup) RemoveAll(path string) error {
	if b.failAt == "RemoveAll" {
		return fmt.Errorf("模拟删除失败")
	}
	return nil
}
