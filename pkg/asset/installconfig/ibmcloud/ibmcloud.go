package ibmcloud

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"gopkg.in/AlecAivazis/survey.v1"

	"github.com/IBM-Cloud/bluemix-go/models"
	"github.com/openshift/installer/pkg/types/ibmcloud"
	"github.com/openshift/installer/pkg/types/ibmcloud/validation"
	"github.com/pkg/errors"
)

// Platform collects IBM Cloud-specific configuration.
func Platform() (*ibmcloud.Platform, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	client, err := NewClient(ctx)

	project, err := selectProject(ctx, client)
	if err != nil {
		return nil, err
	}

	region, err := selectRegion(client)
	if err != nil {
		return nil, err
	}

	return &ibmcloud.Platform{
		ProjectID: project,
		Region:    region,
	}, nil
}

func selectProject(ctx context.Context, client *Client) (string, error) {
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

	// TODO: Remove the os.LookupEnv() as this is only needed for development
	var selectedResourceGroup string
	var ok bool
	if selectedResourceGroup, ok = os.LookupEnv("IBM_RESOURCE_GROUP_ID"); !ok {
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
	}
	selectedResourceGroup = ids[selectedResourceGroup]
	return selectedResourceGroup, err
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

	// TODO: Remove the os.LookupEnv() as this is only needed for development
	var selectedRegion string
	var ok bool
	if selectedRegion, ok = os.LookupEnv("IBM_REGION"); !ok {
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
	}
	return selectedRegion, nil
}
