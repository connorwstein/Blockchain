protoc -I protos/ --go_out=plugins=grpc:protos protos/coin.proto
go build 
