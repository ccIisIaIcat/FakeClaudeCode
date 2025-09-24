package main

import (
	"fmt"
	"log"
	"lukatincode/coder"
	"os"

	"github.com/ccIisIaIcat/GoAgent/agent/general"
)

func main() {

	config, err := general.LoadConfig("./LLMConfig.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	// 读取文件内容
	data, err := os.ReadFile("./SystemPromote/systempromote.txt")
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	// 转换为字符串并启动TUI界面
	content := string(data)
	lukatinCode := coder.GenLukatinCode(config, content)

	fmt.Println("正在启动 LukatinCode Bubble Tea TUI 界面...")
	if err := lukatinCode.StartBubbleTUI(); err != nil {
		log.Fatalf("Bubble Tea TUI应用启动失败: %v", err)
	}
}
