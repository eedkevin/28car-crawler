---
# Updates
## Run within docker swarm
 - install docker with version ^1.13
```
brew cask install docker
```
 - install docker-compose
```
brew install docker-compose
```
 - build docker image
```
docker-compose -f docker-build.yml build
```
 - deploy docker swarm
```
docker stack deploy -c docker-stack.yml 28car-crawler
```
 - optional, you may need to init docker swarm at the first time
```
docker swarm init
```

---
## Run without docker
### Preparation
 - golang and glide
```
brew install go glide
```    
 - redis
```
brew install redis
```
 - mongodb
```
brew install mongodb
```

### Download the source code under your GOPATH/src
```
cd $GOPATH/src
git clone <url-of-this-repo>
```
 
### Build
 - install dependencies
```
cd <project-path>
glide install
```
    
 - build go binary
```
glide rebuild
```

### Requisites
 - ensure redis is running on localhost
 - ensure mongodb is running on localhost
  
### Run
 - run with customize parameters
```
./28car-crawler --redis <redis-host:port> --mongo <mongo-host:port> --mode <'master'|'worker'>
```
 - Or directly run with default parameters if you leave your redis and mongodb as default setting
```
./28car-crawler
```