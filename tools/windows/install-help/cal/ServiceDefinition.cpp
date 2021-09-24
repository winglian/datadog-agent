#include "stdafx.h"
#include "ServiceDefinition.h"
#include <sstream>

std::vector<wchar_t> formatDependencies(const ServiceDefinition::deps_t &dependencies)
{
    std::vector<wchar_t> formattedDeps;
    for (auto dep : dependencies)
    {
        for (auto ch : dep)
        {
            formattedDeps.push_back(ch);
        }
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
    , _tagId(NULL)              // no tag identifier
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
    , _tagId(NULL)              // no tag identifier
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
    , _tagId(NULL)                  // no tag identifier
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

DWORD ServiceDefinition::create(SC_HANDLE hMgr)
{
    DWORD retval = 0;
    WcaLog(LOGMSG_STANDARD, "serviceDef::create()");
    auto formattedDeps = formatDependencies(_dependencies);
    SC_HANDLE hService = CreateService(hMgr, _svcName.c_str(), _displayName.c_str(), _access, _serviceType, _startType,
                                       _errorControl, _binaryPathName.c_str(), _loadOrderGroup.c_str(), _tagId,
                                       &formattedDeps[0], _serviceUsername.c_str(), _serviceUserPassword.c_str());
    if (!hService)
    {

        retval = GetLastError();
        WcaLog(LOGMSG_STANDARD, "Failed to CreateService %d", retval);
        return retval;
    }
    WcaLog(LOGMSG_STANDARD, "Created Service");
    if (this->_startType == SERVICE_AUTO_START)
    {
        // make it delayed-auto-start
        SERVICE_DELAYED_AUTO_START_INFO inf = {TRUE};
        WcaLog(LOGMSG_STANDARD, "setting to delayed auto start");
        ChangeServiceConfig2(hService, SERVICE_CONFIG_DELAYED_AUTO_START_INFO, (LPVOID)&inf);
        WcaLog(LOGMSG_STANDARD, "done setting to delayed auto start");
    }
    // set the description
    if (!_displayDescription.empty())
    {
        WcaLog(LOGMSG_STANDARD, "setting description");
        SERVICE_DESCRIPTION desc = {(LPWSTR)_displayDescription.c_str()};
        ChangeServiceConfig2(hService, SERVICE_CONFIG_DESCRIPTION, (LPVOID)&desc);
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
    ChangeServiceConfig2(hService, SERVICE_CONFIG_FAILURE_ACTIONS, (LPVOID)&failactions);
    WcaLog(LOGMSG_STANDARD, "Done with create() %d", retval);
    return retval;
}

DWORD ServiceDefinition::destroy(SC_HANDLE hMgr)
{
    SC_HANDLE hService = OpenService(hMgr, _svcName.c_str(), DELETE);
    if (!hService)
    {
        return GetLastError();
    }
    DWORD retval = 0;
    if (!DeleteService(hService))
    {
        retval = GetLastError();
    }
    CloseServiceHandle(hService);
    return retval;
}

DWORD ServiceDefinition::verify(SC_HANDLE hMgr)
{
    SC_HANDLE hService = OpenService(hMgr, _svcName.c_str(), SC_MANAGER_ALL_ACCESS);
    if (!hService)
    {
        return GetLastError();
    }
    DWORD retval = 0;
#define QUERY_BUF_SIZE 8192
    //////
    // from 6.11 to 6.12, the location of the service binary changed.  Check the location
    // vs the expected location, and change if it's different
    QUERY_SERVICE_CONFIGW cfg;
    DWORD needed = 0;
    if (!QueryServiceConfigW(hService, &cfg, QUERY_BUF_SIZE, &needed))
    {
        // shouldn't ever fail.  WE're supplying the largest possible buffer
        // according to the docs.
        retval = GetLastError();
        WcaLog(LOGMSG_STANDARD, "Failed to query service status %d\n", retval);
        goto done_verify;
    }
    if (_wcsicmp(cfg.lpBinaryPathName, _binaryPathName.c_str()) == 0)
    {
        // nothing to do, already correctly configured
        WcaLog(LOGMSG_STANDARD, "Service path already correct");
    }
    else
    {
        BOOL bRet = ChangeServiceConfigW(hService, SERVICE_NO_CHANGE, SERVICE_NO_CHANGE, SERVICE_NO_CHANGE,
                                         _binaryPathName.c_str(), NULL, NULL, NULL, NULL, NULL, NULL);
        if (!bRet)
        {
            retval = GetLastError();
            WcaLog(LOGMSG_STANDARD, "Failed to update service config %d\n", retval);
            goto done_verify;
        }
        WcaLog(LOGMSG_STANDARD, "Updated path for existing service");
    }
    {
        WcaLog(LOGMSG_STANDARD, "Resetting dependencies");
        BOOL bRet = ChangeServiceConfigW(hService, SERVICE_NO_CHANGE, SERVICE_NO_CHANGE, SERVICE_NO_CHANGE, NULL, NULL,
                                         NULL, _binaryPathName.c_str(), NULL, NULL, NULL);
        if (!bRet)
        {
            retval = GetLastError();
            WcaLog(LOGMSG_STANDARD, "Failed to update service dependency config %d\n", retval);
            goto done_verify;
        }
        WcaLog(LOGMSG_STANDARD, "Updated dependencies for existing service");
    }

done_verify:
    CloseServiceHandle(hService);

    return retval;
}

const std::wstring &ServiceDefinition::getServiceName() const
{
    return _svcName;
}
