Wake-On-LAN daemon
------------------

Compile:

```
# current arch:
go build -ldflags "-s -w"
# ARMv7:
GOARCH=arm GOARM=7 go build -o go-wol-daemon-arm -ldflags "-s -w"
```

Start daemon:

```
Usage of ./go-wol-daemon:
  -port string
    	Port to listen on (default "3000")
  -token string
    	Token (default "02ec48d46e0a7ae83ed4")
```

Wake up:

```
curl -X POST http://127.0.0.1:3000/02ec48d46e0a7ae83ed4 -d '{"mac":"11:22:33:44:55:66"}'
```
