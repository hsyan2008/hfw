package resolver

import (
	"context"
	"fmt"
	"sync"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/configs"
	"github.com/hsyan2008/hfw/grpc/discovery/common"
	"github.com/hsyan2008/hfw/signal"
	"google.golang.org/grpc/resolver"
)

type consulBuilder struct {
	scheme      string
	address     string
	client      *consulapi.Client
	serviceName string
	tag         string

	lastIndex uint64
}

func NewConsulBuilder(scheme, address, tag string) resolver.Builder {
	config := consulapi.DefaultConfig()
	config.Address = address
	client, err := consulapi.NewClient(config)
	if err != nil {
		logger.Fatal("create consul client error", err.Error())
		return nil
	}
	return &consulBuilder{scheme: scheme, address: address, tag: tag, client: client}
}

func (cb *consulBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	cb.serviceName = target.Endpoint

	adds, serviceConfig, err := cb.resolve()
	if err != nil {
		return nil, err
	}
	cc.NewAddress(adds)
	cc.NewServiceConfig(serviceConfig)

	consulResolver := NewConsulResolver(&cc, cb, opts)
	consulResolver.wg.Add(1)
	go consulResolver.watcher()

	return consulResolver, nil
}

func (cb *consulBuilder) resolve() ([]resolver.Address, string, error) {

	serviceEntries, metainfo, err := cb.client.Health().Service(cb.serviceName, cb.tag, true, &consulapi.QueryOptions{
		WaitIndex: cb.lastIndex, // 同步点，这个调用将一直阻塞，直到有新的更新
	})
	if err != nil {
		return nil, "", err
	}

	cb.lastIndex = metainfo.LastIndex

	adds := make([]resolver.Address, 0)
	for _, serviceEntry := range serviceEntries {
		address := resolver.Address{Addr: fmt.Sprintf("%s:%d", serviceEntry.Service.Address, serviceEntry.Service.Port)}
		adds = append(adds, address)
	}
	return adds, "", nil
}

func (cb *consulBuilder) Scheme() string {
	return cb.scheme
}

type consulResolver struct {
	clientConn    *resolver.ClientConn
	consulBuilder *consulBuilder
	// t                    *time.Ticker
	wg                   sync.WaitGroup
	rn                   chan struct{}
	ctx                  context.Context
	cancel               context.CancelFunc
	disableServiceConfig bool
}

func NewConsulResolver(cc *resolver.ClientConn, cb *consulBuilder, opts resolver.BuildOptions) *consulResolver {
	ctx, cancel := context.WithCancel(signal.GetSignalContext().Ctx)
	return &consulResolver{
		clientConn:    cc,
		consulBuilder: cb,
		// t:                    time.NewTicker(time.Second * 3),
		ctx:                  ctx,
		cancel:               cancel,
		disableServiceConfig: opts.DisableServiceConfig}
}

func (cr *consulResolver) watcher() {
	cr.wg.Done()
	for {
		select {
		case <-cr.ctx.Done():
			return
		case <-cr.rn:
		// case <-cr.t.C:
		default:
		}
		adds, serviceConfig, err := cr.consulBuilder.resolve()
		if err != nil {
			logger.Fatal("query service entries error:", err.Error())
			continue
		}
		(*cr.clientConn).NewAddress(adds)
		(*cr.clientConn).NewServiceConfig(serviceConfig)
	}
}

func (cr *consulResolver) Scheme() string {
	return cr.consulBuilder.Scheme()
}

func (cr *consulResolver) ResolveNow(rno resolver.ResolveNowOptions) {
	select {
	case cr.rn <- struct{}{}:
	default:
	}
}

func (cr *consulResolver) Close() {
	cr.cancel()
	cr.wg.Wait()
	// cr.t.Stop()
}

func init() {
	common.ResolverFuncMap[common.ConsulResolver] = GenerateAndRegisterConsulResolver
}

func GenerateAndRegisterConsulResolver(cc configs.GrpcConfig) (schema string, err error) {
	if len(cc.ResolverAddresses) == 0 {
		return "", fmt.Errorf("GrpcConfig has nil ResolverAddresses")
	}
	if cc.ResolverScheme == "" {
		cc.ResolverScheme = fmt.Sprintf("%s_%s", cc.ResolverType, cc.ServerName)
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
	builder := NewConsulBuilder(cc.ResolverScheme, cc.ResolverAddresses[0], cc.Tag)
	resolver.Register(builder)
	schema = builder.Scheme()
	return
}
