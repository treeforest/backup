package backup

import (
	"github.com/stretchr/testify/require"
	"github.com/treeforest/rollback"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalBackup(t *testing.T) {
	// 创建临时测试目录
	testDir, err := os.MkdirTemp("", "backup-test")
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	// 测试用例
	t.Run("正常备份与回滚", func(t *testing.T) {
		rb := rollback.New()
		srcPath := filepath.Join(testDir, "data.txt")

		// 创建测试文件
		require.NoError(t, os.WriteFile(srcPath, []byte("test data"), 0644))

		// 执行备份
		backupPath, ok, err := LocalBackup(rb, srcPath)
		require.NoError(t, err)
		require.True(t, ok)
		require.FileExists(t, backupPath)

		// 修改原始文件
		require.NoError(t, os.WriteFile(srcPath, []byte("modified"), 0644))

		// 执行回滚
		rb.Rollback(nil)

		// 验证原始文件恢复
		data, err := os.ReadFile(srcPath)
		require.NoError(t, err)
		require.Equal(t, "test data", string(data))
		require.NoFileExists(t, backupPath)
	})

	t.Run("源文件不存在时跳过备份", func(t *testing.T) {
		rb := rollback.New()
		nonExistPath := filepath.Join(testDir, "nonexist.txt")

		// 跳过不存在的文件
		_, ok, err := LocalBackup(rb, nonExistPath, BackupOption{SkipIfNotExist: true})
		require.NoError(t, err)
		require.False(t, ok)

		// 不跳过时报错
		_, _, err = LocalBackup(rb, nonExistPath, BackupOption{SkipIfNotExist: false})
		require.Error(t, err)
	})

	t.Run("备份文件自动清理", func(t *testing.T) {
		rb := rollback.New()
		srcPath := filepath.Join(testDir, "cleanup.txt")
		require.NoError(t, os.WriteFile(srcPath, []byte("data"), 0644))

		// 执行备份但不回滚
		backupPath, _, err := LocalBackup(rb, srcPath)
		require.NoError(t, err)

		// 手动触发最终清理
		rb.ExecDeferFunc()

		require.NoFileExists(t, backupPath)
	})
}

func TestLocalBackupMethods(t *testing.T) {
	b := &LocalBackupImpl{}
	tempDir, err := os.MkdirTemp("", "methods-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("PathExists检查", func(t *testing.T) {
		require.True(t, b.PathExists(tempDir))
		require.False(t, b.PathExists(filepath.Join(tempDir, "nonexist")))
	})

	t.Run("文件拷贝操作", func(t *testing.T) {
		src := filepath.Join(tempDir, "original.txt")
		dst := filepath.Join(tempDir, "copy.txt")

		require.NoError(t, os.WriteFile(src, []byte("content"), 0644))
		require.NoError(t, b.Copy(src, dst))
		require.FileExists(t, dst)

		// 验证内容一致性
		srcData, _ := os.ReadFile(src)
		dstData, _ := os.ReadFile(dst)
		require.Equal(t, srcData, dstData)
	})

	t.Run("目录递归删除", func(t *testing.T) {
		dir := filepath.Join(tempDir, "nested")
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("test"), 0644))

		require.NoError(t, b.RemoveAll(dir))
		require.NoFileExists(t, dir)
	})
}
