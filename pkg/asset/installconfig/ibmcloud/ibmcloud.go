package ibmcloud

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/IBM-Cloud/bluemix-go/models"
	"github.com/openshift/installer/pkg/types/ibmcloud"
	"github.com/openshift/installer/pkg/types/ibmcloud/validation"
	"github.com/pkg/errors"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

// Platform collects IBM Cloud-specific configuration.
func Platform() (*ibmcloud.Platform, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	client, err := NewClient(ctx)
	if err != nil {
		return nil, err
	}

	project, err := selectResourceGroup(ctx, client)
	if err != nil {
		return nil, err
	}

	region, err := selectRegion(client)
	if err != nil {
		return nil, err
	}

	return &ibmcloud.Platform{
		ResourceGroupID: project,
		Region:          region,
	}, nil
}

func selectResourceGroup(ctx context.Context, client *Client) (string, error) {
	groups, err := client.GetResourceGroups(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to list resource groups")
	}

	var defaultResourceGroup *models.ResourceGroup
	for i := range groups {
		if groups[i].Name == "Default" {
			defaultResourceGroup = &groups[i]
		}
	}
	if defaultResourceGroup == nil {
		defaultResourceGroup = &groups[0]
	}

	var options []string
	ids := make(map[string]string)

	var defaultValue string

	for _, group := range groups {
		option := fmt.Sprintf("%s (%s)", group.Name, group.ID)
		ids[option] = group.ID
		if group.ID == defaultResourceGroup.ID {
			defaultValue = option
		}
		options = append(options, option)
	}
	sort.Strings(options)

	var selectedResourceGroup string
	err = survey.Ask([]*survey.Question{
		{
			Prompt: &survey.Select{
				Message: "Resource Group ID",
				Help:    "The resource group id where the cluster will be provisioned. The 'Default' resource group is used if not specified.",
				Default: defaultValue,
				Options: options,
			},
		},
	}, &selectedResourceGroup)
	return ids[selectedResourceGroup], err
}

func selectRegion(client *Client) (string, error) {
	longRegions := make([]string, 0, len(validation.Regions))
	shortRegions := make([]string, 0, len(validation.Regions))
	for id, location := range validation.Regions {
		longRegions = append(longRegions, fmt.Sprintf("%s (%s)", id, location))
		shortRegions = append(shortRegions, id)
	}
	regionTransform := survey.TransformString(func(s string) string {
		return strings.SplitN(s, " ", 2)[0]
	})

	sort.Strings(longRegions)
	sort.Strings(shortRegions)

	defaultRegion := client.ssn.Config.Region

	var selectedRegion string
	err := survey.Ask([]*survey.Question{
		{
			Prompt: &survey.Select{
				Message: "Region",
				Help:    "The IBM Cloud region to be used for installation.",
				Default: fmt.Sprintf("%s (%s)", defaultRegion, validation.Regions[defaultRegion]),
				Options: longRegions,
			},
			Validate: survey.ComposeValidators(survey.Required, func(ans interface{}) error {
				choice := regionTransform(ans).(string)
				i := sort.SearchStrings(shortRegions, choice)
				if i == len(shortRegions) || shortRegions[i] != choice {
					return errors.Errorf("invalid region %q", choice)
				}
				return nil
			}),
			Transform: regionTransform,
		},
	}, &selectedRegion)
	if err != nil {
		return "", err
	}
	return selectedRegion, nil
}
