package client

import (
	"sync"

	"github.com/hashicorp/consul/api"
)

var consulClientMap = make(map[string]*api.Client)
var consulClientRwLock = new(sync.RWMutex)

func NewConsulClient(address string) (*api.Client, error) {
	key := address
	consulClientRwLock.RLock()
	if cr, ok := consulClientMap[key]; ok {
		consulClientRwLock.RUnlock()
		return cr, nil
	}
	consulClientRwLock.RUnlock()

	consulClientRwLock.Lock()
	defer consulClientRwLock.Unlock()

	if cr, ok := consulClientMap[key]; ok {
		return cr, nil
	}

	config := api.DefaultConfig()
	config.Address = address
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}
	consulClientMap[key] = client

	return client, nil
}
