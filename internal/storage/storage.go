package storage

import (
	"github.com/google/uuid"
	"github.com/openchami/node-orchestrator/pkg/nodes"
)

type NodeStorage interface {
	SaveComputeNode(nodeID uuid.UUID, node nodes.ComputeNode) error
	GetComputeNode(nodeID uuid.UUID) (nodes.ComputeNode, error)
	UpdateComputeNode(nodeID uuid.UUID, node nodes.ComputeNode) error
	DeleteComputeNode(nodeID uuid.UUID) error

	LookupComputeNodeByXName(xname string) (nodes.ComputeNode, error)
	LookupComputeNodeByMACAddress(mac string) (nodes.ComputeNode, error)
	SearchComputeNodes(opts ...NodeSearchOption) ([]nodes.ComputeNode, error)

	SaveBMC(bmcID uuid.UUID, bmc nodes.BMC) error
	GetBMC(bmcID uuid.UUID) (nodes.BMC, error)
	UpdateBMC(bmcID uuid.UUID, bmc nodes.BMC) error
	DeleteBMC(bmcID uuid.UUID) error

	LookupBMCByXName(xname string) (nodes.BMC, error)
	LookupBMCByMACAddress(mac string) (nodes.BMC, error)
}

type NodeSearchOptions struct {
	XName           string
	Hostname        string
	Arch            string
	BootMAC         string
	BMCMAC          string
	MissingXName    bool
	MissingHostname bool
	MissingArch     bool
	MissingBootMAC  bool
	MissingBMCMAC   bool
	MissingIPV4     bool
	MissingIPV6     bool
}

type NodeSearchOption func(*NodeSearchOptions)

func WithXName(xname string) NodeSearchOption {
	return func(opts *NodeSearchOptions) {
		opts.XName = xname
	}
}

func WithHostname(hostname string) NodeSearchOption {
	return func(opts *NodeSearchOptions) {
		opts.Hostname = hostname
	}
}

func WithArch(arch string) NodeSearchOption {
	return func(opts *NodeSearchOptions) {
		opts.Arch = arch
	}
}

func WithBootMAC(bootMAC string) NodeSearchOption {
	return func(opts *NodeSearchOptions) {
		opts.BootMAC = bootMAC
	}
}

func WithBMCMAC(bmcMAC string) NodeSearchOption {
	return func(opts *NodeSearchOptions) {
		opts.BMCMAC = bmcMAC
	}
}

func WithMissingXName() NodeSearchOption {
	return func(opts *NodeSearchOptions) {
		opts.MissingXName = true
	}
}

func WithMissingHostname() NodeSearchOption {
	return func(opts *NodeSearchOptions) {
		opts.MissingHostname = true
	}
}

func WithMissingArch() NodeSearchOption {
	return func(opts *NodeSearchOptions) {
		opts.MissingArch = true
	}
}

func WithMissingBootMAC() NodeSearchOption {
	return func(opts *NodeSearchOptions) {
		opts.MissingBootMAC = true
	}
}

func WithMissingBMCMAC() NodeSearchOption {
	return func(opts *NodeSearchOptions) {
		opts.MissingBMCMAC = true
	}
}

func WithMissingIPV4() NodeSearchOption {
	return func(opts *NodeSearchOptions) {
		opts.MissingIPV4 = true
	}
}

func WithMissingIPV6() NodeSearchOption {
	return func(opts *NodeSearchOptions) {
		opts.MissingIPV6 = true
	}
}
