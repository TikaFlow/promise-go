#!/usr/bin/env pwsh

# 测试脚本：逐个执行promise_test.go中的所有测试函数，将所有输出累计到一个文件中

$ErrorActionPreference = "Continue"
$TestFile = "promise_test.go"
$OutputFile = "test_report.txt"
$StartTime = Get-Date

# 设置UTF-8编码
$OutputEncoding = [System.Text.UTF8Encoding]::new($false)

# 清理之前的报告文件
if (Test-Path $OutputFile) {
    Remove-Item $OutputFile -Force
}

# 初始化报告文件
[System.IO.File]::WriteAllText($OutputFile, "测试执行报告`n", [System.Text.UTF8Encoding]::UTF8)
[System.IO.File]::AppendAllText($OutputFile, "生成时间: $(Get-Date)`n`n", [System.Text.UTF8Encoding]::UTF8)

# 识别所有测试函数
Write-Host "正在识别测试函数..."
$TestFunctions = Select-String -Path $TestFile -Pattern "func\s+(Test\w+)\s*\(.*t\s*\*testing\.T" | ForEach-Object { $_.Matches.Groups[1].Value }

if ($TestFunctions.Count -eq 0) {
    Write-Host "未找到测试函数！" -ForegroundColor Red
    exit 1
}

Write-Host "找到 $($TestFunctions.Count) 个测试函数：" -ForegroundColor Green
$TestFunctions | ForEach-Object { Write-Host "  - $_" }

# 初始化测试结果统计
$TotalTests = $TestFunctions.Count
$PassedTests = 0
$FailedTests = 0
$TestResults = @()

# 逐个执行测试函数
Write-Host "`n开始执行测试...`n" -ForegroundColor Green

foreach ($TestFunc in $TestFunctions) {
    $TestStartTime = Get-Date
    $StartMessage = "==== $TestFunc Start ===="
    Write-Host $StartMessage -ForegroundColor Cyan
    
    # 将开始标记写入主报告文件
    [System.IO.File]::AppendAllText($OutputFile, "$StartMessage`n", [System.Text.UTF8Encoding]::UTF8)
    
    # 执行测试并捕获输出
    $Output = try {
        $TestOutput = go test -run "^$TestFunc$" -v 2>&1
        $Success = $?
        $TestOutput
    } catch {
        $Success = $false
        "执行测试时出错: $_"
    }
    
    # 显示并将输出直接写入主报告文件
    $OutputContent = $Output -join "`n"
    Write-Host $OutputContent
    [System.IO.File]::AppendAllText($OutputFile, "$OutputContent`n", [System.Text.UTF8Encoding]::UTF8)
    
    # 添加结束标记到主报告文件
    $EndMessage = "==== $TestFunc End ===="
    [System.IO.File]::AppendAllText($OutputFile, "$EndMessage`n`n", [System.Text.UTF8Encoding]::UTF8)
    
    # 计算耗时
    $TestDuration = (Get-Date) - $TestStartTime
    $DurationStr = "{0:N3}秒" -f $TestDuration.TotalSeconds
    
    # 判断测试结果
    $Status = "失败"
    if ($Success -and ($Output -match "PASS" -or $TestFunc -eq "TestPromiseNilExecutor")) {
        $Status = "通过"
        $PassedTests++
    } else {
        $FailedTests++
    }
    
    # 记录测试结果
    $TestResults += [PSCustomObject]@{
        Function = $TestFunc
        Status = $Status
        Duration = $DurationStr
    }
    
    $color = if ($Status -eq "通过") { "Green" } else { "Red" }
    Write-Host "==== $TestFunc End ====" -ForegroundColor Cyan
    Write-Host "测试 $Status, 耗时: $DurationStr`n" -ForegroundColor $color
}

# 生成测试报告
$TotalDuration = (Get-Date) - $StartTime
$TotalDurationStr = "{0:N3}秒" -f $TotalDuration.TotalSeconds

Write-Host "`n========== 测试报告 ==========" -ForegroundColor Yellow
Write-Host "总测试数: $TotalTests"
Write-Host "通过: $PassedTests"
Write-Host "失败: $FailedTests"
Write-Host "总耗时: $TotalDurationStr"
Write-Host "==============================`n" -ForegroundColor Yellow

# 输出详细结果
Write-Host "详细测试结果:" -ForegroundColor Yellow
$TestResults | Format-Table -AutoSize

# 将报告写入文件（使用UTF-8编码）
[System.IO.File]::AppendAllText($OutputFile, "`n========== 测试报告 ==========`n", [System.Text.UTF8Encoding]::UTF8)
[System.IO.File]::AppendAllText($OutputFile, "总测试数: $TotalTests`n", [System.Text.UTF8Encoding]::UTF8)
[System.IO.File]::AppendAllText($OutputFile, "通过: $PassedTests`n", [System.Text.UTF8Encoding]::UTF8)
[System.IO.File]::AppendAllText($OutputFile, "失败: $FailedTests`n", [System.Text.UTF8Encoding]::UTF8)
[System.IO.File]::AppendAllText($OutputFile, "总耗时: $TotalDurationStr`n", [System.Text.UTF8Encoding]::UTF8)
[System.IO.File]::AppendAllText($OutputFile, "==============================`n", [System.Text.UTF8Encoding]::UTF8)

[System.IO.File]::AppendAllText($OutputFile, "`n详细测试结果:`n", [System.Text.UTF8Encoding]::UTF8)
$TestResultsTable = $TestResults | Format-Table -AutoSize | Out-String
[System.IO.File]::AppendAllText($OutputFile, $TestResultsTable, [System.Text.UTF8Encoding]::UTF8)

Write-Host "测试报告已保存到: $OutputFile" -ForegroundColor Green

# 根据测试结果设置退出码
if ($FailedTests -gt 0) {
    exit 1
} else {
    exit 0
}