if not exist c:\mnt\ goto nomntdir

@echo c:\mnt found, continuing

mkdir \dev\go\src\github.com\DataDog\datadog-agent
cd \dev\go\src\github.com\DataDog\datadog-agent
xcopy /e/s/h/q c:\mnt\*.*

@echo GOPATH %GOPATH%

REM Section to pre-install libyajl2 gem with fix for gcc10 compatibility
Powershell -C "ridk enable; ./tasks/winbuildscripts/libyajl2_install.ps1"
Powershell -C "ridk enable; cd omnibus; bundle install"

inv -e deps || exit /b 103

cd \mnt
REM We don't want to archive agent itself, only other deps
Powershell -C "Compress-Archive \gomodcache modcache.zip"

:nomntdir
@echo directory not mounted, parameters incorrect
exit /b 1
goto :EOF
