# FRP Server Configuration (TOML format for v0.52+)
bindAddr = "{{getenv "FRPS_BIND_ADDRESS" "0.0.0.0"}}"
bindPort = 7000

# UDP port used for KCP protocol, it can be same with 'bindPort'
# If not set, KCP is disabled in frps
kcpBindPort = 7000

# Specify which address proxy will listen for, default value is same with bindAddr
# proxyBindAddr = "127.0.0.1"

# If you want to support virtual host, you must set the http port for listening (optional)
# Note: http port and https port can be same with bind_port
vhostHTTPPort = 80
vhostHTTPSPort = 443

# Response header timeout(seconds) for vhost http server, default is 60s
# vhostHTTPTimeout = 60

{{if env.Getenv "FRPS_DASHBOARD" }}
webServer.addr = "{{getenv "FRPS_DASHBOARD_ADDRESS" "0.0.0.0"}}"
webServer.port = 7500

# Dashboard user and passwd for basic auth protect, if not set, both default value is admin
webServer.user = "{{getenv "FRPS_DASHBOARD_USER" "frpsadmin"}}"
webServer.password = "{{getenv "FRPS_DASHBOARD_PASSWORD" "frpsadmin"}}"
{{end}}

{{if env.Getenv "FRPS_LOGFILE" }}
log.to = "{{getenv "FRPS_LOGFILE" "/var/log/frps.log"}}"

# trace, debug, info, warn, error
log.level = "{{getenv "FRPS_LOG_LEVEL" "warn"}}"

log.maxDays = {{getenv "FRPS_LOG_DAYS" "5"}}
{{end}}

auth.method = "token"
auth.token = "{{getenv "FRPS_AUTH_TOKEN" "abcdefghi"}}"

allowPorts = [
  { start = 30000, end = 30900 }
]

# pool_count in each proxy will change to max_pool_count if they exceed the maximum value
maxPoolCount = 5

maxPortsPerClient = {{getenv "FRPS_MAX_PORTS" "0"}}

subDomainHost = "{{getenv "FRPS_SUBDOMAIN_HOST" "frps.com"}}"

transport.tcpMux = {{getenv "FRPS_TCP_MUX" "true"}}

{{if env.Getenv "FRPS_PERSISTENT_PORTS" }}
[[httpPlugins]]
name = "port-manager"
addr = "127.0.0.1:9001"
path = "/ports"
ops = ["NewProxy"]
{{end}}

{{if env.Getenv "FRPS_LETSENCRYPT_EMAIL" }}
[[httpPlugins]]
name = "acme-manager"
addr = "127.0.0.1:9002"
path = "/acme"
ops = ["NewProxy"]
{{end}}

{{if env.Getenv "FRPS_LINK_NOTIFIER" }}
[[httpPlugins]]
name = "linknotifier"
addr = "127.0.0.1:9003"
path = "/notifier"
ops = ["NewProxy"]
{{end}}
