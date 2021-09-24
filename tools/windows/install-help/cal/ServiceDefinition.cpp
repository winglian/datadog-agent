#include "stdafx.h"
#include "ServiceDefinition.h"
#include <sstream>

std::vector<wchar_t> formatDependencies(const ServiceDefinition::deps_t &dependencies)
{
    std::vector<wchar_t> formattedDeps;
    for (auto dep : dependencies)
    {
        formattedDeps.insert(formattedDeps.end(), dep.begin(), dep.end());
        formattedDeps.push_back(L'\0');
    }
    formattedDeps.push_back(L'\0');
    return formattedDeps;
}

ServiceDefinition::ServiceDefinition()
    : _svcName()
    , _displayName()
    , _displayDescription()
    , _access(SERVICE_ALL_ACCESS)
    , _serviceType(SERVICE_WIN32_OWN_PROCESS)
    , _startType(SERVICE_DEMAND_START)
    , _errorControl(SERVICE_ERROR_NORMAL)
    , _binaryPathName()
    , _loadOrderGroup()         // not needed
    , _dependencies()           // no dependencies to start
    , _serviceUsername()        // will set to LOCAL_SYSTEM by default
    , _serviceUserPassword()    // no password for LOCAL_SYSTEM
{
}

ServiceDefinition::ServiceDefinition(const std::wstring &name)
    : _svcName(name)
    , _displayName()
    , _displayDescription()
    , _access(SERVICE_ALL_ACCESS)
    , _serviceType(SERVICE_WIN32_OWN_PROCESS)
    , _startType(SERVICE_DEMAND_START)
    , _errorControl(SERVICE_ERROR_NORMAL)
    , _binaryPathName()
    , _loadOrderGroup()         // not needed
    , _dependencies()           // no dependencies to start
    , _serviceUsername()        // will set to LOCAL_SYSTEM by default
    , _serviceUserPassword()    // no password for LOCAL_SYSTEM
{
}

ServiceDefinition::ServiceDefinition(const std::wstring &name, const std::wstring &display, const std::wstring &desc,
                                     const std::wstring &path, DWORD st,
                                     const std::wstring &user, const std::wstring &pass)
    : _svcName(name)
    , _displayName(display)
    , _displayDescription(desc)
    , _access(SERVICE_ALL_ACCESS)
    , _serviceType(SERVICE_WIN32_OWN_PROCESS)
    , _startType(st)
    , _errorControl(SERVICE_ERROR_NORMAL)
    , _binaryPathName(path)
    , _loadOrderGroup()             // not needed
    , _serviceUsername(user)
    , _serviceUserPassword(pass)    // no password for LOCAL_SYSTEM
{
}

void ServiceDefinition::addDependency(const std::wstring &serviceName)
{
    _dependencies.push_back(serviceName);
}

void ServiceDefinition::addDependency(ServiceDefinition const &serviceDef)
{
    addDependency(serviceDef.getServiceName());
}

const wchar_t *CStrOrNull(const std::wstring &str)
{
    if (str.empty())
    {
        return nullptr;
    }
    return str.c_str();
}

DWORD ServiceDefinition::create(SC_HANDLE hMgr)
{
    DWORD retval = 0;
    WcaLog(LOGMSG_STANDARD, "serviceDef::create()");
    auto formattedDeps = formatDependencies(_dependencies);
    auto hService = service_handle_p(CreateService(
        hMgr                                /*SC_HANDLE hSCManager*/
        , _svcName.c_str()                  /*LPCWSTR   lpServiceName*/
        , CStrOrNull(_displayName)          /*LPCWSTR   lpDisplayName*/
        , _access                           /*DWORD     dwDesiredAccess*/
        , _serviceType                      /*DWORD     dwServiceType*/
        , _startType                        /*DWORD     dwStartType*/
        , _errorControl                     /*DWORD     dwErrorControl*/
        , CStrOrNull(_binaryPathName)       /*LPCWSTR   lpBinaryPathName*/
        , CStrOrNull(_loadOrderGroup)       /*LPCWSTR   lpLoadOrderGroup*/
        , nullptr                           /*LPCWSTR   lpdwTagId*/
        , &formattedDeps[0]                 /*LPCWSTR   lpDependencies*/
        , CStrOrNull(_serviceUsername)      /*LPCWSTR   lpServiceStartName*/
        , CStrOrNull(_serviceUserPassword)  /*LPCWSTR   lpPassword*/
    ));
    if (hService == nullptr)
    {

        retval = GetLastError();
        WcaLog(LOGMSG_STANDARD, "Failed to CreateService %d", retval);
        return retval;
    }
    WcaLog(LOGMSG_STANDARD, "Created Service");
    if (_startType == SERVICE_AUTO_START)
    {
        // make it delayed-auto-start
        SERVICE_DELAYED_AUTO_START_INFO inf = {TRUE};
        WcaLog(LOGMSG_STANDARD, "setting to delayed auto start");
        ChangeServiceConfig2(hService.get(), SERVICE_CONFIG_DELAYED_AUTO_START_INFO, (LPVOID)&inf);
        WcaLog(LOGMSG_STANDARD, "done setting to delayed auto start");
    }
    // set the description
    if (!_displayDescription.empty())
    {
        WcaLog(LOGMSG_STANDARD, "setting description");
        SERVICE_DESCRIPTION desc = {(LPWSTR)_displayDescription.c_str()};
        ChangeServiceConfig2(hService.get(), SERVICE_CONFIG_DESCRIPTION, (LPVOID)&desc);
        WcaLog(LOGMSG_STANDARD, "done setting description");
    }
    // set the error recovery actions
    SC_ACTION actions[4] = {
        {SC_ACTION_RESTART, 60000}, // restart after 60 seconds
        {SC_ACTION_RESTART, 60000}, // restart after 60 seconds
        {SC_ACTION_RESTART, 60000}, // restart after 60 seconds
        {SC_ACTION_NONE, 0},        // restart after 60 seconds
    };
    SERVICE_FAILURE_ACTIONS failactions = {60,   // reset count after 60 seconds
                                           NULL, // no reboot message
                                           NULL, // no command to execute
                                           4,    // 4 actions
                                           actions};
    WcaLog(LOGMSG_STANDARD, "Setting failure actions");
    ChangeServiceConfig2(hService.get(), SERVICE_CONFIG_FAILURE_ACTIONS, (LPVOID)&failactions);
    WcaLog(LOGMSG_STANDARD, "Done with create() %d", retval);
    return retval;
}

