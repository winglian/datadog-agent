#
# Cookbook Name:: dd-agent-disable-system-repos
# Recipe:: default
#
# Copyright (C) 2021-present Datadog
#
# All rights reserved - Do Not Redistribute
#

# We completely disable all package repositories on RPM based
# platforms, so for now we only need to do this on DEB based
# NOTE: apt only supports mirrorlist on Debian >= 10 and Ubuntu >= 18
if (node[:platform] == 'debian' && node['lsb']['release'].to_f >= 10.0 ) ||
    (node[:platform] == 'ubuntu' && node['lsb']['release'].to_f >= 18.0 )
  # NOTE about APT mirrorlists:
  # It seems that this feature could use some improvement. If you just get mirrorlist
  # from mirrors.ubuntu.com/mirrors.txt, it might contain faulty mirrors that either
  # cause `apt update` to fail with exit code 100 or make it hang on `0% [Working]`
  # indefinitely. Therefore we create a mirrorlist with the 2 mirrors that we know
  # should be reliable enough in combination and also well maintained.
  template '/etc/apt/mirrorlist' do
    source "#{node[:platform]}-mirrorlist"
    mode '0644'
    owner 'root'
    group 'root'
  end

  puts node
  puts node[:cpu]
  puts node[:cpu][:architecture]

  template '/etc/apt/sources.list' do
    source "sourcelist"
    mode '0644'
    owner 'root'
    group 'root'
  end

  apt_update
end
