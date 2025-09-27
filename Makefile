proto:
	protoc --proto_path=proto --go_out=. --go-grpc_out=. .\proto\notification\notification.proto .\proto\order\order.proto .\proto\payment\payment.proto .\proto\product\product.proto .\proto\user\user.proto

.PHONY: proto