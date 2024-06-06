package memory

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/openchami/node-orchestrator/pkg/nodes"
	"github.com/openchami/node-orchestrator/pkg/xnames"
)

type InMemoryStorage struct {
	nodes      map[uuid.UUID]nodes.ComputeNode
	bmcEntries map[uuid.UUID]nodes.BMC
}

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		nodes:      make(map[uuid.UUID]nodes.ComputeNode),
		bmcEntries: make(map[uuid.UUID]nodes.BMC),
	}
}

func (s *InMemoryStorage) SaveComputeNode(nodeID uuid.UUID, node nodes.ComputeNode) error {
	s.nodes[nodeID] = node
	return nil
}

func (s *InMemoryStorage) GetComputeNode(nodeID uuid.UUID) (nodes.ComputeNode, error) {
	node, ok := s.nodes[nodeID]
	if !ok {
		return nodes.ComputeNode{}, fmt.Errorf("ComputeNode not found")
	}
	return node, nil
}

func (s *InMemoryStorage) UpdateComputeNode(nodeID uuid.UUID, node nodes.ComputeNode) error {
	_, ok := s.nodes[nodeID]
	if !ok {
		return fmt.Errorf("ComputeNode not found")
	}
	s.nodes[nodeID] = node
	return nil
}

func (s *InMemoryStorage) DeleteComputeNode(nodeID uuid.UUID) error {
	_, ok := s.nodes[nodeID]
	if !ok {
		return fmt.Errorf("ComputeNode not found")
	}
	delete(s.nodes, nodeID)
	return nil
}

func (s *InMemoryStorage) SaveBMC(bmcID uuid.UUID, bmc nodes.BMC) error {
	s.bmcEntries[bmcID] = bmc
	return nil
}

func (s *InMemoryStorage) GetBMC(bmcID uuid.UUID) (nodes.BMC, error) {
	bmc, ok := s.bmcEntries[bmcID]
	if !ok {
		return nodes.BMC{}, fmt.Errorf("BMC not found")
	}
	return bmc, nil
}

func (s *InMemoryStorage) UpdateBMC(bmcID uuid.UUID, bmc nodes.BMC) error {
	_, ok := s.bmcEntries[bmcID]
	if !ok {
		return fmt.Errorf("BMC not found")
	}
	s.bmcEntries[bmcID] = bmc
	return nil
}

func (s *InMemoryStorage) DeleteBMC(bmcID uuid.UUID) error {
	_, ok := s.bmcEntries[bmcID]
	if !ok {
		return fmt.Errorf("BMC not found")
	}
	delete(s.bmcEntries, bmcID)
	return nil
}

func (s *InMemoryStorage) LookupComputeNodeByXName(xname string) (nodes.ComputeNode, error) {
	for _, node := range s.nodes {
		if (node.XName == xnames.NodeXname{Value: xname}) {
			return node, nil
		}
	}
	return nodes.ComputeNode{}, fmt.Errorf("ComputeNode not found")
}

func (s *InMemoryStorage) SearchComputeNodes(xname, hostname, arch, bootMAC, bmcMAC string) ([]nodes.ComputeNode, error) {
	var nodes []nodes.ComputeNode
	for _, node := range s.nodes {
		if (xname == "" || node.XName.Value == xname) &&
			(hostname == "" || node.Hostname == hostname) &&
			(arch == "" || node.Architecture == arch) &&
			(bootMAC == "" || node.BootMac == bootMAC) {
			nodes = append(nodes, node)
		}
	}
	return nodes, nil
}

func (s *InMemoryStorage) LookupBMCByXName(xname string) (nodes.BMC, error) {
	for _, bmc := range s.bmcEntries {
		if bmc.XName.Value == xname {
			return bmc, nil
		}
	}
	return nodes.BMC{}, fmt.Errorf("BMC not found")
}

func (s *InMemoryStorage) LookupComputeNodeByMACAddress(mac string) (nodes.ComputeNode, error) {
	for _, node := range s.nodes {
		for _, iface := range node.NetworkInterfaces {
			if iface.MACAddress == mac {
				return node, nil
			}
		}
	}
	return nodes.ComputeNode{}, fmt.Errorf("ComputeNode not found")
}

func (s *InMemoryStorage) LookupBMCByMACAddress(mac string) (nodes.BMC, error) {
	for _, bmc := range s.bmcEntries {
		if bmc.MACAddress == mac {
			return bmc, nil
		}
	}
	return nodes.BMC{}, fmt.Errorf("BMC not found")
}
