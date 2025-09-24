package function

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// CheckAndInstallRipgrep 检测系统中是否有 ripgrep (rg)，如果没有则尝试安装
func CheckAndInstallRipgrep() (string, bool) {
	// 首先检查 rg 命令是否可用
	if isRipgrepAvailable() {
		return "rg", true
	}

	// 如果没有，尝试安装
	fmt.Println("未检测到 ripgrep (rg)，正在尝试安装...")
	
	success := false
	switch runtime.GOOS {
	case "windows":
		success = installRipgrepWindows()
	case "linux":
		success = installRipgrepLinux()
	case "darwin":
		success = installRipgrepMacOS()
	default:
		fmt.Printf("不支持的操作系统: %s\n", runtime.GOOS)
		return "grep", false
	}

	if success && isRipgrepAvailable() {
		fmt.Println("ripgrep 安装成功!")
		return "rg", true
	}

	fmt.Println("ripgrep 安装失败，将使用传统的 grep")
	return "grep", false
}

// isRipgrepAvailable 检查 ripgrep 是否可用
func isRipgrepAvailable() bool {
	cmd := exec.Command("rg", "--version")
	err := cmd.Run()
	return err == nil
}

// installRipgrepWindows 在 Windows 上安装 ripgrep
func installRipgrepWindows() bool {
	// 尝试使用 chocolatey
	if isCommandAvailable("choco") {
		cmd := exec.Command("choco", "install", "ripgrep", "-y")
		if cmd.Run() == nil {
			return true
		}
	}

	// 尝试使用 scoop
	if isCommandAvailable("scoop") {
		cmd := exec.Command("scoop", "install", "ripgrep")
		if cmd.Run() == nil {
			return true
		}
	}

	// 尝试使用 winget
	if isCommandAvailable("winget") {
		cmd := exec.Command("winget", "install", "BurntSushi.ripgrep.MSVC")
		if cmd.Run() == nil {
			return true
		}
	}

	fmt.Println("Windows: 未找到包管理器 (chocolatey/scoop/winget)，无法自动安装 ripgrep")
	fmt.Println("请手动安装: https://github.com/BurntSushi/ripgrep/releases")
	return false
}

// installRipgrepLinux 在 Linux 上安装 ripgrep
func installRipgrepLinux() bool {
	// 尝试使用 apt (Ubuntu/Debian)
	if isCommandAvailable("apt") {
		cmd := exec.Command("apt", "update")
		cmd.Run()
		cmd = exec.Command("apt", "install", "-y", "ripgrep")
		if cmd.Run() == nil {
			return true
		}
	}

	// 尝试使用 yum (CentOS/RHEL)
	if isCommandAvailable("yum") {
		cmd := exec.Command("yum", "install", "-y", "ripgrep")
		if cmd.Run() == nil {
			return true
		}
	}

	// 尝试使用 dnf (Fedora)
	if isCommandAvailable("dnf") {
		cmd := exec.Command("dnf", "install", "-y", "ripgrep")
		if cmd.Run() == nil {
			return true
		}
	}

	// 尝试使用 pacman (Arch)
	if isCommandAvailable("pacman") {
		cmd := exec.Command("pacman", "-S", "--noconfirm", "ripgrep")
		if cmd.Run() == nil {
			return true
		}
	}

	fmt.Println("Linux: 无法通过包管理器安装 ripgrep")
	fmt.Println("请手动安装: https://github.com/BurntSushi/ripgrep#installation")
	return false
}

// installRipgrepMacOS 在 macOS 上安装 ripgrep
func installRipgrepMacOS() bool {
	// 尝试使用 homebrew
	if isCommandAvailable("brew") {
		cmd := exec.Command("brew", "install", "ripgrep")
		if cmd.Run() == nil {
			return true
		}
	}

	// 尝试使用 MacPorts
	if isCommandAvailable("port") {
		cmd := exec.Command("port", "install", "ripgrep")
		if cmd.Run() == nil {
			return true
		}
	}

	fmt.Println("macOS: 未找到包管理器 (homebrew/macports)，无法自动安装 ripgrep")
	fmt.Println("请手动安装: brew install ripgrep")
	return false
}

// isCommandAvailable 检查命令是否可用
func isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// GetOptimalSearchCommand 获取最优的搜索命令
func GetOptimalSearchCommand() string {
	if isRipgrepAvailable() {
		return "rg"
	}
	return "grep"
}

// BuildSearchCommand 构建搜索命令
func BuildSearchCommand(pattern string, path string, searchCmd string) string {
	if searchCmd == "rg" {
		// ripgrep 命令构建
		cmd := fmt.Sprintf("rg \"%s\"", pattern)
		if path != "" && path != "." {
			cmd += fmt.Sprintf(" \"%s\"", path)
		}
		return cmd
	} else {
		// grep 命令构建
		cmd := fmt.Sprintf("grep -r \"%s\"", pattern)
		if path != "" && path != "." {
			cmd += fmt.Sprintf(" \"%s\"", path)
		} else {
			cmd += " ."
		}
		return cmd
	}
}

// LogRipgrepStatus 记录 ripgrep 状态到日志
func LogRipgrepStatus() {
	// 确保log目录存在
	if _, err := os.Stat("log"); os.IsNotExist(err) {
		os.MkdirAll("log", 0755)
	}

	// 打开或创建ripgrep.txt文件
	file, err := os.OpenFile("log/ripgrep.txt", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return
	}
	defer file.Close()

	searchCmd, available := CheckAndInstallRipgrep()
	status := "可用"
	if !available {
		status = "不可用"
	}

	fmt.Fprintf(file, "=== Ripgrep 状态检查 ===\n")
	fmt.Fprintf(file, "操作系统: %s\n", runtime.GOOS)
	fmt.Fprintf(file, "架构: %s\n", runtime.GOARCH)
	fmt.Fprintf(file, "推荐搜索命令: %s (%s)\n", searchCmd, status)
	
	if available {
		// 获取版本信息
		cmd := exec.Command("rg", "--version")
		output, err := cmd.Output()
		if err == nil {
			version := strings.Split(string(output), "\n")[0]
			fmt.Fprintf(file, "版本信息: %s\n", version)
		}
	}
	fmt.Fprintf(file, "========================\n\n")
}