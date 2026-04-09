# запуск и останов редис и докер через docker-compose.yml, только для user-service

docker-compose --env-file apps/user-service/.env -f deployments/user-service/docker-compose.yml up -d

docker-compose --env-file apps/user-service/.env -f deployments/user-service/docker-compose.yml down
