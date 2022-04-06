package glitterboot

import (
	"fmt"
	"net"
	"strings"

	"github.com/pkg/errors"
)

type NodeAddr struct {
	Address string
	Host    string
	Port    string
}

func (n *NodeAddr) String() string {
	return fmt.Sprintf("%s@%s:%s", n.Address, n.Host, n.Port)
}

func parseNodeAddr(idHostPort string) (*NodeAddr, error) {
	v := strings.Split(idHostPort, "@")
	if len(v) != 2 {
		return nil, errors.Errorf("invalid node IdHostPort: %s", idHostPort)
	}
	n := &NodeAddr{
		Address: v[0],
	}
	host, port, err := net.SplitHostPort(v[1])
	if err != nil {
		return nil, errors.Errorf("invalid node IdHostPort: %s err=%v", idHostPort, err)
	}
	n.Host = host
	n.Port = port
	return n, nil
}
