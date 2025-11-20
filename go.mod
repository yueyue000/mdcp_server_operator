module github.com/wumitech-com/mdcp_server_operator

go 1.24.3

require (
	github.com/wumitech-com/mdcp_common v0.5.7-0.20251020035753-1775de4ba687
	github.com/wumitech-com/mdcp_proto v0.2.1-0.20251020030426-d792b94cb1fb
	google.golang.org/grpc v1.72.2
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/wumitech-com/mdcp_common => ../mdcp_common
replace github.com/wumitech-com/mdcp_proto => ../mdcp_proto

