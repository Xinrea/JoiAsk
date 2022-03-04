build:
	docker build --platform linux/amd64 -t registry.cn-hongkong.aliyuncs.com/joi/jask:latest .
	docker push registry.cn-hongkong.aliyuncs.com/joi/jask:latest