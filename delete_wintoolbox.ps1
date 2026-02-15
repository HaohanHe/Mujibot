# 需要以管理员身份运行
$ErrorActionPreference = "SilentlyContinue"

Write-Host "正在停止 winToolBox 相关服务..." -ForegroundColor Yellow

$services = @("CClearUpdateSrv", "KaoZipUpdateSrv", "pdfReaderUpdateSrv", "WinInterceptUpdateSrv", "WinToolBoxUpdateSrv")

foreach ($service in $services) {
    Write-Host "正在停止服务: $service"
    Stop-Service -Name $service -Force
    Start-Sleep -Milliseconds 500
    
    # 使用 WMI 删除服务
    $svc = Get-WmiObject -Class Win32_Service -Filter "Name='$service'"
    if ($svc) {
        $svc.Delete()
        Write-Host "已删除服务: $service" -ForegroundColor Green
    }
}

Start-Sleep -Seconds 2

Write-Host "`n正在获取文件夹所有权..." -ForegroundColor Yellow
$folder = "C:\Users\LENOVO\AppData\Local\winToolBox"
takeown /f "$folder" /r /d y | Out-Null
icacls "$folder" /grant "$env:USERNAME:F" /t | Out-Null

Write-Host "`n正在删除文件夹..." -ForegroundColor Yellow
Remove-Item -Path $folder -Recurse -Force

if (Test-Path $folder) {
    Write-Host "`n删除失败，尝试另一种方法..." -ForegroundColor Red
    # 使用 robocopy 方法
    $empty = "C:\Users\LENOVO\AppData\Local\temp_empty"
    New-Item -ItemType Directory -Path $empty -Force | Out-Null
    robocopy $empty $folder /purge /mt:8
    Remove-Item $empty -Force
    Remove-Item $folder -Recurse -Force
}

if (Test-Path $folder) {
    Write-Host "`n删除失败！文件夹仍存在。" -ForegroundColor Red
} else {
    Write-Host "`n删除成功！winToolBox 文件夹已删除。" -ForegroundColor Green
}

Write-Host "`n按任意键退出..."
$null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
