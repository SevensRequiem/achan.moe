<h1 align="center" id="title">achan.moe</h1>

<p align="center"><img src="https://socialify.git.ci/SevensRequiem/achan.moe/image?font=Inter&amp;forks=1&amp;issues=1&amp;language=1&amp;name=1&amp;owner=1&amp;pattern=Solid&amp;pulls=1&amp;stargazers=1&amp;theme=Auto" alt="project-image"></p>
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
- anonymous login system
- plugins
- better admin panel
- realtime / websockets
- global + per board config files
- board janny / panel
- ratelimits
- update system
