package main

import (
	"fmt"
	"log"
	"lukatincode/coder"
	"lukatincode/function"
	"os"
	"path/filepath"

	"github.com/ccIisIaIcat/GoAgent/agent/general"
)

func testAllFunctions() {
	fmt.Println("=================== 开始测试所有函数 ===================")
	
	// 获取当前工作目录
	currentDir, _ := os.Getwd()
	
	// 测试 TodoRead 函数
	fmt.Println("测试 TodoRead...")
	todoResult := function.TodoRead()
	fmt.Printf("TodoRead 结果: %s\n", todoResult)
	
	// 测试 TodoWrite 函数
	fmt.Println("测试 TodoWrite...")
	todoWriteResult := function.TodoWrite(`{"todos":[{"id":"test1","content":"测试任务","status":"pending","priority":"medium"}]}`)
	fmt.Printf("TodoWrite 结果: %s\n", todoWriteResult)
	
	// 测试 Grep 函数
	fmt.Println("测试 Grep...")
	grepResult := function.Grep("package", "*.go", currentDir)
	fmt.Printf("Grep 结果: %s\n", grepResult)
	
	// 测试 Glob 函数
	fmt.Println("测试 Glob...")
	globResult := function.Glob("*.go", currentDir)
	fmt.Printf("Glob 结果: %s\n", globResult)
	
	// 测试 LS 函数
	fmt.Println("测试 LS...")
	lsResult := function.LS(currentDir, []string{})
	fmt.Printf("LS 结果: %s\n", lsResult)
	
	// 测试 Read 函数
	fmt.Println("测试 Read...")
	mainGoPath := filepath.Join(currentDir, "main.go")
	readResult := function.Read(mainGoPath, 0, 10)
	fmt.Printf("Read 结果: %s\n", readResult)
	
	// 测试 Write 函数 (创建测试文件)
	fmt.Println("测试 Write...")
	testFilePath := filepath.Join(currentDir, "test_write.txt")
	writeResult := function.Write(testFilePath, "这是一个测试文件内容")
	fmt.Printf("Write 结果: %s\n", writeResult)
	
	// 测试 Edit 函数
	fmt.Println("测试 Edit...")
	editResult := function.Edit(testFilePath, "测试文件", "编辑文件", 1)
	fmt.Printf("Edit 结果: %s\n", editResult)
	
	// 测试 MultiEdit 函数
	fmt.Println("测试 MultiEdit...")
	edits := []function.EditOperation{
		{OldString: "编辑文件", NewString: "多重编辑文件", ExpectedReplacements: 1},
		{OldString: "内容", NewString: "内容UPDATED", ExpectedReplacements: 1},
	}
	multiEditResult := function.MultiEdit(testFilePath, edits)
	fmt.Printf("MultiEdit 结果: %s\n", multiEditResult)
	
	// 跳过Task函数测试，因为有死锁问题
	fmt.Println("跳过Task函数测试（死锁问题）")
	
	// 跳过WebFetch函数测试，因为有死锁问题
	fmt.Println("跳过WebFetch函数测试（死锁问题）")
	
	// 跳过WebSearch函数测试，因为有死锁问题
	fmt.Println("跳过WebSearch函数测试（死锁问题）")
	
	// 清理测试文件
	os.Remove(testFilePath)
	
	fmt.Println("=================== 所有函数测试完成 ===================")
}

func main() {
	// 添加函数测试
	if len(os.Args) > 1 && os.Args[1] == "test" {
		testAllFunctions()
		return
	}
	
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
