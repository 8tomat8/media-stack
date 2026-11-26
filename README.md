# media-stack

A stack of self-hosted media managers and streamer along with VPN. 

Stack include VPN, Radarr, Sonarr, Prowlarr, qBittorrent and Jellyfin.

## Requirements

- Docker version 24.0.5 and above
- Docker compose version v2.20.2 and above
- It may also work on some of lower versions, but its not tested.

## Install media stack

There are two ways this stack can be deployed.

1. With a VPN (Recommended)
2. Without a VPN

Before we deploy the stack, We must create docker network first

```bash
docker network create mynetwork
```

**Deploy the stack with VPN**

If VPN is enabled, qBittorrent and Prowlarr will be put behind VPN.

By default, NordVPN is configured in `docker-compose.yml` file. This can be updated to use ExpressVPN, SurfShark, Custom OpenVPN or Wireguard VPN. It uses OpenVPN type for all the providers. 

Check respective document of your VPN provider to generate OpenVPN username and password.

By default, VPN is disabled in `docker-compose.yml`. We just need comment and uncomment few lines in `docker-compose.yml` file to enable and use VPN. Go through the comment messages in `docker-compose.yml` file to update them accordingly. Its very well guided in the compose file itself.

Update the `docker-compose.yml` file as guided in comment messsage in same file and follow below commands to deploy the stack.

To deploy the stack with VPN (with nordvpn):

```bash
VPN_SERVICE_PROVIDER=nordvpn OPENVPN_USER=openvpn-username OPENVPN_PASSWORD=openvpn-password SERVER_COUNTRIES=Switzerland docker compose --profile vpn up -d

# docker compose -f docker-compose-nginx.yml up -d # OPTIONAL to use Nginx as reverse proxy
```

**Deploy the stack without VPN**

To deploy the stack with without VPN (highly discouraged), Run below command.

```bash
docker compose up -d
# docker compose -f docker-compose-nginx.yml up -d # OPTIONAL to use Nginx as reverse proxy
```

## Configure qBittorrent

- Open qBitTorrent at http://localhost:5080. Default username:password is admin:adminadmin
- Go to Tools --> Options --> WebUI --> Change password
- Run below commands on the server

```bash
docker exec -it qbittorrent bash # Get inside qBittorrent container

# Above command will get you inside qBittorrent interactive terminal, Run below command in qbt terminal
mkdir /downloads/movies /downloads/tvshows
chown 1000:1000 /downloads/movies /downloads/tvshows
```

## Configure Radarr

- Open Radarr at http://localhost:7878
- Settings --> Media Management --> Check mark "Movies deleted from disk are automatically unmonitored in Radarr" under File management section --> Save
- Settings --> Download clients --> Transmission --> Add Host (qbittorrent) and port (5080) --> Username and password --> Test --> Save **Note: If VPN is enabled, then qbittorrent is reachable on vpn's service name**
- Settings --> General --> Enable advance setting --> Select AUthentication and add username and password
- Indexer will get automatically added during configuration of Prowlarr. See 'Configure Prowlarr' section.

**Add a movie** (After Prowlarr is configured)

