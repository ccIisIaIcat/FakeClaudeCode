package coder

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"lukatincode/function"
)

func (lc *LukatinCode) RegisterAllFunction() {
	lc.Logger.Println("开始注册函数")

	// 读取函数描述文件
	functionDescFile := "./function/function_description.json"
	data, err := ioutil.ReadFile(functionDescFile)
	if err != nil {
		lc.Logger.Printf("读取函数描述文件失败: %v", err)
		fmt.Printf("读取函数描述文件失败: %v\n", err)
		return
	}

	var functionDescs map[string]FunctionDescription
	err = json.Unmarshal(data, &functionDescs)
	if err != nil {
		lc.Logger.Printf("解析函数描述文件失败: %v", err)
		fmt.Printf("解析函数描述文件失败: %v\n", err)
		return
	}

	// 注册 Bash 函数 (使用持久化Shell)
	if desc, ok := functionDescs["Bash"]; ok {
		var paramNames []string
		var paramDescs []string
		// 添加必需参数
		for _, param := range desc.Parameters.Required {
			if paramInfo, exists := desc.Parameters.Properties[param]; exists {
				paramNames = append(paramNames, param)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		// 添加可选参数
		for paramName, paramInfo := range desc.Parameters.Properties {
			isRequired := false
			for _, req := range desc.Parameters.Required {
				if req == paramName {
					isRequired = true
					break
				}
			}
			if !isRequired {
				paramNames = append(paramNames, paramName)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		err := lc.CM.RegisterFunction("Bash", desc.Description, lc.Bash, paramNames, paramDescs)
		if err != nil {
			lc.Logger.Printf("注册Bash函数失败: %v", err)
			fmt.Printf("注册Bash函数失败: %v\n", err)
		} else {
			lc.Logger.Println("成功注册Bash函数 (持久化Shell)")
		}
	}

	// 注册 TodoRead 函数
	if desc, ok := functionDescs["TodoRead"]; ok {
		err := lc.CM.RegisterFunction("TodoRead", desc.Description, function.TodoRead, []string{}, []string{})
		if err != nil {
			lc.Logger.Printf("注册TodoRead函数失败: %v", err)
			fmt.Printf("注册TodoRead函数失败: %v\n", err)
		} else {
			lc.Logger.Println("成功注册TodoRead函数")
			// fmt.Printf("成功注册TodoRead函数\n")
		}
	}

	// 注册 TodoWrite 函数
	if desc, ok := functionDescs["TodoWrite"]; ok {
		var paramNames []string
		var paramDescs []string
		for _, param := range desc.Parameters.Required {
			if paramInfo, exists := desc.Parameters.Properties[param]; exists {
				paramNames = append(paramNames, param)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		err := lc.CM.RegisterFunction("TodoWrite", desc.Description, function.TodoWrite, paramNames, paramDescs)
		if err != nil {
			lc.Logger.Printf("注册TodoWrite函数失败: %v", err)
			fmt.Printf("注册TodoWrite函数失败: %v\n", err)
		} else {
			lc.Logger.Println("成功注册TodoWrite函数")
			// fmt.Printf("成功注册TodoWrite函数\n")
		}
	}

	// 注册 Grep 函数
	if desc, ok := functionDescs["Grep"]; ok {
		var paramNames []string
		var paramDescs []string
		for _, param := range desc.Parameters.Required {
			if paramInfo, exists := desc.Parameters.Properties[param]; exists {
				paramNames = append(paramNames, param)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		// 添加可选参数
		for paramName, paramInfo := range desc.Parameters.Properties {
			isRequired := false
			for _, req := range desc.Parameters.Required {
				if req == paramName {
					isRequired = true
					break
				}
			}
			if !isRequired {
				paramNames = append(paramNames, paramName)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		err := lc.CM.RegisterFunction("Grep", desc.Description, function.Grep, paramNames, paramDescs)
		if err != nil {
			lc.Logger.Printf("注册Grep函数失败: %v", err)
			fmt.Printf("注册Grep函数失败: %v\n", err)
		} else {
			lc.Logger.Println("成功注册Grep函数")
			// fmt.Printf("成功注册Grep函数\n")
		}
	}

	// 注册 Glob 函数
	if desc, ok := functionDescs["Glob"]; ok {
		var paramNames []string
		var paramDescs []string
		for _, param := range desc.Parameters.Required {
			if paramInfo, exists := desc.Parameters.Properties[param]; exists {
				paramNames = append(paramNames, param)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		// 添加可选参数
		for paramName, paramInfo := range desc.Parameters.Properties {
			isRequired := false
			for _, req := range desc.Parameters.Required {
				if req == paramName {
					isRequired = true
					break
				}
			}
			if !isRequired {
				paramNames = append(paramNames, paramName)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		err := lc.CM.RegisterFunction("Glob", desc.Description, function.Glob, paramNames, paramDescs)
		if err != nil {
			lc.Logger.Printf("注册Glob函数失败: %v", err)
			fmt.Printf("注册Glob函数失败: %v\n", err)
		} else {
			lc.Logger.Println("成功注册Glob函数")
			// fmt.Printf("成功注册Glob函数\n")
		}
	}

	// 注册 Task 函数
	if desc, ok := functionDescs["Task"]; ok {
		var paramNames []string
		var paramDescs []string
		for _, param := range desc.Parameters.Required {
			if paramInfo, exists := desc.Parameters.Properties[param]; exists {
				paramNames = append(paramNames, param)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		err := lc.CM.RegisterFunction("Task", desc.Description, function.Task, paramNames, paramDescs)
		if err != nil {
			lc.Logger.Printf("注册Task函数失败: %v", err)
			fmt.Printf("注册Task函数失败: %v\n", err)
		} else {
			lc.Logger.Println("成功注册Task函数")
			// fmt.Printf("成功注册Task函数\n")
		}
	}

	// 注册 LS 函数
	if desc, ok := functionDescs["LS"]; ok {
		var paramNames []string
		var paramDescs []string
		for _, param := range desc.Parameters.Required {
			if paramInfo, exists := desc.Parameters.Properties[param]; exists {
				paramNames = append(paramNames, param)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		// 添加可选参数
		for paramName, paramInfo := range desc.Parameters.Properties {
			isRequired := false
			for _, req := range desc.Parameters.Required {
				if req == paramName {
					isRequired = true
					break
				}
			}
			if !isRequired {
				paramNames = append(paramNames, paramName)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		err := lc.CM.RegisterFunction("LS", desc.Description, function.LS, paramNames, paramDescs)
		if err != nil {
			lc.Logger.Printf("注册LS函数失败: %v", err)
			fmt.Printf("注册LS函数失败: %v\n", err)
		} else {
			lc.Logger.Println("成功注册LS函数")
			// fmt.Printf("成功注册LS函数\n")
		}
	}

	// 注册 Read 函数
	if desc, ok := functionDescs["Read"]; ok {
		var paramNames []string
		var paramDescs []string
		for _, param := range desc.Parameters.Required {
			if paramInfo, exists := desc.Parameters.Properties[param]; exists {
				paramNames = append(paramNames, param)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		// 添加可选参数
		for paramName, paramInfo := range desc.Parameters.Properties {
			isRequired := false
			for _, req := range desc.Parameters.Required {
				if req == paramName {
					isRequired = true
					break
				}
			}
			if !isRequired {
				paramNames = append(paramNames, paramName)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		err := lc.CM.RegisterFunction("Read", desc.Description, function.Read, paramNames, paramDescs)
		if err != nil {
			lc.Logger.Printf("注册Read函数失败: %v", err)
			fmt.Printf("注册Read函数失败: %v\n", err)
		} else {
			lc.Logger.Println("成功注册Read函数")
			// fmt.Printf("成功注册Read函数\n")
		}
	}

	// 注册 Edit 函数
	if desc, ok := functionDescs["Edit"]; ok {
		var paramNames []string
		var paramDescs []string
		for _, param := range desc.Parameters.Required {
			if paramInfo, exists := desc.Parameters.Properties[param]; exists {
				paramNames = append(paramNames, param)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		// 添加可选参数
		for paramName, paramInfo := range desc.Parameters.Properties {
			isRequired := false
			for _, req := range desc.Parameters.Required {
				if req == paramName {
					isRequired = true
					break
				}
			}
			if !isRequired {
				paramNames = append(paramNames, paramName)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		err := lc.CM.RegisterFunction("Edit", desc.Description, function.Edit, paramNames, paramDescs)
		if err != nil {
			lc.Logger.Printf("注册Edit函数失败: %v", err)
			fmt.Printf("注册Edit函数失败: %v\n", err)
		} else {
			lc.Logger.Println("成功注册Edit函数")
			// fmt.Printf("成功注册Edit函数\n")
		}
	}

	// 注册 MultiEdit 函数
	if desc, ok := functionDescs["MultiEdit"]; ok {
		var paramNames []string
		var paramDescs []string
		for _, param := range desc.Parameters.Required {
			if paramInfo, exists := desc.Parameters.Properties[param]; exists {
				paramNames = append(paramNames, param)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		err := lc.CM.RegisterFunction("MultiEdit", desc.Description, function.MultiEdit, paramNames, paramDescs)
		if err != nil {
			lc.Logger.Printf("注册MultiEdit函数失败: %v", err)
			fmt.Printf("注册MultiEdit函数失败: %v\n", err)
		} else {
			lc.Logger.Println("成功注册MultiEdit函数")
			// fmt.Printf("成功注册MultiEdit函数\n")
		}
	}

	// 注册 Write 函数
	if desc, ok := functionDescs["Write"]; ok {
		var paramNames []string
		var paramDescs []string
		for _, param := range desc.Parameters.Required {
			if paramInfo, exists := desc.Parameters.Properties[param]; exists {
				paramNames = append(paramNames, param)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		err := lc.CM.RegisterFunction("Write", desc.Description, function.Write, paramNames, paramDescs)
		if err != nil {
			lc.Logger.Printf("注册Write函数失败: %v", err)
			fmt.Printf("注册Write函数失败: %v\n", err)
		} else {
			lc.Logger.Println("成功注册Write函数")
			// fmt.Printf("成功注册Write函数\n")
		}
	}

	// 注册 WebFetch 函数
	if desc, ok := functionDescs["WebFetch"]; ok {
		var paramNames []string
		var paramDescs []string
		for _, param := range desc.Parameters.Required {
			if paramInfo, exists := desc.Parameters.Properties[param]; exists {
				paramNames = append(paramNames, param)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		err := lc.CM.RegisterFunction("WebFetch", desc.Description, function.WebFetch, paramNames, paramDescs)
		if err != nil {
			lc.Logger.Printf("注册WebFetch函数失败: %v", err)
			fmt.Printf("注册WebFetch函数失败: %v\n", err)
		} else {
			lc.Logger.Println("成功注册WebFetch函数")
			// fmt.Printf("成功注册WebFetch函数\n")
		}
	}

	// 注册 WebSearch 函数
	if desc, ok := functionDescs["WebSearch"]; ok {
		var paramNames []string
		var paramDescs []string
		for _, param := range desc.Parameters.Required {
			if paramInfo, exists := desc.Parameters.Properties[param]; exists {
				paramNames = append(paramNames, param)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		// 添加可选参数
		for paramName, paramInfo := range desc.Parameters.Properties {
			isRequired := false
			for _, req := range desc.Parameters.Required {
				if req == paramName {
					isRequired = true
					break
				}
			}
			if !isRequired {
				paramNames = append(paramNames, paramName)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		err := lc.CM.RegisterFunction("WebSearch", desc.Description, function.WebSearch, paramNames, paramDescs)
		if err != nil {
			lc.Logger.Printf("注册WebSearch函数失败: %v", err)
			fmt.Printf("注册WebSearch函数失败: %v\n", err)
		} else {
			lc.Logger.Println("成功注册WebSearch函数")
			// fmt.Printf("成功注册WebSearch函数\n")
		}
	}

	lc.Logger.Println("函数注册完成")
}
