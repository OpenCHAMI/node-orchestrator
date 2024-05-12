package main

import (
	"fmt"

	"github.com/google/uuid"
)

type InMemoryStorage struct {
	nodes      map[uuid.UUID]ComputeNode
	bmcEntries map[uuid.UUID]BMC
}

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		nodes:      make(map[uuid.UUID]ComputeNode),
		bmcEntries: make(map[uuid.UUID]BMC),
	}
}

func (s *InMemoryStorage) SaveComputeNode(nodeID uuid.UUID, node ComputeNode) error {
	s.nodes[nodeID] = node
	return nil
}

func (s *InMemoryStorage) GetComputeNode(nodeID uuid.UUID) (ComputeNode, error) {
	node, ok := s.nodes[nodeID]
	if !ok {
		return ComputeNode{}, fmt.Errorf("ComputeNode not found")
	}
	return node, nil
}

func (s *InMemoryStorage) UpdateComputeNode(nodeID uuid.UUID, node ComputeNode) error {
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

func (s *InMemoryStorage) SaveBMC(bmcID uuid.UUID, bmc BMC) error {
	s.bmcEntries[bmcID] = bmc
	return nil
}

func (s *InMemoryStorage) GetBMC(bmcID uuid.UUID) (BMC, error) {
	bmc, ok := s.bmcEntries[bmcID]
	if !ok {
		return BMC{}, fmt.Errorf("BMC not found")
	}
	return bmc, nil
}

func (s *InMemoryStorage) UpdateBMC(bmcID uuid.UUID, bmc BMC) error {
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

func (s *InMemoryStorage) LookupComputeNodeByXName(xname string) (ComputeNode, error) {
	for _, node := range s.nodes {
		if (node.XName == NodeXname{Value: xname}) {
			return node, nil
		}
	}
	return ComputeNode{}, fmt.Errorf("ComputeNode not found")
}

func (s *InMemoryStorage) LookupBMCByXName(xname string) (BMC, error) {
	for _, bmc := range s.bmcEntries {
		if bmc.XName == xname {
			return bmc, nil
		}
	}
	return BMC{}, fmt.Errorf("BMC not found")
}

func (s *InMemoryStorage) LookupComputeNodeByMACAddress(mac string) (ComputeNode, error) {
	for _, node := range s.nodes {
		for _, iface := range node.NetworkInterfaces {
			if iface.MACAddress == mac {
				return node, nil
			}
		}
	}
	return ComputeNode{}, fmt.Errorf("ComputeNode not found")
}

func (s *InMemoryStorage) LookupBMCByMACAddress(mac string) (BMC, error) {
	for _, bmc := range s.bmcEntries {
		if bmc.MACAddress == mac {
			return bmc, nil
		}
	}
	return BMC{}, fmt.Errorf("BMC not found")
}
