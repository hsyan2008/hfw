package serviceDiscovery

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hsyan2008/hfw"
)

type balancePolicy uint

const (
	UnknownPolicy balancePolicy = iota
	RobinPolicy
	RandPolicy
)

type ConsulResolver struct {
	client      *api.Client
	serviceName string
	tag         string

	addresses []string
	tags      []string

	httpCtx *hfw.HTTPContext

	wg *sync.WaitGroup

	queryOptions *api.QueryOptions

	policy  balancePolicy
	lbIndex uint64
}

var consulResolverMap = make(map[string]*ConsulResolver)
var consulRwLock = new(sync.RWMutex)

func NewConsulResolver(serviceName, address string, policy balancePolicy, tag string) (*ConsulResolver, error) {
	key := fmt.Sprintf("%s_%s_%d_%s", serviceName, address, policy, tag)
	consulRwLock.RLock()
	if cr, ok := consulResolverMap[key]; ok {
		consulRwLock.RUnlock()
		return cr, nil
	}
	consulRwLock.RUnlock()

	consulRwLock.Lock()
	defer consulRwLock.Unlock()

	if cr, ok := consulResolverMap[key]; ok {
		return cr, nil
	}

	httpCtx := hfw.NewHTTPContext()
	config := api.DefaultConfig()
	config.Address = address
	client, err := api.NewClient(config)
	if err != nil {
		httpCtx.Fatal("create consul client error", err.Error())
		httpCtx.Cancel()
		return nil, err
	}
	cr := &ConsulResolver{
		client:       client,
		serviceName:  serviceName,
		tag:          tag,
		wg:           new(sync.WaitGroup),
		httpCtx:      httpCtx,
		policy:       policy,
		queryOptions: (&api.QueryOptions{}).WithContext(httpCtx.Ctx),
	}

	err = cr.resolve()
	if err != nil {
		return nil, err
	}

	cr.wg.Add(1)
	go cr.watch()

	consulResolverMap[key] = cr

	httpCtx.Info("NewConsulResolver:", key)

	return cr, nil
}
func (consulResolver *ConsulResolver) resolve() (err error) {
	serviceEntries, metaInfo, err := consulResolver.client.Health().Service(consulResolver.serviceName, consulResolver.tag, true, consulResolver.queryOptions)
	if err != nil {
		if e, ok := err.(*url.Error); ok {
			if e.Err == context.Canceled {
				return nil
			}
		}
		return
	}

	consulResolver.queryOptions.WaitIndex = metaInfo.LastIndex

	var adds []string
	for _, serviceEntry := range serviceEntries {
		address := fmt.Sprintf("%s:%d", serviceEntry.Service.Address, serviceEntry.Service.Port)
		adds = append(adds, address)
		consulResolver.tags = serviceEntry.Service.Tags
	}

	consulResolver.addresses = adds

	return
}

func (consulResolver *ConsulResolver) watch() {
	defer consulResolver.wg.Done()

	for {
		err := consulResolver.resolve()
		if err != nil {
			consulResolver.httpCtx.Fatal("query service entries error:", err.Error())
		}

		select {
		case <-consulResolver.httpCtx.Ctx.Done():
			consulResolver.httpCtx.Cancel()
			return
		default:
		}
	}
}

func (consulResolver *ConsulResolver) Close() {
	consulResolver.httpCtx.Cancel()
	consulResolver.wg.Wait()
}

func (consulResolver *ConsulResolver) Addresses() []string {
	if consulResolver == nil {
		return nil
	}
	return consulResolver.addresses
}

func (consulResolver *ConsulResolver) GetAddress() (address string, err error) {

	if consulResolver == nil {
		return "", errors.New("consul not init")
	}

	addresses := consulResolver.Addresses()
	num := uint64(len(addresses))
	if num == 0 {
		return "", errors.New("addresses is nil")
	}
	if num == 1 {
		return addresses[0], nil
	}
	if consulResolver.policy == RandPolicy {
		//随机
		address = addresses[rand.New(rand.NewSource(time.Now().UnixNano())).Int63n(int64(num))]
	} else {
		//轮询
		address = addresses[consulResolver.lbIndex%num]
		atomic.AddUint64(&consulResolver.lbIndex, 1)
	}
	return
}

func (consulResolver *ConsulResolver) HasTag(tag string) bool {
	for _, v := range consulResolver.tags {
		if v == tag {
			return true
		}
	}
	return false
}
