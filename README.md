
# achan.moe

A simple imageboard written in golang.




## Installation

Install achan.moe on ubuntu/debian

```bash
  adduser achan
  git clone https://github.com/SevensRequiem/achan.moe.git
  cd achan.moe
  go build
  mkdir production && mv achan.moe /production && cp views /production && cp banners /production && cp assets /production && cd production && mkdir boards && cp ../.env .env
  cd .. && mv production /home/achan && chmod +x /home/achan/production/achan.moe

```
### Running
```./home/achan/production/achan.moe or install service with systemd```
    

