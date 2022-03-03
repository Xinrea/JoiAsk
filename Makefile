version=1.0.0

build:
	docker build --platform linux/amd64 -t registry.cn-hongkong.aliyuncs.com/joi/jask:${version} .