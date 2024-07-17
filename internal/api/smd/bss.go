package smd

// Copied from https://github.com/Cray-HPE/hms-bss/blob/master/pkg/bssTypes/types.go

// This is the main data structure used to communicate with the client.  It
// allows the client to set parameters along the with kernel and initrd
// references.  It is also used to return boot info to the user.  The expected
// usage is that one of arrays hosts, macs, or nids is used to  specify the
// hosts for booting.  We could also allow special names for hosts such as
// "compute" or "service" meaning all nodes of those categories, or we
// could introduce an additional property for this type of selection.  We also
// provide a "default" selection which provides a way to supply default
// parameters for any node which is not explicitly configured.
type BootParams struct {
	Hosts     []string  `json:"hosts,omitempty"`
	Macs      []string  `json:"macs,omitempty"`
	Nids      []int32   `json:"nids,omitempty"`
	Params    string    `json:"params,omitempty"`
	Kernel    string    `json:"kernel,omitempty"`
	Initrd    string    `json:"initrd,omitempty"`
	CloudInit CloudInit `json:"cloud-init,omitempty"`
}

// The main cloud-init struct. Leave the meta-data, user-data, and phone home
// info as generic interfaces as the user defines how much info exists in it
type CloudDataType map[string]interface{}
type CloudInit struct {
	MetaData  CloudDataType `json:"meta-data"`
	UserData  CloudDataType `json:"user-data"`
	PhoneHome PhoneHome     `json:"phone-home,omitempty"`
}

type PhoneHome struct {
	PublicKeyDSA     string `form:"pub_key_dsa" json:"pub_key_dsa" binding:"omitempty"`
	PublicKeyRSA     string `form:"pub_key_rsa" json:"pub_key_rsa" binding:"omitempty"`
	PublicKeyECDSA   string `form:"pub_key_ecdsa" json:"pub_key_ecdsa" binding:"omitempty"`
	PublicKeyED25519 string `form:"pub_key_ed25519" json:"pub_key_ed25519,omitempty"`
	InstanceID       string `form:"instance_id" json:"instance_id" binding:"omitempty"`
	Hostname         string `form:"hostname" json:"hostname" binding:"omitempty"`
	FQDN             string `form:"fqdn" json:"fqdn" binding:"omitempty"`
}
