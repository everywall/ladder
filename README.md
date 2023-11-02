<p align="center">
    <img src="public/pigeon.svg" width="100px">
</p>

<h1 align="center">Ladder</h1>

*Ladder is a web proxy to help bypass paywalls.*

### Why

Freedom of information is an essential pillar of democracy and informed decision-making. While media organizations have legitimate financial interests, it is crucial to strike a balance between profitability and the public's right to access information. The proliferation of paywalls raises concerns about the erosion of this fundamental freedom, and it is imperative for society to find innovative ways to preserve access to vital information without compromising the sustainability of journalism. In a world where knowledge should be shared and not commodified, paywalls should be critically examined to ensure that they do not undermine the principles of an open and informed society.

Some site might have missing images. This is due to the fact that the site using javascript or CSS to load the images. This is a limitation of this proxy. If you prefer a full experience, please concider buying a subscription for the site.

### Features
- [x] Bypass Paywalls
- [x] Remove CORS
- [x] Keep Site browsable
- [x] Docker Container
- [x] Linux Binary
- [x] Mac OS Binary
- [x] Windows Binary (Untested)


## How to use

1) Download Binary
2) Run Binary
3) Open Browser (Default: http://localhost:2000)
4) Enter URL
5) Hit Enter

## Configuration

### Environment Variables

| Variable | Description | Default |
| --- | --- | --- |
| `PORT` | Port to listen on | `2000` |
| `PREFORK` | Spawn multiple server instances | `false` |
