$ErrorActionPreference = "Stop"
Trap { Write-Host "Error in install.ps1: $_" }

function Install-Service {
  param(
    [Parameter(Mandatory=$true)][string] $SvcName,
    [Parameter(Mandatory=$true)][string] $BinPath,
    [Parameter(Mandatory=$false)][string[]] $Depends
  )
  if ( $Depends.count -gt 0 ){
      New-Service -Name $SvcName -StartupType Manual -BinaryPathName $BinPath -Depends $Depends
  } else {
      New-Service -Name $SvcName -StartupType Manual -BinaryPathName $BinPath
  }
  $eventSourceData = new-object System.Diagnostics.EventSourceCreationData("$SvcName", "Application")  
  $eventSourceData.CategoryResourceFile = $BinPath
  $eventSourceData.MessageResourceFile = $BinPath

  If (![System.Diagnostics.EventLog]::SourceExists($eventSourceData.Source))
  {      
  [System.Diagnostics.EventLog]::CreateEventSource($eventSourceData)  
  } 
}

if ("$env:WITH_JMX" -ne "false") {
    Invoke-WebRequest -OutFile jre-11.0.6.zip https://github.com/AdoptOpenJDK/openjdk11-binaries/releases/download/jdk-11.0.6%2B10/OpenJDK11U-jre_x64_windows_hotspot_11.0.6_10.zip
    Expand-Archive -Path jre-11.0.6.zip -DestinationPath C:/
    Remove-Item jre-11.0.6.zip
    Move-Item C:/jdk-11.0.6+10-jre/ C:/java
    # Add java to the path for jmxfetch
    setx /m PATH "$Env:Path;C:/java/bin"
    $Env:Path="$Env:Path;C:/java/bin"
}

$services = [ordered]@{
  "datadogagent" = "C:\Program Files\Datadog\Datadog Agent\bin\agent.exe",@()
  "datadog-process-agent" = "C:\Program Files\Datadog\Datadog Agent\bin\agent\process-agent.exe",@("datadogagent")
  "datadog-trace-agent" = "C:\Program Files\Datadog\Datadog Agent\bin\agent\trace-agent.exe",@("datadogagent")
}

foreach ($s in $services.Keys) {
    Install-Service -SvcName $s -BinPath $services[$s][0] $services[$s][1]
}

# Allow to run agent binaries as `agent`
setx /m PATH "$Env:Path;C:/Program Files/Datadog/Datadog Agent/bin;C:/Program Files/Datadog/Datadog Agent/bin/agent"
$Env:Path="$Env:Path;C:/Program Files/Datadog/Datadog Agent/bin;C:/Program Files/Datadog/Datadog Agent/bin/agent"

# Set variable indicating we are running in a container
setx /m DOCKER_DD_AGENT "true"
$Env:DOCKER_DD_AGENT="true"

# Create install_info
Write-Output @"
---
install_method:
  tool: docker-win
  tool_version: docker-win-$env:INSTALL_INFO
  installer_version: docker-win-$env:INSTALL_INFO
"@ > C:/ProgramData/Datadog/install_info
