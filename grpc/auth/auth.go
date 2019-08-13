package auth

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

type Auth struct {
	isHTTPS bool
	value   string
}

func NewAuthWithHTTPS(value string) *Auth {
	return &Auth{
		isHTTPS: true,
		value:   value,
	}
}

func NewAuth(value string) *Auth {
	return &Auth{
		isHTTPS: false,
		value:   value,
	}
}

func (this *Auth) getKey() string {
	//固定为x
	return "x"
}

func (this *Auth) getValue() string {
	return this.value
}

func (this *Auth) GetRequestMetadata(context.Context, ...string) (
	map[string]string, error,
) {
	return map[string]string{this.getKey(): this.getValue()}, nil
}

func (this *Auth) RequireTransportSecurity() bool {
	//如果没有证书，则返回false
	return this.isHTTPS
}

func (this *Auth) Auth(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return fmt.Errorf("missing credentials")
	}

	var x string
	if val, ok := md[this.getKey()]; ok {
		x = val[0]
	} else {
		return grpc.Errorf(codes.Unauthenticated, "not found token")
	}

	if x != this.getValue() {
		return grpc.Errorf(codes.Unauthenticated, "invalid token")
	}

	return nil
}
