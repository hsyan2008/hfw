package register

import (
	"context"
	"fmt"
	"time"

	etcd3 "github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/etcdserver/api/v3rpc/rpctypes"
	"github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/signal"
)

// Prefix should start and end with no slash
var Prefix = "etcd3_naming"

type EtcdRegister struct {
	target []string
	ttl    int

	client     *etcd3.Client
	serviceKey string
}

func NewEtcdRegister(target []string, ttl int) *EtcdRegister {
	return &EtcdRegister{target: target, ttl: ttl}
}

// Register
func (er *EtcdRegister) Register(info RegisterInfo) error {
	serviceValue := fmt.Sprintf("%s:%d", info.Host, info.Port)
	er.serviceKey = fmt.Sprintf("/%s/%s/%s", Prefix, info.ServiceName, serviceValue)
	//TODO
	logger.Warn(er.serviceKey)

	// get endpoints for register dial address
	var err error
	er.client, err = etcd3.New(etcd3.Config{
		Endpoints: er.target,
	})
	if err != nil {
		return fmt.Errorf("grpclb: create etcd3 client failed: %v", err)
	}

	go func() {
		// invoke self-register with ticker
		ticker := time.NewTicker(info.UpdateInterval)
		for {
			// minimum lease TTL is ttl-second
			resp, _ := er.client.Grant(signal.GetSignalContext().Ctx, int64(er.ttl))
			// should get first, if not exist, set it
			_, err := er.client.Get(signal.GetSignalContext().Ctx, er.serviceKey)
			if err != nil {
				if err == rpctypes.ErrKeyNotFound {
					if _, err := er.client.Put(signal.GetSignalContext().Ctx, er.serviceKey, serviceValue, etcd3.WithLease(resp.ID)); err != nil {
						logger.Warnf("grpclb: set service '%s' with ttl to etcd3 failed: %s", info.ServiceName, err.Error())
					}
				} else {
					logger.Warnf("grpclb: service '%s' connect to etcd3 failed: %s", info.ServiceName, err.Error())
				}
			} else {
				// refresh set to true for not notifying the watcher
				if _, err := er.client.Put(signal.GetSignalContext().Ctx, er.serviceKey, serviceValue, etcd3.WithLease(resp.ID)); err != nil {
					logger.Warnf("grpclb: refresh service '%s' with ttl to etcd3 failed: %s", info.ServiceName, err.Error())
				}
			}
			select {
			case <-signal.GetSignalContext().Ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()

	signal.GetSignalContext().WgAdd()

	return nil
}

// UnRegister delete registered service from etcd
func (er *EtcdRegister) UnRegister() error {
	defer signal.GetSignalContext().WgDone()

	var err error
	if _, err := er.client.Delete(context.Background(), er.serviceKey); err != nil {
		logger.Warnf("grpclb: deregister '%s' failed: %s", er.serviceKey, err.Error())
	} else {
		logger.Warnf("grpclb: deregister '%s' ok.", er.serviceKey)
	}
	return err
}
