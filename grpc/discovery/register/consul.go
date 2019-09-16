// // register service
// cr := register.NewConsulRegister(fmt.Sprintf("%s:%d", host, consul_port), 15)
// cr.Register(RegisterInfo{
// 	Host:           host,
// 	Port:           port,
// 	ServiceName:    "HelloService",
// 	UpdateInterval: time.Second})
package register

import (
	"fmt"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/signal"
)

type ConsulRegister struct {
	target string
	ttl    int

	registerInfo RegisterInfo
}

func NewConsulRegister(target string, ttl int) *ConsulRegister {
	return &ConsulRegister{target: target, ttl: ttl}
}

func (cr *ConsulRegister) Register(info RegisterInfo) error {
	cr.registerInfo = info
	// initial consul client config
	config := consulapi.DefaultConfig()
	config.Address = cr.target
	client, err := consulapi.NewClient(config)
	if err != nil {
		return fmt.Errorf("create consul client error: %s", err.Error())
	}

	serviceId := generateServiceId(info.ServiceName, info.Host, info.Port)

	reg := &consulapi.AgentServiceRegistration{
		ID:      serviceId,
		Name:    info.ServiceName,
		Tags:    []string{info.ServiceName},
		Port:    info.Port,
		Address: info.Host,
	}

	if err = client.Agent().ServiceRegister(reg); err != nil {
		return fmt.Errorf("register service to consul error: %s", err.Error())
	}

	// initial register service check
	check := consulapi.AgentServiceCheck{TTL: fmt.Sprintf("%ds", cr.ttl), Status: consulapi.HealthPassing}
	err = client.Agent().CheckRegister(
		&consulapi.AgentCheckRegistration{
			ID:                serviceId,
			Name:              info.ServiceName,
			ServiceID:         serviceId,
			AgentServiceCheck: check})
	if err != nil {
		return fmt.Errorf("initial register service check to consul error: %s", err.Error())
	}

	go func() {
		ticker := time.NewTicker(time.Duration(info.UpdateInterval) * time.Second)
		for {
			select {
			case <-signal.GetSignalContext().Ctx.Done():
				cr.UnRegister()
				return
			case <-ticker.C:
				err = client.Agent().UpdateTTL(serviceId, "", check.Status)
				if err != nil {
					logger.Warn("update ttl of service error: ", err.Error())
				}
			}
		}
	}()

	signal.GetSignalContext().WgAdd()

	return nil
}

func (cr *ConsulRegister) UnRegister() error {
	defer signal.GetSignalContext().WgDone()

	serviceId := generateServiceId(cr.registerInfo.ServiceName, cr.registerInfo.Host, cr.registerInfo.Port)

	config := consulapi.DefaultConfig()
	config.Address = cr.target
	client, err := consulapi.NewClient(config)
	if err != nil {
		return fmt.Errorf("create consul client error: %s", err.Error())
	}

	err = client.Agent().ServiceDeregister(serviceId)
	if err != nil {
		return fmt.Errorf("deregister service error: %s", err.Error())
	}
	logger.Info("deregistered service from consul server.")

	err = client.Agent().CheckDeregister(serviceId)
	if err != nil {
		return fmt.Errorf("deregister check error: %s", err.Error())
	}

	return nil
}

func generateServiceId(name, host string, port int) string {
	return fmt.Sprintf("%s-%s-%d", name, host, port)
}
