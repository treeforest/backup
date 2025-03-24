package backup

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/treeforest/rollback"
)

// Backuper 定义备份操作接口
type Backuper interface {
	// PathExists 检查路径是否存在（文件或目录）
	PathExists(path string) bool

	// Copy 拷贝文件或目录
	Copy(src, dst string) error

	// Rename 重命名文件或目录
	Rename(src, dst string) error

	// RemoveAll 递归删除路径（文件或目录）
	RemoveAll(path string) error
}

// Backup 执行备份操作
// 返回值: (备份路径, 是否执行了备份, 错误)
func Backup(
	rb rollback.Rollbacker,
	b Backuper,
	srcPath string,
	opts ...BackupOption,
) (string, bool, error) {
	// 处理可选参数
	opt := BackupOption{
		Suffix:         time.Now().Format(".backup.20060102150405"),
		SkipIfNotExist: true,
	}
	if len(opts) > 0 {
		opt = opts[0]
	}

	// 检查源路径是否存在
	if !b.PathExists(srcPath) {
		if opt.SkipIfNotExist {
			return "", false, nil
		}
		return "", false, errors.Errorf("源路径不存在: %s", srcPath)
	}

	// 生成备份路径
	backupPath := generateBackupPath(srcPath, opt.Suffix)

	// 清理可能存在的旧备份
	if b.PathExists(backupPath) {
		if err := b.RemoveAll(backupPath); err != nil {
			return "", false, errors.Wrapf(err, "清理旧备份失败: %s", backupPath)
		}
	}

	// 执行备份操作
	if err := performBackup(b, srcPath, backupPath, opt.KeepSource); err != nil {
		return "", false, err
	}

	// 注册回滚操作，恢复源文件
	if rb != nil {
		registerRollback(rb, b, srcPath, backupPath, opt.KeepSource)
	}

	return backupPath, true, nil
}

// 生成带唯一后缀的备份路径
func generateBackupPath(srcPath, suffix string) string {
	base := filepath.Base(srcPath)
	dir := filepath.Dir(srcPath)
	return filepath.Join(dir, fmt.Sprintf("%s%s", base, suffix))
}

// 执行实际的备份操作
func performBackup(b Backuper, src, dst string, keepSource bool) error {
	if keepSource {
		if err := b.Rename(src, dst); err != nil {
			return errors.Wrapf(err, "文件备份失败: %s -> %s", src, dst)
		}
	} else {
		if err := b.Copy(src, dst); err != nil {
			return errors.Wrapf(err, "文件备份失败: %s -> %s", src, dst)
		}
	}
	return nil
}

// 注册回滚操作
func registerRollback(rb rollback.Rollbacker, b Backuper, srcPath, backupPath string, keepSource bool) {
	if !keepSource {
		// 恢复操作
		rb.PushFront(func() error {
			// 清理当前源路径
			if b.PathExists(srcPath) {
				if err := b.RemoveAll(srcPath); err != nil {
					return errors.Wrapf(err, "清理当前路径失败: %s", srcPath)
				}
			}

			// 恢复备份
			if err := b.Rename(backupPath, srcPath); err != nil {
				return errors.Wrapf(err, "恢复备份失败: %s -> %s", backupPath, srcPath)
			}
			return nil
		})
	}

	// 最终删除备份文件
	rb.PushDefer(func() error {
		if !b.PathExists(backupPath) {
			return nil
		}
		if err := b.RemoveAll(backupPath); err != nil {
			return errors.Wrapf(err, "清理备份文件失败: %s", backupPath)
		}
		return nil
	})
}
