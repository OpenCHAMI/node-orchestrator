package csm

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/openchami/node-orchestrator/internal/api/smd"
	"github.com/openchami/node-orchestrator/internal/storage"
	"github.com/openchami/node-orchestrator/pkg/nodes"
)

type CSMStorage struct {
	BaseURI string
	JWT     string
	Client  *http.Client
}

func NewCSMStorage(baseURI, jwt string) *CSMStorage {
	return &CSMStorage{
		BaseURI: baseURI,
		JWT:     jwt,
		Client:  createHTTPClient(jwt),
	}
}

func createHTTPClient(jwt string) *http.Client {
	// create a transport with default settings
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	// create a new http client with the transport
	client := &http.Client{
		Transport: transport,
	}

	// set the default headers for authentication and encoding
	client.Transport = &authTransport{
		Transport: client.Transport,
		JWT:       jwt,
	}

	return client
}

type authTransport struct {
	Transport http.RoundTripper
	JWT       string
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// set the JWT authentication header
	req.Header.Set("Authorization", "Bearer "+t.JWT)

	// set the default encoding header
	req.Header.Set("Content-Type", "application/json")

	// set the default accept header
	req.Header.Set("Accept", "application/json")

	// set the default user-agent header
	req.Header.Set("User-Agent", "OpenCHAMI CSM Client")

	// perform the actual request using the underlying transport
	return t.Transport.RoundTrip(req)
}

func (s *CSMStorage) SaveComputeNode(nodeID uuid.UUID, node nodes.ComputeNode, nid int) error {
	// Call SMD to create the Components representing the Comptue Node and BMC
	csmNodeComponent := smd.Component{
		ID:    node.LocationString,
		Role:  "Compute",
		Arch:  "X86",
		State: "Ready",
		NID:   nid,
	}
	csmBMCComponent := smd.Component{
		ID:   node.BMC.LocationString,
		Role: "BMC",
	}
	csmNodeComponentJSON, _ := json.Marshal(csmNodeComponent)
	s.Client.Post(s.BaseURI+"v2/State/Components/", "application/json", bytes.NewBuffer(csmNodeComponentJSON))
	csmBMCComponentJSON, _ := json.Marshal(csmBMCComponent)
	s.Client.Post(s.BaseURI+"v2/State/Components/", "application/json", bytes.NewBuffer(csmBMCComponentJSON))

	// Call SMD to create the EthernetInterfaces representing the Compute Node's network interfaces
	for _, intf := range node.NetworkInterfaces {
		csmInterface := smd.CompEthInterface{
			MACAddr: intf.MACAddress,
			IPAddrs: []smd.IPAddressMapping{{IPAddr: intf.IPv4Address}},
			CompID:  node.LocationString,
		}
		csmInterfaceJSON, _ := json.Marshal(csmInterface)
		s.Client.Post(s.BaseURI+"v2/Inventory/EthernetInterfaces/", "application/json", bytes.NewBuffer(csmInterfaceJSON))
	}

	// Call BSS to set the boot parameters
	bootParams := smd.BootParams{
		Macs:   []string{node.BootMac},
		Kernel: node.BootData.KernelURL,
		Initrd: node.BootData.ImageURL,
		Params: node.BootData.KernelCommandLine,
	}
	bootParamsJSON, _ := json.Marshal(bootParams)
	s.Client.Post(s.BaseURI+"/bootparameters", "application/json", bytes.NewBuffer(bootParamsJSON))
	return nil
}

func (s *CSMStorage) GetComputeNode(nodeID uuid.UUID) (nodes.ComputeNode, error) {
	// TODO: Implement GetComputeNode method
	return nodes.ComputeNode{}, nil
}

func (s *CSMStorage) UpdateComputeNode(nodeID uuid.UUID, node nodes.ComputeNode) error {
	// TODO: Implement UpdateComputeNode method
	return nil
}

func (s *CSMStorage) DeleteComputeNode(nodeID uuid.UUID) error {
	// TODO: Implement DeleteComputeNode method
	return nil
}

func (s *CSMStorage) LookupComputeNodeByXName(xname string) (nodes.ComputeNode, error) {
	// TODO: Implement LookupComputeNodeByXName method
	return nodes.ComputeNode{}, nil
}

func (s *CSMStorage) LookupComputeNodeByMACAddress(mac string) (nodes.ComputeNode, error) {
	// TODO: Implement LookupComputeNodeByMACAddress method
	return nodes.ComputeNode{}, nil
}

func (s *CSMStorage) SearchComputeNodes(opts ...storage.NodeSearchOption) ([]nodes.ComputeNode, error) {
	// TODO: Implement SearchComputeNodes method
	return []nodes.ComputeNode{}, nil
}

func (s *CSMStorage) SaveBMC(bmcID uuid.UUID, bmc nodes.BMC) error {
	// TODO: Implement SaveBMC method
	return nil
}

func (s *CSMStorage) GetBMC(bmcID uuid.UUID) (nodes.BMC, error) {
	// TODO: Implement GetBMC method
	return nodes.BMC{}, nil
}

func (s *CSMStorage) UpdateBMC(bmcID uuid.UUID, bmc nodes.BMC) error {
	// TODO: Implement UpdateBMC method
	return nil
}

func (s *CSMStorage) DeleteBMC(bmcID uuid.UUID) error {
	// TODO: Implement DeleteBMC method
	return nil
}

func (s *CSMStorage) LookupBMCByXName(xname string) (nodes.BMC, error) {
	// TODO: Implement LookupBMCByXName method
	return nodes.BMC{}, nil
}

func (s *CSMStorage) LookupBMCByMACAddress(mac string) (nodes.BMC, error) {
	// TODO: Implement LookupBMCByMACAddress method
	return nodes.BMC{}, nil
}
