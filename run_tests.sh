
echo "========== 运行测试并生成覆盖率报告开始 =========="
### 运行测试并生成覆盖率报告
go test ./... -coverprofile=coverage.out  && go tool cover -html=coverage.out -o coverage.html
echo "========== 运行测试并生成覆盖率报告结束 =========="