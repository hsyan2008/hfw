// // register service
// cr := register.NewConsulRegister(fmt.Sprintf("%s:%d", host, consul_port), 15)
// cr.Register(discovery.RegisterInfo{
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
	"github.com/hsyan2008/hfw2/grpc/discovery"
	"github.com/hsyan2008/hfw2/signal"
)

type ConsulRegister struct {
	Target string
	Ttl    int
}

func NewConsulRegister(target string, ttl int) *ConsulRegister {
	return &ConsulRegister{Target: target, Ttl: ttl}
}

func (cr *ConsulRegister) Register(info discovery.RegisterInfo) error {
	// initial consul client config
	config := consulapi.DefaultConfig()
	config.Address = cr.Target
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
	check := consulapi.AgentServiceCheck{TTL: fmt.Sprintf("%ds", cr.Ttl), Status: consulapi.HealthPassing}
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
		ticker := time.NewTicker(info.UpdateInterval)
		for {
			select {
			case <-signal.GetSignalContext().Ctx.Done():
				cr.DeRegister(info)
			case <-ticker.C:
				err = client.Agent().UpdateTTL(serviceId, "", check.Status)
				if err != nil {
					logger.Warn("update ttl of service error: ", err.Error())
				}
			}
		}
	}()

	return nil
}

func (cr *ConsulRegister) DeRegister(info discovery.RegisterInfo) error {

	serviceId := generateServiceId(info.ServiceName, info.Host, info.Port)

	config := consulapi.DefaultConfig()
	config.Address = cr.Target
	client, err := consulapi.NewClient(config)
	if err != nil {
		return fmt.Errorf("create consul client error: %s", err.Error())
	}

	err = client.Agent().ServiceDeregister(serviceId)
	if err != nil {
		return fmt.Errorf("deregister service error: %s", err.Error())
	} else {
		logger.Info("deregistered service from consul server.")
	}

	err = client.Agent().CheckDeregister(serviceId)
	if err != nil {
		return fmt.Errorf("deregister check error: %s", err.Error())
	}

	return nil
}

func generateServiceId(name, host string, port int) string {
	return fmt.Sprintf("%s-%s-%d", name, host, port)
}
