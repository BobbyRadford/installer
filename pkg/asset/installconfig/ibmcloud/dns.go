package ibmcloud

import (
	"context"
	"sort"
	"time"

	"github.com/pkg/errors"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

// GetBaseDomain returns a base domain chosen from among the project's public DNS zones.
func GetBaseDomain(project string) (string, error) {
	client, err := NewClient(context.TODO())
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	publicZones, err := client.GetPublicDomains(ctx, project)
	if err != nil {
		return "", errors.Wrap(err, "could not retrieve base domains")
	}
	if len(publicZones) == 0 {
		return "", errors.New("no domain names found in project")
	}
	sort.Strings(publicZones)

	var domain string
	if err := survey.AskOne(&survey.Select{
		Message: "Base Domain",
		Help:    "The base domain of the cluster. All DNS records will be sub-domains of this base and will also include the cluster name.\n\nIf you don't see you intended base-domain listed, create a new public hosted zone and rerun the installer.",
		Options: publicZones,
	}, &domain, func(ans interface{}) error {
		choice := ans.(string)
		i := sort.SearchStrings(publicZones, choice)
		if i == len(publicZones) || publicZones[i] != choice {
			return errors.Errorf("invalid base domain %q", choice)
		}
		return nil
	}); err != nil {
		return "", errors.Wrap(err, "failed UserInput")
	}

	return domain, nil
}
