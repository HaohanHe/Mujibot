@echo off
echo 正在停止并删除 winToolBox 相关服务...
echo.

sc stop CClearUpdateSrv
sc stop KaoZipUpdateSrv
sc stop pdfReaderUpdateSrv
sc stop WinInterceptUpdateSrv
sc stop WinToolBoxUpdateSrv

timeout /t 3 /nobreak >nul

sc delete CClearUpdateSrv
sc delete KaoZipUpdateSrv
sc delete pdfReaderUpdateSrv
sc delete WinInterceptUpdateSrv
sc delete WinToolBoxUpdateSrv

echo.
echo 正在删除 winToolBox 文件夹...

takeown /f "C:\Users\LENOVO\AppData\Local\winToolBox" /r /d y
icacls "C:\Users\LENOVO\AppData\Local\winToolBox" /grant administrators:F /t

rd /s /q "C:\Users\LENOVO\AppData\Local\winToolBox"

echo.
if exist "C:\Users\LENOVO\AppData\Local\winToolBox" (
    echo 删除失败，文件夹仍存在
) else (
    echo 删除成功！
)

echo.
pause
