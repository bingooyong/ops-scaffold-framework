package version

// Version 版本号，通过构建时注入
var Version = "dev"

// BuildTime 构建时间，通过构建时注入
var BuildTime = "unknown"

// GitCommit Git 提交哈希，通过构建时注入
var GitCommit = "unknown"

// GetVersion 获取版本信息
func GetVersion() string {
	return Version
}

// GetBuildTime 获取构建时间
func GetBuildTime() string {
	return BuildTime
}

// GetGitCommit 获取 Git 提交哈希
func GetGitCommit() string {
	return GitCommit
}

// GetFullVersion 获取完整版本信息
func GetFullVersion() string {
	return Version + " (" + GitCommit + ")" + " built at " + BuildTime
}
