# 田野语料转写质检与发布工作台

本项目为语言学田野团队提供录音片段建档、带时间码转写、自动质量检查、专家复核整改以及冻结发布凭据的一体化浏览器工作台。服务使用 Go 原生 HTTP 提供 `/workbench` 页面和同源 JSON API，数据默认保存在 `.fieldlingua-data`。

工作台会按项目状态、伦理状态和负责人加载项目摘要，并可继续查看最新项目详情。业务 API 包括项目摘要与详情、授权片段登记、修订提交与历史、问题汇总、专家复核队列、冻结发布及发布凭据核验。所有修改命令均支持 `expectedVersion` 乐观并发控制；建档、片段、修订、复核和发布命令支持持久化 `idempotencyKey`。

主要查询入口：

- `GET /api/projects?status=draft&ethicsStatus=approved&ownerID=...`
- `GET /api/projects/{projectID}`
- `GET /api/revisions?projectID=...&segmentID=...`
- `GET /api/issues?projectID=...&revisionID=...&code=...`
- `GET /api/reviews?projectID=...`
- `GET /api/credentials/{credentialID}` 与 `POST /api/credentials/verify`

标准构建、运行和测试：

```bash
go test ./...
go run ./cmd/fieldlingua -selfcheck -addr=127.0.0.1:19081
go run ./cmd/fieldlingua -addr=127.0.0.1:19081
```

也可以设置 `PORT` 环境变量指定回环端口。浏览器访问 `http://127.0.0.1:19081/workbench`。
