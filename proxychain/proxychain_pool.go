package proxychain

import (
	"net/url"
)

type ProxyChainPool map[url.URL]ProxyChain

func NewProxyChainPool() ProxyChainPool {
	return map[url.URL]ProxyChain{}
}
