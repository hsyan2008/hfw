// // register service
// cr := register.NewConsulRegister(fmt.Sprintf("%s:%d", host, consul_port), 15)
// cr.Register(common.RegisterInfo{
// 	Host:           host,
// 	Port:           port,
// 	ServerName:     "HelloService",
// 	UpdateInterval: time.Second})
package register

import (
	"context"
	"fmt"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hsyan2008/go-logger"
	utils "github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/grpc/discovery/common"
	dc "github.com/hsyan2008/hfw/service_discovery/client"
	"github.com/hsyan2008/hfw/signal"
)

type ConsulRegister struct {
	target string
	ttl    int

	client *consulapi.Client

	ctx    context.Context
	cancel context.CancelFunc

	registerInfo common.RegisterInfo

	serviceID string
}

var _ common.Register = &ConsulRegister{}

func init() {
	common.RegisterFuncMap[common.ConsulResolver] = NewConsulRegister
}

func NewConsulRegister(target []string, ttl int) common.Register {
	cr := &ConsulRegister{target: target[0], ttl: ttl}
	return cr
}

func (cr *ConsulRegister) Register(info common.RegisterInfo) (err error) {
	cr.ctx, cr.cancel = context.WithCancel(signal.GetSignalContext().Ctx)
	cr.registerInfo = info

	cr.client, err = dc.NewConsulClient(cr.target)
	if err != nil {
		return fmt.Errorf("create consul client error: %s", err.Error())
	}

	cr.serviceID = generateServiceId(info.ServerName, info.Host, info.Port)

	reg := &consulapi.AgentServiceRegistration{
		ID:      cr.serviceID,
		Name:    info.ServerName,
		Tags:    info.Tags,
		Port:    info.Port,
		Address: info.Host,
	}

	if err = cr.client.Agent().ServiceRegister(reg); err != nil {
		return fmt.Errorf("register service to consul error: %s", err.Error())
	}

	// initial register service check
	check := consulapi.AgentServiceCheck{TTL: fmt.Sprintf("%ds", cr.ttl), Status: consulapi.HealthPassing}
	err = cr.client.Agent().CheckRegister(
		&consulapi.AgentCheckRegistration{
			ID:                cr.serviceID,
			Name:              info.ServerName,
			ServiceID:         cr.serviceID,
			AgentServiceCheck: check})
	if err != nil {
		return fmt.Errorf("initial register service check to consul error: %s", err.Error())
	}

	go func() {
		ticker := time.NewTicker(time.Duration(info.UpdateInterval) * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-signal.GetSignalContext().Ctx.Done():
				cr.cancel()
				return
			case <-cr.ctx.Done():
				return
			case <-ticker.C:
				err = cr.client.Agent().UpdateTTL(cr.serviceID, "", check.Status)
				if err != nil {
					logger.Warn("update ttl of service error: ", err.Error())
				}
			}
		}
	}()

	signal.GetSignalContext().WgAdd()

	return nil
}

func (cr *ConsulRegister) UnRegister() (err error) {
	defer func() {
		signal.GetSignalContext().WgDone()
		cr.cancel()
	}()

	err = cr.client.Agent().ServiceDeregister(cr.serviceID)
	if err != nil {
		return fmt.Errorf("deregister service error: %s", err.Error())
	}
	logger.Infof("deregistered service: %s from consul server.", cr.registerInfo.ServerName)

	err = cr.client.Agent().CheckDeregister(cr.serviceID)
	if err != nil {
		return fmt.Errorf("deregister check error: %s", err.Error())
	}

	return nil
}

func generateServiceId(name, host string, port int) string {
	//docker里，多个服务映射到同个ip和端口
	return fmt.Sprintf("%s-%d-%s", host, port, utils.GetHostName())
}
