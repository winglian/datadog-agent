#pragma once
#include <string>
#include <vector>

class ServiceDefinition
{
  public:
    typedef std::vector<std::wstring> deps_t;
  private:
    std::wstring _svcName;
    std::wstring _displayName;
    std::wstring _displayDescription;
    DWORD _access;
    DWORD _serviceType;
    DWORD _startType;
    DWORD _errorControl;
    std::wstring _binaryPathName;
    std::wstring _loadOrderGroup;
    deps_t _dependencies;
    std::wstring _serviceUsername;
    std::wstring _serviceUserPassword;

  public:
    ServiceDefinition();
    ServiceDefinition(const std::wstring &name);
    ServiceDefinition(const std::wstring &name, const std::wstring &display, const std::wstring &desc,
                      const std::wstring &path, DWORD st, const std::wstring &user,
                      const std::wstring &pass);
    void addDependency(const std::wstring &serviceName);
    void addDependency(ServiceDefinition const &serviceDef);
    DWORD create(SC_HANDLE hMgr);
    DWORD destroy(SC_HANDLE hMgr);
    DWORD verify(SC_HANDLE hMgr);
    const std::wstring &getServiceName() const;
};
