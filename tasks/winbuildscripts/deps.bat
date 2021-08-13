if not exist c:\mnt\ goto nomntdir

@echo c:\mnt found, continuing

mkdir \dev\go\src\github.com\DataDog\datadog-agent
cd \dev\go\src\github.com\DataDog\datadog-agent
xcopy /e/s/h/q c:\mnt\*.*

@echo GOPATH %GOPATH%

pip3 install -r requirements.txt || exit /b 102

inv -e deps || exit /b 103
@echo Done fetching deps

cd \mnt\omnibus\pkg
REM We don't want to archive agent itself, only other deps
Powershell -C "Compress-Archive \gomodcache modcache.zip -CompressionLevel fastest -Force"
@echo Done compressing deps

:nomntdir
@echo directory not mounted, parameters incorrect
exit /b 1
goto :EOF
