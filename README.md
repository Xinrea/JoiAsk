# JoiAsk 提问箱

## Configuration 配置

### OSS 存储图片

```
{
    "db": {
        "host": "192.168.50.58",
        "port": 3306,
        "user": "root",
        "pass": "test",
        "name": "jask"
    },
    "server": {
        "host": "0.0.0.0",
        "port": 8080
    },
    "storage_type": "oss",
    "oss":{
        "address": "https://i0.vjoi.cn",
        "endpoint":"oss-cn-beijing.aliyuncs.com",
        "access_key":"",
        "secret_key":"",
        "bucket":"jwebsite-storage"
    }
}
```

### 本地存储图片

```
{
    "db": {
        "host": "192.168.50.58",
        "port": 3306,
        "user": "root",
        "pass": "test",
        "name": "jask"
    },
    "server": {
        "host": "0.0.0.0",
        "port": 8080
    },
    "storage_type": "local",
}
```
