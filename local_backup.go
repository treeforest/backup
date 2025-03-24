package backup

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/treeforest/rollback"
	"golang.org/x/sync/errgroup"
)

// LocalBackup 本地文件系统备份
func LocalBackup(rb rollback.Rollbacker, path string, opts ...BackupOption) (string, bool, error) {
	return Backup(rb, &LocalBackupImpl{}, path, opts...)
}

type LocalBackupImpl struct {
	Concurrency int
}

func (b *LocalBackupImpl) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (b *LocalBackupImpl) Rename(src, dst string) error {
	return os.Rename(src, dst)
}

func (b *LocalBackupImpl) Copy(src, dst string) (err error) {
	// 获取源文件信息
	var info os.FileInfo
	info, err = os.Stat(src)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = os.RemoveAll(dst)
		}
	}()

	if info.IsDir() {
		return b.copyDirectory(src, dst)
	}
	return b.copyFile(src, dst)
}

// 递归拷贝目录
func (b *LocalBackupImpl) copyDirectory(src, dst string) error {
	// 创建带并发控制的 errgroup
	g, ctx := errgroup.WithContext(context.Background())
	if b.Concurrency > 0 {
		g.SetLimit(b.Concurrency) // 限制最大并发数
	}

	// 获取源目录信息
	srcInfo, err := os.Stat(src)
	if err != nil {
		return errors.Wrap(err, "获取源目录信息失败")
	}

	// 删除旧目录
	if b.PathExists(dst) {
		if err = b.RemoveAll(dst); err != nil {
			return errors.Wrap(err, "删除旧目录失败")
		}
	}

	// 创建目标目录（保持权限）
	if err = os.Mkdir(dst, srcInfo.Mode().Perm()); err != nil {
		return errors.Wrap(err, "创建目标目录失败")
	}

	// 遍历源目录
	entries, err := os.ReadDir(src)
	if err != nil {
		return errors.Wrap(err, "读取目录失败")
	}

	// 处理目录条目
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// 递归处理子目录（同步执行）
			if err = b.copyDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// 提交文件拷贝任务到并发组
			g.Go(func() error {
				select {
				case <-ctx.Done(): // 快速失败
					return ctx.Err()
				default:
					return b.copyFile(srcPath, dstPath)
				}
			})
		}
	}

	// 等待所有并发任务完成
	if err = g.Wait(); err != nil {
		return errors.Wrap(err, "并发拷贝失败")
	}

	// 保留目录元数据
	return b.preserveMetadata(srcInfo, dst)
}

// 拷贝单个文件
func (b *LocalBackupImpl) copyFile(src, dst string) error {
	// 删除旧文件
	if b.PathExists(dst) {
		if err := b.RemoveAll(dst); err != nil {
			return errors.Wrap(err, "删除旧文件失败")
		}
	}

	// 获取源文件信息
	srcInfo, err := os.Stat(src)
	if err != nil {
		return errors.Wrap(err, "获取源文件状态失败")
	}

	// 保留原始权限模式
	perm := srcInfo.Mode().Perm()

	// 打开源文件
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// 创建目标文件（保持权限）
	dstFile, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return errors.Wrap(err, "创建目标文件失败")
	}
	defer dstFile.Close()

	// 拷贝内容
	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return errors.Wrap(err, "文件内容拷贝失败")
	}

	// 同步元数据（权限和时间戳）
	if err = b.preserveMetadata(srcInfo, dst); err != nil {
		return errors.Wrap(err, "保留元数据失败")
	}

	// 同步到磁盘
	return dstFile.Sync()
}

// 保留文件元数据（权限、时间戳等）
func (b *LocalBackupImpl) preserveMetadata(info os.FileInfo, path string) error {
	// 保留修改时间
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		return err
	}

	// 保留权限模式
	return os.Chmod(path, info.Mode().Perm())
}

func (b *LocalBackupImpl) PathExists(path string) bool {
	if path == "" {
		return false
	}

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
