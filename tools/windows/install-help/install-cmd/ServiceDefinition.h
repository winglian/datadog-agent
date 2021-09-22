#pragma once

class serviceDefinition
{
  private:
    std::wstring svcName;
    std::wstring displayName;
    std::wstring displayDescription;
    DWORD access;
    DWORD serviceType;
    DWORD startType;
    DWORD dwErrorControl;
    std::wstring lpBinaryPathName;
    std::wstring lpLoadOrderGroup;
    LPDWORD lpdwTagId;
    std::wstring lpDependencies; // list of single-null-terminated strings, double null at end
    std::wstring lpServiceStartName;
    std::wstring lpPassword;

  public:
    serviceDefinition()
        : svcName()
        , displayName()
        , displayDescription()
        , access(SERVICE_ALL_ACCESS)
        , serviceType(SERVICE_WIN32_OWN_PROCESS)
        , startType(SERVICE_DEMAND_START)
        , dwErrorControl(SERVICE_ERROR_NORMAL)
        , lpBinaryPathName()
        , lpLoadOrderGroup()
        , // not needed
        lpdwTagId(NULL)
        , // no tag identifier
        lpDependencies()
        , // no dependencies to start
        lpServiceStartName()
        ,            // will set to LOCAL_SYSTEM by default
        lpPassword() // no password for LOCAL_SYSTEM
    {
    }

    serviceDefinition(const std::wstring &name)
        : svcName(name)
        , displayName()
        , displayDescription()
        , access(SERVICE_ALL_ACCESS)
        , serviceType(SERVICE_WIN32_OWN_PROCESS)
        , startType(SERVICE_DEMAND_START)
        , dwErrorControl(SERVICE_ERROR_NORMAL)
        , lpBinaryPathName()
        , lpLoadOrderGroup()
        , // not needed
        lpdwTagId(NULL)
        , // no tag identifier
        lpDependencies()
        , // no dependencies to start
        lpServiceStartName()
        ,            // will set to LOCAL_SYSTEM by default
        lpPassword() // no password for LOCAL_SYSTEM
    {
    }

    serviceDefinition(const std::wstring &name, const std::wstring &display, const std::wstring &desc,
               const std::wstring &path, const std::wstring &deps, DWORD st, const std::wstring &user,
               const std::wstring &pass)
        : svcName(name)
        , displayName(display)
        , displayDescription(desc)
        , access(SERVICE_ALL_ACCESS)
        , serviceType(SERVICE_WIN32_OWN_PROCESS)
        , startType(st)
        , dwErrorControl(SERVICE_ERROR_NORMAL)
        , lpBinaryPathName(path)
        , lpLoadOrderGroup(NULL)
        , // not needed
        lpdwTagId(NULL)
        , // no tag identifier
        lpDependencies(deps)
        , lpServiceStartName(user)
        , lpPassword(pass) // no password for LOCAL_SYSTEM
    {
    }

    DWORD create(SC_HANDLE hMgr)
    {
        DWORD retval = 0;
        WcaLog(LOGMSG_STANDARD, "serviceDef::create()");
        SC_HANDLE hService =
            CreateService(hMgr, svcName.c_str(), displayName.c_str(), access, serviceType, startType, dwErrorControl,
                          lpBinaryPathName.c_str(), lpLoadOrderGroup.c_str(), lpdwTagId, lpDependencies.c_str(),
                          lpServiceStartName.c_str(), lpPassword.c_str());
        if (!hService)
        {

            retval = GetLastError();
            WcaLog(LOGMSG_STANDARD, "Failed to CreateService %d", retval);
            return retval;
        }
        WcaLog(LOGMSG_STANDARD, "Created Service");
        if (this->startType == SERVICE_AUTO_START)
        {
            // make it delayed-auto-start
            SERVICE_DELAYED_AUTO_START_INFO inf = {TRUE};
            WcaLog(LOGMSG_STANDARD, "setting to delayed auto start");
            ChangeServiceConfig2(hService, SERVICE_CONFIG_DELAYED_AUTO_START_INFO, (LPVOID)&inf);
            WcaLog(LOGMSG_STANDARD, "done setting to delayed auto start");
        }
        // set the description
        if (!displayDescription.empty())
        {
            WcaLog(LOGMSG_STANDARD, "setting description");
            SERVICE_DESCRIPTION desc = {(LPWSTR)displayDescription.c_str()};
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

    DWORD destroy(SC_HANDLE hMgr)
    {
        SC_HANDLE hService = OpenService(hMgr, svcName.c_str(), DELETE);
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
    DWORD verify(SC_HANDLE hMgr)
    {
        SC_HANDLE hService = OpenService(hMgr, svcName.c_str(), SC_MANAGER_ALL_ACCESS);
        if (!hService)
        {
            return GetLastError();
        }
        DWORD retval = 0;
#define QUERY_BUF_SIZE 8192
        //////
        // from 6.11 to 6.12, the location of the service binary changed.  Check the location
        // vs the expected location, and change if it's different
        char *buf = new char[QUERY_BUF_SIZE];
        memset(buf, 0, QUERY_BUF_SIZE);
        QUERY_SERVICE_CONFIGW *cfg = (QUERY_SERVICE_CONFIGW *)buf;
        DWORD needed = 0;
        if (!QueryServiceConfigW(hService, cfg, QUERY_BUF_SIZE, &needed))
        {
            // shouldn't ever fail.  WE're supplying the largest possible buffer
            // according to the docs.
            retval = GetLastError();
            WcaLog(LOGMSG_STANDARD, "Failed to query service status %d\n", retval);
            goto done_verify;
        }
        if (_wcsicmp(cfg->lpBinaryPathName, lpBinaryPathName.c_str()) == 0)
        {
            // nothing to do, already correctly configured
            WcaLog(LOGMSG_STANDARD, "Service path already correct");
        }
        else
        {
            BOOL bRet = ChangeServiceConfigW(hService, SERVICE_NO_CHANGE, SERVICE_NO_CHANGE, SERVICE_NO_CHANGE,
                                             lpBinaryPathName.c_str(), NULL, NULL, NULL, NULL, NULL, NULL);
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
            BOOL bRet = ChangeServiceConfigW(hService, SERVICE_NO_CHANGE, SERVICE_NO_CHANGE, SERVICE_NO_CHANGE, NULL,
                                             NULL, NULL, lpDependencies.c_str(), NULL, NULL, NULL);
            if (!bRet)
            {
                retval = GetLastError();
                WcaLog(LOGMSG_STANDARD, "Failed to update service dependency config %d\n", retval);
                goto done_verify;
            }
            WcaLog(LOGMSG_STANDARD, "Updated dependencies for existing service, dependencies now %S",
                   lpDependencies.c_str());
        }

    done_verify:
        CloseServiceHandle(hService);
        if (buf)
        {
            delete[] buf;
        }

        return retval;
    }
    const wchar_t *getServiceName() const
    {
        return svcName.c_str();
    }
};
