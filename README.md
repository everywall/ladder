<p align="center">
    <img src="public/pigeon.svg" width="100px">
</p>

<h1 align="center">Ladder</h1>

*Ladder is a web proxy to help bypass paywalls.* This is a selfhosted version of [1ft.io](https://1ft.io) and [12ft.io](https://12ft.io). It is inspired by [13ft](https://github.com/wasi-master/13ft).

### Why

Freedom of information is an essential pillar of democracy and informed decision-making. While media organizations have legitimate financial interests, it is crucial to strike a balance between profitability and the public's right to access information. The proliferation of paywalls raises concerns about the erosion of this fundamental freedom, and it is imperative for society to find innovative ways to preserve access to vital information without compromising the sustainability of journalism. In a world where knowledge should be shared and not commodified, paywalls should be critically examined to ensure that they do not undermine the principles of an open and informed society.

Certain sites may display missing images or encounter formatting issues. This can be attributed to the site's reliance on JavaScript or CSS for image and resource loading, which presents a limitation when accessed through this proxy. If you prefer a full experience, please concider buying a subscription for the site.

### Features
- [x] Bypass Paywalls
- [x] Remove CORS Headers from Responses, Assets, and Images ...
- [x] Keep Site browsable
- [x] Add a debug path
- [x] Add a API
- [x] Docker Container
- [x] Linux Binary
- [x] Mac OS Binary
- [x] Windows Binary (Untested)
- [ ] Basic Auth

## Installation

### Binary
1) Download Binary
2) Run Binary
3) Open Browser (Default: http://localhost:8080)

### Docker
```bash
docker run -p 8080:8080 -d --name ladder ghcr.io/kubero-dev/ladder:latest
```

### Docker Compose
```bash
wget https://raw.githubusercontent.com/kubero-dev/ladder/main/docker-compose.yml
docker-compose up -d
```

## Configuration

### Environment Variables

| Variable | Description | Default |
| --- | --- | --- |
| `PORT` | Port to listen on | `8080` |
| `PREFORK` | Spawn multiple server instances | `false` |
