name "snmp-traps"
default_version "0.1.0"

source :url => "https://s3.amazonaws.com/dd-agent-omnibus/snmp_traps_db/dd_traps_db-#{version}.json.gz",
       :sha256 => "69c3ea4e3898b6889f951a9a6768339635de06b5a9ff8be85825d6cbe08e28f0",
       :target_filename => "dd_traps_db.json.gz"


build do
  # The dir for confs
  if osx?
    traps_db_dir = "#{install_dir}/etc/conf.d/snmp.d/traps_db"
  else
    traps_db_dir = "#{install_dir}/etc/datadog-agent/conf.d/snmp.d/traps_db"
  end
  mkdir traps_db_dir
  copy "dd_traps_db.json.gz", "#{traps_db_dir}/dd_traps_db.json.gz"
end
