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
./28car-crawler --redis-host <redis-host:port> --mongo-host <mongo-host:port>
```
 - Or directly run with default parameters if you leave your redis and mongodb as default setting
```
./28car-crawler
```