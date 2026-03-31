### для генерации proto файлов для user_service и common (так как user_service зависит от common):

### запускается из папки ./api (там, где лежит go.mod)

protoc --proto_path=. --plugin=protoc-gen-go="C:\Son_Alex\GO_projects\go\bin\protoc-gen-go.exe" --plugin=protoc-gen-go-grpc="C:\Son_Alex\GO_projects\go\bin\protoc-gen-go-grpc.exe" --go_out=. --go_opt=module=api --go-grpc_out=. --go-grpc_opt=module=api proto/user/v1/user_service.proto proto/common/v1/types.proto
