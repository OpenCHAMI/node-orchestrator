package storage

import (
	"github.com/google/uuid"
	"github.com/openchami/node-orchestrator/pkg/nodes"
)

type Storage interface {
	SaveComputeNode(nodeID uuid.UUID, node nodes.ComputeNode) error
	GetComputeNode(nodeID uuid.UUID) (nodes.ComputeNode, error)
	UpdateComputeNode(nodeID uuid.UUID, node nodes.ComputeNode) error
	DeleteComputeNode(nodeID uuid.UUID) error

	LookupComputeNodeByXName(xname string) (nodes.ComputeNode, error)
	LookupComputeNodeByMACAddress(mac string) (nodes.ComputeNode, error)
	SearchComputeNodes(xname, hostname, arch, bootMAC, bmcMAC string) ([]nodes.ComputeNode, error)

	SaveBMC(bmcID uuid.UUID, bmc nodes.BMC) error
	GetBMC(bmcID uuid.UUID) (nodes.BMC, error)
	UpdateBMC(bmcID uuid.UUID, bmc nodes.BMC) error
	DeleteBMC(bmcID uuid.UUID) error

	LookupBMCByXName(xname string) (nodes.BMC, error)
	LookupBMCByMACAddress(mac string) (nodes.BMC, error)
}
