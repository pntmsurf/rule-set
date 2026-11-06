<p align="center">
  <img src="assets/images/rule-set.png" alt="rule-set" width="100%">
</p>

<p align="center">
  <b>Мульти-региональный монорепозиторий правил маршрутизации для Xray-Core, Sing-Box и Mihomo</b>
</p>

## Архитектура

Репозиторий организован по принципу разделения глобальных атомарных/составных категорий (`data/shared/`) и региональных категорий (`data/{ru,cn,ir,us,gb}/`):

```text
data/
  shared/
    telegram.toml
    whatsapp.toml
    discord.toml
    messengers.toml
    social.toml
    neuro.toml
    claude.toml
    gemini.toml
    muse.toml
    ...
  ru/
    gov.toml
    neuro.toml
    movies.toml
    finance.toml
    manifest.yaml
  cn/
    censor.toml
    ads.toml
    gov.toml
    finance.toml
    markets.toml
    ...
    manifest.yaml
  ir/
    censor.toml
    ads.toml
    gov.toml
    finance.toml
    markets.toml
    arvancloud.toml
    ...
    manifest.yaml
  us/
    gov.toml
    movies.toml
    finance.toml
    manifest.yaml
  gb/
    gov.toml
    movies.toml
    finance.toml
    manifest.yaml
```

### Формат `.toml` категорий и `@`-ссылки

Каждая категория описывается в едином файле `.toml` с двумя секциями `[domains]` и `[cidr]`, а в комментариях указываются только прямые ссылки на источники `# <source-url>`:

```toml
[domains]
# https://github.com/itdoginfo/allow-domains/blob/main/Services/telegram.lst
+.t.me
+.telegram.org

[cidr]
# https://github.com/itdoginfo/allow-domains/blob/main/Subnets/IPv4/telegram.lst
149.154.160.0/20
# https://github.com/itdoginfo/allow-domains/blob/main/Subnets/IPv6/telegram.lst
2001:67c:4e8::/48
```

```toml
[domains]
@telegram
@fbmessenger
@whatsapp
@discord
@viber
@zoom
@skype

[cidr]
@telegram
@discord
```

### Региональные `manifest.yaml`

В каждом регионе (`data/<region>/manifest.yaml`) задаётся явный белый список `shared`-категорий, которые включаются в региональные `<region>-geosite.dat` и `<region>-geoip.dat` вместе с собственными категориями региона:

```yaml
shared:
  - telegram
  - muse
  - gemini
  - claude
```

## Подключение

### HAPP  🇷🇺
```json
{
  "Name": "Looney RU Routing",
  "GlobalProxy": "true",
  "RouteOrder": "block-proxy-direct",
  "RemoteDNSType": "DoH",
  "RemoteDNSDomain": "https://8.8.8.8/dns-query",
  "RemoteDNSIP": "8.8.8.8",
  "DomesticDNSType": "DoH",
  "DomesticDNSDomain": "https://77.88.8.8/dns-query",
  "DomesticDNSIP": "77.88.8.8",
  "Geoipurl": "https://github.com/pntmsurf/rule-set/releases/download/latest/ru-geoip.dat",
  "Geositeurl": "https://github.com/pntmsurf/rule-set/releases/download/latest/ru-geosite.dat",
  "LastUpdated": "1785305201",
  "DnsHosts": {
    "lkfl2.nalog.ru": "213.24.64.175",
    "lknpd.nalog.ru": "213.24.64.181"
  },
  "DirectSites": [
    "geosite:ru",
    "geosite:ru-boards",
    "geosite:ru-dev",
    "geosite:ru-finance",
    "geosite:ru-food",
    "geosite:ru-forums",
    "geosite:ru-games",
    "geosite:ru-gov",
    "geosite:ru-job",
    "geosite:ru-map",
    "geosite:ru-markets",
    "geosite:ru-massmedia",
    "geosite:ru-media",
    "geosite:ru-messengers",
    "geosite:ru-movies",
    "geosite:ru-music",
    "geosite:ru-neuro",
    "geosite:ru-porn",
    "geosite:ru-social",
    "geosite:ru-telecom",
    "geosite:ru-videos",
    "geosite:ru-yandex"
  ],
  "DirectIp": [
    "geoip:private"
  ],
  "ProxySites": [
    "geosite:claude",
    "geosite:gemini",
    "geosite:muse",
    "geosite:youtube",
    "geosite:messengers",
    "geosite:social"
  ],
  "ProxyIp": [
    "geoip:telegram",
    "geoip:discord",
    "geoip:meta",
    "geoip:x"
  ],
  "BlockSites": [
    "geosite:ads",
    "geosite:ru-ads"
  ],
  "BlockIp": [],
  "DomainStrategy": "IPIfNonMatch",
  "FakeDNS": "false",
  "UseChunkFiles": "false"
}
```
