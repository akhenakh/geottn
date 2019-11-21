//go:generate protoc --proto_path=..:. -I .. --go_out=plugins=grpc:. geottnsvc.proto

package geottnsvc
