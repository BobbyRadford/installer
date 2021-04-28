package ibmcloud

// Platform stores all the global configuration that all machinesets use.
type Platform struct {
	// ResourceGroup is the name of an existing resource group where the cluster
	// and all required resources will be created.
	// +optional
	ResourceGroup string `json:"resourceGroup,omitempty"`

	// Region specifies the IBM Cloud region where the cluster will be
	// created.
	Region string `json:"region"`

	// CISInstanceCRN is the Cloud Internet Services CRN of the base domain DNS
	// zone.
	CISInstanceCRN string `json:"cisInstanceCRN"`

	// ClusterOSImage is the name of the custom RHCOS image.
	ClusterOSImage string `json:"clusterOSImage"`
}

// SetBaseDomain sets the CISInstanceCRN.
func (p *Platform) SetBaseDomain(cisInstanceCRN string) error {
	p.CISInstanceCRN = cisInstanceCRN
	return nil
}
