# Backup

文件系统备份工具，提供原子操作保障和高效并发处理，适用于需要可靠回滚机制的关键业务场景。

## 特性亮点 ✨

- **双模式备份策略**  
  ✅ **移动模式**：高性能文件重命名  
  ✅ **拷贝模式**：安全保留源文件

- ​**智能并发引擎**
  ```go
  backuper := &LocalBackupImpl{Concurrency: 16} // 按需调整并发度
  ```

## 快速接入 🚀

### 安装

```bash
go get github.com/treeforest/backup
```

### 基础用法

```go
import "github.com/treeforest/backup"

// 执行移动式备份（不保留源文件）
backupPath, ok, err := backup.LocalBackup(nil, "/data/bak", backup.BackupOption{KeepSource: false})
```

## 授权许可

Apache 许可证 2.0 版本 - 详见 [LICENSE](https://www.apache.org/licenses/LICENSE-2.0.txt)