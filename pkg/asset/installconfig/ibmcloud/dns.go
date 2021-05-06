package ibmcloud

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/pkg/errors"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

// Zone represents a DNS Zone
type Zone struct {
	Name           string
	CISInstanceCRN string
}

// GetDNSZone returns a DNS Zone chosen by survey.
func GetDNSZone() (*Zone, error) {
	client, err := NewClient(context.TODO())
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	publicZones, err := client.GetDNSZones(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve base domains")
	}
	if len(publicZones) == 0 {
		return nil, errors.New("no domain names found in project")
	}

	var options []string
	var optionToZoneMap = make(map[string]*Zone, len(publicZones))
	for _, zone := range publicZones {
		option := fmt.Sprintf("%s (%s)", zone.Name, zone.CISInstanceName)
		optionToZoneMap[option] = &Zone{
			Name:           zone.Name,
			CISInstanceCRN: zone.CISInstanceCRN,
		}
		options = append(options, option)
	}

	var zoneChoice string
	if err := survey.AskOne(&survey.Select{
		Message: "Base Domain",
		Help:    "The base domain of the cluster. All DNS records will be sub-domains of this base and will also include the cluster name.\n\nIf you don't see you intended base-domain listed, create a new public hosted zone and rerun the installer.",
		Options: options,
	}, &zoneChoice, func(ans interface{}) error {
		choice := ans.(string)
		i := sort.SearchStrings(options, choice)
		if i == len(publicZones) || options[i] != choice {
			return errors.Errorf("invalid base domain %q", choice)
		}
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "failed UserInput")
	}

	return optionToZoneMap[zoneChoice], nil
}
