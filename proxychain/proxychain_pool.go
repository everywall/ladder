package proxychain

import (
	"net/url"
)

type Pool map[url.URL]ProxyChain

func NewPool() Pool {
	return map[url.URL]ProxyChain{}
}
