module github.com/wumitech-com/mdcp_server_operator

go 1.24.3

require (
	github.com/wumitech-com/mdcp_common v0.5.7-0.20251020035753-1775de4ba687
	github.com/wumitech-com/mdcp_proto v0.2.1-0.20251020030426-d792b94cb1fb
	google.golang.org/grpc v1.72.2
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250512202823-5a2f75b736a9 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gorm.io/gorm v1.30.0 // indirect
)

replace github.com/wumitech-com/mdcp_common => ../mdcp_common

replace github.com/wumitech-com/mdcp_proto => ../mdcp_proto
