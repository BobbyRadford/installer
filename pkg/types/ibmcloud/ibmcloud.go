package ibmcloud

// DNSZoneResponse represents a DNS zone response.
type DNSZoneResponse struct {
	// Name is the domain name of the zone.
	Name string

	// CISInstanceCRN is the IBM Cloud Resource Name for the CIS instance where
	// the DNS zone is managed.
	CISInstanceCRN string

	// CISInstanceName is the display name of the CIS instance where the DNS zone
	// is managed.
	CISInstanceName string
}

// EncryptionKeyResponse ...
type EncryptionKeyResponse struct{}
