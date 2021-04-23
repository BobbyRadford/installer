package ibmcloud

// Platform stores all the global configuration that all machinesets
// use.
type Platform struct {
	// ResourceGroupID is the the resource group that will be used for the cluster.
	ResourceGroupID string `json:"resourceGroupID"`

	// Region specifies the IBM Cloud region where the cluster will be created.
	Region string `json:"region"`

	// CISInstanceCRN is the Cloud Internet Services CRN of the base domain DNS zone.
	CISInstanceCRN string `json:"cisInstanceCRN"`
}

// SetBaseDomain sets the CISInstanceCRN.
func (p *Platform) SetBaseDomain(cisInstanceCRN string) error {
	p.CISInstanceCRN = cisInstanceCRN
	return nil
}
