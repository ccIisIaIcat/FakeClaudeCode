package coder

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

// PersistentShell 持久化Shell结构体
type PersistentShell struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	scanner   *bufio.Scanner
	errScanner *bufio.Scanner
	isRunning bool
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewPersistentShell 创建新的持久化Shell实例
func NewPersistentShell() *PersistentShell {
	ctx, cancel := context.WithCancel(context.Background())
	return &PersistentShell{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start 启动shell子进程
func (ps *PersistentShell) Start() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	
	if ps.isRunning {
		return fmt.Errorf("shell is already running")
	}
	
	// 获取当前工作目录
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %v", err)
	}
	
	// 创建shell命令 (根据操作系统选择)
	if runtime.GOOS == "windows" {
		// 在Windows上使用cmd，更兼容
		ps.cmd = exec.CommandContext(ps.ctx, "cmd", "/k")
	} else {
		// 在Unix系统上使用bash
		ps.cmd = exec.CommandContext(ps.ctx, "/bin/bash")
	}
	
	// 设置工作目录为当前目录
	ps.cmd.Dir = currentDir
	
	// 获取stdin, stdout, stderr管道
	stdin, err := ps.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %v", err)
	}
	ps.stdin = stdin
	
	stdout, err := ps.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	ps.stdout = stdout
	
	stderr, err := ps.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}
	ps.stderr = stderr
	
	// 启动进程
	if err := ps.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start shell process: %v", err)
	}
	
	// 创建扫描器
	ps.scanner = bufio.NewScanner(ps.stdout)
	ps.errScanner = bufio.NewScanner(ps.stderr)
	
	ps.isRunning = true
	return nil
}

// ExecuteCommand 执行命令并返回结果
func (ps *PersistentShell) ExecuteCommand(command string) (string, error) {
	ps.mu.RLock()
	if !ps.isRunning {
		ps.mu.RUnlock()
		return "", fmt.Errorf("shell is not running")
	}
	ps.mu.RUnlock()
	
	// 添加命令结束标记 (根据操作系统调整格式)
	marker := fmt.Sprintf("__CMD_END_%d__", time.Now().UnixNano())
	var fullCommand string
	
	if runtime.GOOS == "windows" {
		// Windows cmd格式
		fullCommand = fmt.Sprintf("%s & echo %s\n", command, marker)
	} else {
		// Bash格式
		fullCommand = fmt.Sprintf("%s; echo '%s'\n", command, marker)
	}
	
	// 写入命令
	_, err := ps.stdin.Write([]byte(fullCommand))
	if err != nil {
		return "", fmt.Errorf("failed to write command: %v", err)
	}
	
	// 读取输出直到遇到标记
	var output []string
	var stderr_output []string
	
	// 使用channel来同时读取stdout和stderr
	outputChan := make(chan string)
	errorChan := make(chan string)
	doneChan := make(chan bool)
	
	// 读取stdout
	go func() {
		for ps.scanner.Scan() {
			line := ps.scanner.Text()
			if line == marker {
				doneChan <- true
				return
			}
			outputChan <- line
		}
	}()
	
	// 读取stderr
	go func() {
		for ps.errScanner.Scan() {
			errorChan <- ps.errScanner.Text()
		}
	}()
	
	// 收集输出，设置超时
	timeout := time.After(30 * time.Second)
	
	for {
		select {
		case line := <-outputChan:
			output = append(output, line)
		case errLine := <-errorChan:
			stderr_output = append(stderr_output, errLine)
		case <-doneChan:
			// 命令执行完成
			result := ""
			if len(output) > 0 {
				result += fmt.Sprintf("stdout:\n%s\n", joinLines(output))
			}
			if len(stderr_output) > 0 {
				result += fmt.Sprintf("stderr:\n%s\n", joinLines(stderr_output))
			}
			if result == "" {
				result = "Command executed successfully (no output)"
			}
			return result, nil
		case <-timeout:
			return "", fmt.Errorf("command execution timeout")
		case <-ps.ctx.Done():
			return "", fmt.Errorf("shell context cancelled")
		}
	}
}

// IsRunning 检查shell是否正在运行
func (ps *PersistentShell) IsRunning() bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.isRunning
}

// Stop 停止shell进程
func (ps *PersistentShell) Stop() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	
	if !ps.isRunning {
		return fmt.Errorf("shell is not running")
	}
	
	// 发送exit命令
	if ps.stdin != nil {
		ps.stdin.Write([]byte("exit\n"))
		ps.stdin.Close()
	}
	
	// 取消上下文
	ps.cancel()
	
	// 等待进程结束
	if ps.cmd != nil && ps.cmd.Process != nil {
		// 给进程一些时间优雅退出
		done := make(chan error, 1)
		go func() {
			done <- ps.cmd.Wait()
		}()
		
		select {
		case <-done:
			// 进程已退出
		case <-time.After(5 * time.Second):
			// 超时，强制终止
			ps.cmd.Process.Kill()
			<-done
		}
	}
	
	// 关闭所有管道
	if ps.stdout != nil {
		ps.stdout.Close()
	}
	if ps.stderr != nil {
		ps.stderr.Close()
	}
	
	ps.isRunning = false
	return nil
}

// joinLines 连接字符串行
func joinLines(lines []string) string {
	result := ""
	for i, line := range lines {
		if i > 0 {
			result += "\n"
		}
		result += line
	}
	return result
}