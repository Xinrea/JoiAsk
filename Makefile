build:
	docker build --platform linux/amd64 -t registry.cn-beijing.aliyuncs.com/joi/jask:latest .
	docker push registry.cn-beijing.aliyuncs.com/joi/jask:latest