DWORD ServiceDefinition::destroy(SC_HANDLE hMgr)
{
    service_handle_p hService = service_handle_p(OpenService(hMgr, _svcName.c_str(), DELETE));
    if (!hService)
    {
        return GetLastError();
    }
    DWORD retval = 0;
    if (!DeleteService(hService.get()))
    {
        retval = GetLastError();
    }
    return retval;
}

DWORD ServiceDefinition::verify(SC_HANDLE hMgr)
{
    service_handle_p hService = service_handle_p(OpenService(hMgr, _svcName.c_str(), SC_MANAGER_ALL_ACCESS));
    if (!hService)
    {
        return GetLastError();
    }
    DWORD retval = 0;

    //////
    // from 6.11 to 6.12, the location of the service binary changed.  Check the location
    // vs the expected location, and change if it's different
    QUERY_SERVICE_CONFIGW *cfg = nullptr;
    std::vector<char> buffer;
    DWORD needed = 0;
    (void)QueryServiceConfigW(hService.get(), nullptr, 0, &needed);
    DWORD lastError = GetLastError();
    if (lastError != ERROR_INSUFFICIENT_BUFFER)
    {
        WcaLog(LOGMSG_STANDARD, "Failed to query service status %d\n", retval);
        return lastError;
    }

    buffer.resize(needed);
    cfg = reinterpret_cast<decltype(cfg)>(&buffer[0]);
    if (!QueryServiceConfigW(hService.get(), cfg, needed, &needed))
    {
        WcaLog(LOGMSG_STANDARD, "Failed to query service status %d\n", retval);
        return GetLastError();
    }

    if (_wcsicmp(cfg->lpBinaryPathName, _binaryPathName.c_str()) == 0)
    {
        // nothing to do, already correctly configured
        WcaLog(LOGMSG_STANDARD, "Service path already correct");
    }
    else
    {
        BOOL bRet = ChangeServiceConfigW(hService.get(), SERVICE_NO_CHANGE, SERVICE_NO_CHANGE, SERVICE_NO_CHANGE,
                                         _binaryPathName.c_str(), NULL, NULL, NULL, NULL, NULL, NULL);
        if (!bRet)
        {
            retval = GetLastError();
            WcaLog(LOGMSG_STANDARD, "Failed to update service config %d\n", retval);
        }
        WcaLog(LOGMSG_STANDARD, "Updated path for existing service");
    }
    {
        WcaLog(LOGMSG_STANDARD, "Resetting dependencies");
        BOOL bRet = ChangeServiceConfigW(hService.get(), SERVICE_NO_CHANGE, SERVICE_NO_CHANGE, SERVICE_NO_CHANGE, NULL, NULL,
                                         NULL, _binaryPathName.c_str(), NULL, NULL, NULL);
        if (!bRet)
        {
            retval = GetLastError();
            WcaLog(LOGMSG_STANDARD, "Failed to update service dependency config %d\n", retval);
        }
        WcaLog(LOGMSG_STANDARD, "Updated dependencies for existing service");
    }

    return retval;
}

const std::wstring &ServiceDefinition::getServiceName() const
{
    return _svcName;
}
