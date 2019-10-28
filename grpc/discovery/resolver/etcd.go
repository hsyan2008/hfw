// +build etcd

package resolver

import (
	"fmt"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/configs"
	"github.com/hsyan2008/hfw/grpc/discovery/common"
	"github.com/hsyan2008/hfw/signal"
	"golang.org/x/net/context"
	"google.golang.org/grpc/resolver"
)

type etcdBuilder struct {
	rawAddr []string
	cc      resolver.ClientConn
	cli     *clientv3.Client

	schema string

	ctx context.Context
}

// NewEtcdResolver initialize an etcd client
func NewEtcdBuilder(schema string, etcdAddrs []string) resolver.Builder {
	return &etcdBuilder{rawAddr: etcdAddrs, schema: schema, ctx: signal.GetSignalContext().Ctx}
}

func (r *etcdBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOption) (resolver.Resolver, error) {
	var err error

	if r.cli == nil {
		r.cli, err = clientv3.New(clientv3.Config{
			Endpoints:   r.rawAddr,
			DialTimeout: 5 * time.Second,
		})
		if err != nil {
			return nil, err
		}
	}

	r.cc = cc

	go r.watch("/" + target.Scheme + "/" + target.Endpoint + "/")

	return r, nil
}

func (r etcdBuilder) Scheme() string {
	return r.schema
}

func (r etcdBuilder) ResolveNow(rn resolver.ResolveNowOption) {
	// log.Println("ResolveNow") // TODO check
}

// Close closes the resolver.
func (r etcdBuilder) Close() {
	// log.Println("Close")
}

func (r *etcdBuilder) watch(keyPrefix string) {
	var addrList []resolver.Address

	getResp, err := r.cli.Get(r.ctx, keyPrefix, clientv3.WithPrefix())
	if err != nil {
		logger.Warn(err)
	} else {
		for i := range getResp.Kvs {
			addrList = append(addrList, resolver.Address{Addr: strings.TrimPrefix(string(getResp.Kvs[i].Key), keyPrefix)})
		}
	}

	r.cc.NewAddress(addrList)

	rch := r.cli.Watch(r.ctx, keyPrefix, clientv3.WithPrefix())
	for n := range rch {
		for _, ev := range n.Events {
			addr := strings.TrimPrefix(string(ev.Kv.Key), keyPrefix)
			switch ev.Type {
			case mvccpb.PUT:
				if !exist(addrList, addr) {
					addrList = append(addrList, resolver.Address{Addr: addr})
					r.cc.NewAddress(addrList)
				}
			case mvccpb.DELETE:
				if s, ok := remove(addrList, addr); ok {
					addrList = s
					r.cc.NewAddress(addrList)
				}
			}
		}
	}
}

func exist(l []resolver.Address, addr string) bool {
	for i := range l {
		if l[i].Addr == addr {
			return true
		}
	}
	return false
}

func remove(s []resolver.Address, addr string) ([]resolver.Address, bool) {
	for i := range s {
		if s[i].Addr == addr {
			s[i] = s[len(s)-1]
			return s[:len(s)-1], true
		}
	}
	return nil, false
}

func init() {
	common.ResolverFuncMap[common.EtcdResolver] = GenerateAndRegisterEtcdResolver
}

func GenerateAndRegisterEtcdResolver(cc configs.GrpcConfig) (schema string, err error) {
	if len(cc.ResolverAddresses) < 1 {
		return "", fmt.Errorf("GrpcConfig has nil ResolverAddresses")
	}
	if cc.ResolverScheme == "" {
		cc.ResolverScheme = common.EtcdResolver
	}

	lock.RLock()
	if resolver.Get(cc.ResolverScheme) != nil {
		lock.RUnlock()
		return cc.ResolverScheme, nil
	}
	lock.RUnlock()

	lock.Lock()
	defer lock.Unlock()

	if resolver.Get(cc.ResolverScheme) != nil {
		return cc.ResolverScheme, nil
	}
	builder := NewEtcdBuilder(cc.ResolverScheme, cc.ResolverAddresses)
	resolver.Register(builder)
	schema = builder.Scheme()
	return
}
