<h1 align="center" id="title">achan.moe</h1>
<img src="https://img.shields.io/liberapay/receives/sevensrequiem.svg?logo=liberapay"><img src="https://img.shields.io/github/commit-activity/w/sevensrequiem/achan.moe"><img src="https://img.shields.io/github/last-commit/sevensrequiem/achan.moe"><img src="https://img.shields.io/github/languages/top/sevensrequiem/achan.moe"><img src="https://img.shields.io/github/repo-size/sevensrequiem/achan.moe"><img src="https://img.shields.io/github/downloads/sevensrequiem/achan.moe/total">




Install achan.moe on ubuntu/debian

```bash
  adduser achan
  su achan
  git clone https://github.com/SevensRequiem/achan.moe.git
  cd achan.moe
  go build
  mkdir production && mv achan.moe /production && cp views /production && cp banners /production && cp assets /production && cd production && mkdir boards && cp ../.env .env
  cd .. && mv production /home/achan && chmod +x /home/achan/production/achan.moe

```
### Running
```./home/achan/production/achan.moe or install service with systemd```


### todo
- docker
- anonymous login system (**DONE**)
- plugins (*10%*)
- better admin panel (*20%*)
- realtime / websockets (*20%*)
- global + per board config files
- board janny / panel 
- ratelimits (**DONE**)
- update system
- json config system (*replace DB*)
- cache system (*20%?*)
- markdown support (**DONE**)
- ping check for minecraft / servers
- reply system

### contributing
feel free to contribute whenever, wherever.
