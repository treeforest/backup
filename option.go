package backup

// BackupOption 备份配置选项
type BackupOption struct {
	// Suffix 备份文件后缀，默认使用时间戳格式
	Suffix string

	// SkipIfNotExist 源文件不存在时跳过备份
	SkipIfNotExist bool

	// KeepSource 是否保留源文件；若为true，则备份完成后，源文件会被保留；否则，则不保留源文件。
	KeepSource bool
}
