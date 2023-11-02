<p align="center">
    <img src="assets/pigeon.svg" width="100px">
</p>

<h1 align="center">Ladder</h1>

*Ladder is a web proxy to help bypass paywalls.* This is a selfhosted version of [1ft.io](https://1ft.io) and [12ft.io](https://12ft.io). It is inspired by [13ft](https://github.com/wasi-master/13ft).

### Why

Freedom of information is an essential pillar of democracy and informed decision-making. While media organizations have legitimate financial interests, it is crucial to strike a balance between profitability and the public's right to access information. The proliferation of paywalls raises concerns about the erosion of this fundamental freedom, and it is imperative for society to find innovative ways to preserve access to vital information without compromising the sustainability of journalism. In a world where knowledge should be shared and not commodified, paywalls should be critically examined to ensure that they do not undermine the principles of an open and informed society.

Certain sites may display missing images or encounter formatting issues. This can be attributed to the site's reliance on JavaScript or CSS for image and resource loading, which presents a limitation when accessed through this proxy. If you prefer a full experience, please concider buying a subscription for the site.

### Features
- [x] Bypass Paywalls
- [x] Remove CORS headers from responses, assets, and images ...
- [x] Keep site browsable
- [x] Add a raw path
- [x] Add a API
- [x] Docker container
- [x] Linux binary
- [x] Mac OS binary
- [x] Windows binary (untested)
- [x] Remove most of the ads (unexpected side effect)
- [x] Basic Auth
- [x] Disable Logs

## Installation

### Binary
1) Download binary [here](https://github.com/kubero-dev/ladder/releases/latest)
2) Unpack and run the binary `./ladder`
3) Open Browser (Default: http://localhost:8080)

### Docker
```bash
docker run -p 8080:8080 -d --name ladder ghcr.io/kubero-dev/ladder:latest
```

### Docker Compose
```bash
wget https://raw.githubusercontent.com/kubero-dev/ladder/main/docker-compose.yaml
docker-compose up -d
```

## Usage

### Browser
1) Open Browser (Default: http://localhost:8080)
2) Enter URL
3) Press Enter

Or direct by appending the URL to the end of the proxy URL:
http://localhost:8080/https://www.google.com

### API
```bash
curl -X GET "http://localhost:8080/api/https://www.google.com"
```

### RAW
http://localhost:8080/raw/https://www.google.com

## Configuration

### Environment Variables

| Variable | Description | Default |
| --- | --- | --- |
| `PORT` | Port to listen on | `8080` |
| `PREFORK` | Spawn multiple server instances | `false` |
| `USER_AGENT` | User agent to emulate | `Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)` |
| `X_FORWARDED_FOR` | IP forwarder address | `66.249.66.1` |
| `USERPASS` | Enables Basic Auth, format `admin:123456` | `` |
| `LOG_URLS` | Log fetched URL's | `true` |
