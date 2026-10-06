
.PHONY:

help:
	@echo "Docker 命令"
	@echo ""
	@echo " make <command>"
	@echo ""
	@echo "命令:"
	@echo "  docker-exec  - 进入开发容器"
	@echo "  docker-up  - 启动开发容器"

docker-up:
	docker compose up -d

# 进入开发容器
docker-exec:
	docker exec -it golang bash


