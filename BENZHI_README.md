# CatalogSvc 数据目录与血缘服务

CatalogSvc 是通用数据目录基础设施：注册数据表与字段，管理元数据版本（草稿、
发布、回滚），记录表间血缘依赖并做环检测，向订阅方推送目录变更通知，并提供
一个浏览器目录浏览页面。

## 构建

```bash
docker build --platform linux/amd64 -f benzhi.Dockerfile -t catalogsvc .
docker build --platform linux/arm64 -f benzhi.Dockerfile -t catalogsvc:arm64 .
```

也可以直接使用 build 脚本：

```bash
./build_benzhi_docker.sh catalogsvc linux/amd64
```

## 运行

```bash
docker run --rm -p 8080:8080 catalogsvc bash -c "cd /app && go run ./cmd/catalogsvc -addr :8080"
```

启动后：

- `GET /` 返回目录浏览页面（web/browse.html）。
- `GET /api/health` 返回服务健康状态。
- `POST /api/tables` 注册数据表。
- `GET /api/tables` 列出全部表。
- `POST /api/lineage/edges` 登记血缘依赖。
- `GET /api/lineage/upstream/{id}` 遍历上游依赖。

## 容器内验证

```bash
go build ./...
go test ./...
go vet ./...
```

项目使用离线 vendor 目录，`GOPROXY=off` 下不依赖外网即可构建。
