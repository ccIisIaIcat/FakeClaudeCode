# Ripgrep 自动检测和安装

## 概述

系统在初始化时会自动检测 ripgrep (`rg`) 是否可用，如果没有安装会尝试自动安装。Ripgrep 是一个高性能的文本搜索工具，比传统的 `grep` 快很多倍。

## 支持的平台

### Windows
- **Chocolatey**: `choco install ripgrep -y`
- **Scoop**: `scoop install ripgrep`  
- **Winget**: `winget install BurntSushi.ripgrep.MSVC`

### Linux
- **Ubuntu/Debian**: `apt install ripgrep`
- **CentOS/RHEL**: `yum install ripgrep`
- **Fedora**: `dnf install ripgrep`
- **Arch**: `pacman -S ripgrep`

### macOS
- **Homebrew**: `brew install ripgrep`
- **MacPorts**: `port install ripgrep`

## 功能特性

1. **自动检测**: 启动时检测 `rg` 命令是否可用
2. **自动安装**: 尝试使用系统包管理器安装 ripgrep
3. **智能回退**: 如果 ripgrep 不可用，自动回退到传统的 `grep`
4. **命令替换**: Task 子代理中的 `grep` 命令会自动替换为 `rg` (如果可用)

## 日志记录

所有 ripgrep 相关的检测和安装信息都会记录到 `log/ripgrep.txt` 文件中，包括：
- 操作系统信息
- 检测结果
- 安装尝试记录
- 版本信息

## 手动安装

如果自动安装失败，可以手动安装：

1. 访问 [ripgrep GitHub 发布页面](https://github.com/BurntSushi/ripgrep/releases)
2. 下载适合您操作系统的版本
3. 将 `rg` 可执行文件放入 PATH 环境变量中

## 使用示例

```bash
# 传统 grep 方式
grep -r "TODO" .

# ripgrep 方式（更快更好）
rg "TODO"

# 更多 ripgrep 选项
rg "pattern" --type js     # 只搜索 JavaScript 文件
rg "pattern" -C 3          # 显示上下文 3 行
rg "pattern" --json        # JSON 格式输出
```

## 性能优势

- **速度**: 比 grep 快 2-10 倍
- **智能忽略**: 自动忽略 .gitignore、二进制文件
- **彩色输出**: 更好的可读性
- **Unicode 支持**: 完整的 UTF-8 支持
- **并行搜索**: 自动使用多核心