- Movies --> Search for a movie --> Add Root folder (/downloads) --> Quality profile --> Add movie
- Go to qBittorrent (http://localhost:5080) and see if movie is getting downloaded.

## Configure Jellyfin

- Open Jellyfin at http://localhost:8096
- Configure as it asks for first time.
- Add media library folder and choose /data/movies/

## Configure Prowlarr

- Open Prowlarr at http://localhost:9696
- Settings --> General --> Authentications --> Select Authentication and add username and password
- Add Indexers, Indexers --> Add Indexer --> Search for indexer --> Choose base URL --> Test and Save
- Add application, Settings --> Apps --> Add application --> Choose Sonarr or Radarr or any apps to link --> Prowlarr server (http://localhost:9696) --> Radarr server (http://localhost:7878) --> API Key --> Test and Save
- This will add indexers in respective apps automatically.

## Configure Nginx

- Get inside Nginx container
- `cd /etc/nginx/conf.d`
- Add proxies for all tools.

`docker cp nginx.conf nginx:/etc/nginx/conf.d/default.conf && docker exec -it nginx nginx -s reload`
- Close ports of other tools in firewall/security groups except port 80 and 443.


## Apply SSL in Nginx

- Open port 80 and 443.
- Get inside Nginx container and install certbot and certbot-nginx `apk add certbot certbot-nginx`
- Add URL in server block. e.g. `server_name  localhost armdev.navratangupta.in;` in /etc/nginx/conf.d/default.conf
- Run `certbot --nginx` and provide details asked.

## Radarr Nginx reverse proxy

- Settings --> General --> URL Base --> Add base (/radarr)
- Add below proxy in nginx configuration

```
location /radarr {
    proxy_pass http://radarr:7878;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection $http_connection;
  }
```

- Restart containers.

## Sonarr Nginx reverse proxy

- Settings --> General --> URL Base --> Add base (/sonarr)
- Add below proxy in nginx configuration

```
location /radarr {
    proxy_pass http://sonarr:8989;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection $http_connection;
  }
```

## Prowlarr Nginx reverse proxy

- Settings --> General --> URL Base --> Add base (/prowlarr)
- Add below proxy in nginx configuration

This may need to change configurations in indexers and base in URL.

```
location /prowlarr {
    proxy_pass http://prowlarr:9696; # Comment this line if VPN is enabled.
    # proxy_pass http://vpn:9696; # Uncomment this line if VPN is enabled.
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection $http_connection;
  }
```

- Restart containers.

**Note: If VPN is enabled, then Prowlarr is reachable on vpn's service name**

## qBittorrent Nginx proxy

```
location /qbt/ {
    proxy_pass         http://qbittorrent:5080/; # Comment this line if VPN is enabled.
    # proxy_pass         http://vpn:5080/; # Uncomment this line if VPN is enabled.
    proxy_http_version 1.1;

    proxy_set_header   Host               http://qbittorrent:5080; # Comment this line if VPN is enabled.
    # proxy_set_header   Host               http://vpn:5080; # Uncomment this line if VPN is enabled.
    proxy_set_header   X-Forwarded-Host   $http_host;
    proxy_set_header   X-Forwarded-For    $remote_addr;
    proxy_cookie_path  /                  "/; Secure";
}
```

**Note: If VPN is enabled, then qbittorrent is reachable on vpn's service name**

## Jellyfin Nginx proxy

- Add base URL, Admin Dashboard -> Networking -> Base URL (/jellyfin)
- Add below config in Ngix config

```
 location /jellyfin {
        return 302 $scheme://$host/jellyfin/;
    }

    location /jellyfin/ {

        proxy_pass http://jellyfin:8096/jellyfin/;

        proxy_pass_request_headers on;

        proxy_set_header Host $host;

        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Host $http_host;

        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $http_connection;

        # Disable buffering when the nginx proxy gets very resource heavy upon streaming
        proxy_buffering off;
    }
```

- Restart containers

## Monitoring

Separate compose project `monitoring` in [`monitoring/`](monitoring/) ([`docker-compose.yml`](monitoring/docker-compose.yml)). It never touches the root project or `gateway` and publishes no ports.

| service | purpose | mem limit |
|---|---|---|
| `victoriametrics` | metrics store ([`scrape.yml`](monitoring/victoriametrics/scrape.yml)), 1y retention | 512m |
| `node-exporter` | host metrics | 64m |
| `telegraf` | per-container metrics ([`telegraf.conf`](monitoring/telegraf/telegraf.conf)) | 128m |
| `qbittorrent-exporter` | qBittorrent WebUI API metrics | 128m |
| `grafana` | [dashboards](monitoring/grafana/dashboards/) + [alerting](monitoring/grafana/provisioning/) | 512m |
| `vectorchord` | Postgres for Streamystats | 1g |
| `streamystats-job-server` | Jellyfin sync | 1g |
| `streamystats-nextjs` | Streamystats UI | 512m |

- `https://grafana.ms.${DOMAIN}`: PocketID OIDC login, no local admin.
- Home dashboard: [`overview.json`](monitoring/grafana/dashboards/overview.json) (`Server overview`), a one-screen summary of host, containers, disks, network, VPN/qBittorrent and alerts.
- `https://streamystats.ms.${DOMAIN}`: owner only, PocketID edge login (`oidc-auth@file`) plus Streamystats' Jellyfin login. This is the Jellyfin usage UI (who watched what, per-user stats, popular content). First run: Jellyfin URL `http://vpn:8096` + a new Jellyfin API key.

### Environment

`docker compose` needs these variables (it refuses to start while one is missing):

- `DOMAIN`: same as the root stack.
- `TELEGRAM_BOT_TOKEN`: `@BotFather` → `/newbot`. `TELEGRAM_CHAT_ID`: press Start in the bot, then `curl -s "https://api.telegram.org/bot<token>/getUpdates" | jq '.result[].message.chat.id'`.
- `GF_AUTH_GENERIC_OAUTH_CLIENT_ID` / `GF_AUTH_GENERIC_OAUTH_CLIENT_SECRET`: PocketID client `Grafana`, callback `https://grafana.ms.${DOMAIN}/login/generic_oauth`, allowed user groups = owner only.
- `QBITTORRENT_API_KEY`: qBittorrent → Options → WebUI → API Key.
- `POSTGRES_PASSWORD`, `SESSION_SECRET`, `NEXT_SERVER_ACTIONS_ENCRYPTION_KEY`: random strings (`openssl rand -hex 24`, `openssl rand -base64 32`, `openssl rand -hex 16`).

### Deploy (on the NAS)

```bash
cd /home/8tomat8/media-stack
[ "$(stat -c %g /var/run/docker.sock)" = 121 ] && docker inspect -f '{{.Name}} {{.State.StartedAt}}' $(docker ps -q) | sort > /tmp/started-before.txt
git pull --ff-only
mkdir -p config/monitoring/{grafana,victoriametrics,vectorchord}
sudo cp monitoring/monitoring-compose.service /etc/systemd/system/
sudo systemctl daemon-reload && sudo systemctl enable --now monitoring-compose.service
```

Verify: the same `docker inspect` line into `/tmp/started-after.txt`; `diff` shows only the 8 new containers. Grafana → Alerting → Contact points → `telegram` → Test. Crash-loop test: `docker run -d --name crashtest --restart always alpine:3 sh -c 'sleep 5; exit 1'`, expect `ContainerCrashLoop` within ~5 min, then `docker rm -f crashtest`.

Rollback: `sudo systemctl disable --now monitoring-compose.service` (runs `docker compose -p monitoring down`; data stays in `config/monitoring/`).

### Alerts

Rules in [`alerting.yml`](monitoring/grafana/provisioning/alerting/alerting.yml), sent to Telegram.

| alert | fires when |
|---|---|
| `HostMemoryLow` | MemAvailable < 25% of RAM for 10m |
| `HostCpuSaturated` | CPU PSI > 0.5 for 15m |
| `ContainerCrashLoop` | a container restarted 3+ times in 15m |
| `ContainerNotRunning` | a compose container exited/dead/restarting/created for 5m |
| `ContainerUnhealthy` | healthcheck unhealthy for 5m |
| `ContainerMissing` | a compose container seen in the last day is gone for 5m |
| `DiskSpaceLow` | `/`, `/volume1` or `/volume2` < 20% free for 15m |
| `ScrapeTargetDown` | a scrape target down for 3m |
| `QbittorrentDown` | qBittorrent WebUI API not answering for 5m |

Tuning: edit `alerting.yml`, `git pull --ff-only` on the NAS, then `docker restart grafana`.

Optional, each needs a maintenance window:

- O1: Gluetun control server + Telegraf check of `vpn:8000/v1/portforward` + `VpnPortForwardMissing` alert; recreates `vpn` and all media apps (3–10 min down).
- O2: Traefik Prometheus metrics on a private entrypoint + scrape job; recreates `traefik` (~1 min, all HTTPS routes down).
