package ibmcloud

import (
	"fmt"
	"testing"

	"github.com/IBM-Cloud/bluemix-go/models"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/golang/mock/gomock"
	"github.com/openshift/installer/pkg/asset/installconfig/ibmcloud/mock"
	"github.com/openshift/installer/pkg/ipnet"
	"github.com/openshift/installer/pkg/types"
	"github.com/openshift/installer/pkg/types/ibmcloud"
	"github.com/stretchr/testify/assert"
)

type editFunctions []func(ic *types.InstallConfig)

var (
	validRegion                  = "us-south"
	validCIDR                    = "10.0.0.0/16"
	validClusterOSImage          = "valid-rhcos-image"
	validCISCRN                  = "crn:v1:bluemix:public:internet-svcs:us-south:a/account:instance::"
	validDNSZoneID               = "valid-zone-id"
	validBaseDomain              = "valid.base.domain"
	validVPC                     = "valid-vpc"
	validVPCResourceGroup        = "valid-vpc-resource-group"
	validVPCResourceGroupID      = "valid-vpc-resource-group-id"
	validPublicSubnetUSSouth1ID  = "public-subnet-us-south-1-id"
	validPublicSubnetUSSouth2ID  = "public-subnet-us-south-2-id"
	validPrivateSubnetUSSouth1ID = "private-subnet-us-south-1-id"
	validPrivateSubnetUSSouth2ID = "private-subnet-us-south-2-id"
	validSubnets                 = []string{
		validPublicSubnetUSSouth1ID,
		validPublicSubnetUSSouth2ID,
		validPrivateSubnetUSSouth1ID,
		validPrivateSubnetUSSouth2ID,
	}
	validZoneUSSouth1 = "us-south-1"

	notFoundCISInstanceCRN = func(ic *types.InstallConfig) { ic.IBMCloud.CISInstanceCRN = "not:found" }
	notFoundBaseDomain     = func(ic *types.InstallConfig) { ic.BaseDomain = "notfound.base.domain" }
	notFoundClusterOSImage = func(ic *types.InstallConfig) { ic.IBMCloud.ClusterOSImage = "not-found" }
	validVPCConfig         = func(ic *types.InstallConfig) {
		ic.IBMCloud.VPC = validVPC
		ic.IBMCloud.VPCResourceGroup = validVPCResourceGroup
		ic.IBMCloud.Subnets = validSubnets
	}
	notFoundVPC                   = func(ic *types.InstallConfig) { ic.IBMCloud.VPC = "not-found" }
	internalErrorVPC              = func(ic *types.InstallConfig) { ic.IBMCloud.VPC = "internal-error-vpc" }
	notFoundVPCResourceGroup      = func(ic *types.InstallConfig) { ic.IBMCloud.VPCResourceGroup = "not-found" }
	internalErrorVPCResourceGroup = func(ic *types.InstallConfig) { ic.IBMCloud.VPCResourceGroup = "internal-error-resource-group" }
	subnetInvalidZone             = func(ic *types.InstallConfig) { ic.IBMCloud.Subnets = []string{"subnet-invalid-zone"} }
)

func validInstallConfig() *types.InstallConfig {
	return &types.InstallConfig{
		BaseDomain: validBaseDomain,
		Networking: &types.Networking{
			MachineNetwork: []types.MachineNetworkEntry{
				{CIDR: *ipnet.MustParseCIDR(validCIDR)},
			},
		},
		Publish: types.ExternalPublishingStrategy,
		Platform: types.Platform{
			IBMCloud: validMinimalPlatform(),
		},
		ControlPlane: &types.MachinePool{
			Platform: types.MachinePoolPlatform{
				IBMCloud: validMachinePool(),
			},
		},
		Compute: []types.MachinePool{{
			Platform: types.MachinePoolPlatform{
				IBMCloud: validMachinePool(),
			},
		}},
	}
}

func validMinimalPlatform() *ibmcloud.Platform {
	return &ibmcloud.Platform{
		Region:         validRegion,
		CISInstanceCRN: validCISCRN,
		ClusterOSImage: validClusterOSImage,
	}
}

func validMachinePool() *ibmcloud.MachinePool {
	return &ibmcloud.MachinePool{}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name     string
		edits    editFunctions
		errorMsg string
	}{
		{
			name:     "Valid install config",
			edits:    editFunctions{},
			errorMsg: "",
		},
		{
			name:     "not found cisInstanceCRN",
			edits:    editFunctions{notFoundCISInstanceCRN},
			errorMsg: `^platorm\.ibmcloud\.cisInstanceCRN: Not found: "not:found"$`,
		},
		{
			name:     "not found baseDomain",
			edits:    editFunctions{notFoundBaseDomain},
			errorMsg: fmt.Sprintf("^platorm.ibmcloud.cisInstanceCRN: Invalid value: \"%s\": the cis instance does not have an active DNS zone for the base domain: %s$", validCISCRN, "notfound.base.domain"),
		},
		{
			name:     "not found clusterOSImage",
			edits:    editFunctions{notFoundClusterOSImage},
			errorMsg: `^platorm\.ibmcloud\.clusterOSImage: Not found: "not-found"$`,
		},
		{
			name:     "valid vpc config",
			edits:    editFunctions{validVPCConfig},
			errorMsg: "",
		},
		{
			name:     "not found vpcResourceGroup",
			edits:    editFunctions{validVPCConfig, notFoundVPCResourceGroup},
			errorMsg: `^platorm\.ibmcloud\.vpcResourceGroup: Not found: "not-found"$`,
		},
		{
			name:     "internal error vpcResourceGroup",
			edits:    editFunctions{validVPCConfig, internalErrorVPCResourceGroup},
			errorMsg: `^platorm\.ibmcloud\.vpcResourceGroup: Internal error$`,
		},
		{
			name:     "not found vpcResourceGroup",
			edits:    editFunctions{validVPCConfig, notFoundVPC},
			errorMsg: `^platorm\.ibmcloud\.vpc: Not found: \"not-found\"$`,
		},
		{
			name:     "internal error vpcResourceGroup",
			edits:    editFunctions{validVPCConfig, internalErrorVPC},
			errorMsg: `^platorm\.ibmcloud\.vpc: Internal error$`,
		},
		{
			name:     "subnet invalid zone",
			edits:    editFunctions{validVPCConfig, subnetInvalidZone},
			errorMsg: `^\Qplatorm.ibmcloud.subnets: Invalid value: "subnet-invalid-zone": subnet is not in expected zones: [us-south-1 us-south-2 us-south-3]\E$`,
		},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ibmcloudClient := mock.NewMockAPI(mockCtrl)

	ibmcloudClient.EXPECT().GetCISInstance(gomock.Any(), validCISCRN).Return(nil, nil).AnyTimes()
	ibmcloudClient.EXPECT().GetCISInstance(gomock.Any(), gomock.Not(validCISCRN)).Return(nil, fmt.Errorf("failed to get cis instances")).AnyTimes()

	ibmcloudClient.EXPECT().GetZoneIDByName(gomock.Any(), validCISCRN, validBaseDomain).Return(validDNSZoneID, nil).AnyTimes()
	ibmcloudClient.EXPECT().GetZoneIDByName(gomock.Any(), validCISCRN, gomock.Not(validBaseDomain)).Return("", fmt.Errorf("")).AnyTimes()

	ibmcloudClient.EXPECT().GetCustomImageByName(gomock.Any(), validClusterOSImage).Return(&vpcv1.Image{}, nil).AnyTimes()
	ibmcloudClient.EXPECT().GetCustomImageByName(gomock.Any(), gomock.Not(validClusterOSImage)).Return(nil, fmt.Errorf("")).AnyTimes()

	ibmcloudClient.EXPECT().GetResourceGroup(gomock.Any(), validVPCResourceGroup).Return(&models.ResourceGroup{ID: validVPCResourceGroupID}, nil).AnyTimes()
	ibmcloudClient.EXPECT().GetResourceGroup(gomock.Any(), "not-found").Return(nil, fmt.Errorf("Given resource Group : \"not-found\" doesn't exist")).AnyTimes()
	ibmcloudClient.EXPECT().GetResourceGroup(gomock.Any(), "internal-error-resource-group").Return(nil, fmt.Errorf("")).AnyTimes()

	ibmcloudClient.EXPECT().GetVPCByName(gomock.Any(), validVPC, validVPCResourceGroupID).Return(&vpcv1.VPC{}, nil).AnyTimes()
	ibmcloudClient.EXPECT().GetVPCByName(gomock.Any(), "not-found", validVPCResourceGroupID).Return(nil, fmt.Errorf("vpc not found: \"not-found\""))
	ibmcloudClient.EXPECT().GetVPCByName(gomock.Any(), "internal-error-vpc", validVPCResourceGroupID).Return(nil, fmt.Errorf(""))

	ibmcloudClient.EXPECT().GetSubnet(gomock.Any(), validPublicSubnetUSSouth1ID).Return(&vpcv1.Subnet{Zone: &vpcv1.ZoneReference{Name: &validZoneUSSouth1}}, nil).AnyTimes()
	ibmcloudClient.EXPECT().GetSubnet(gomock.Any(), validPublicSubnetUSSouth2ID).Return(&vpcv1.Subnet{Zone: &vpcv1.ZoneReference{Name: &validZoneUSSouth1}}, nil).AnyTimes()
	ibmcloudClient.EXPECT().GetSubnet(gomock.Any(), validPrivateSubnetUSSouth1ID).Return(&vpcv1.Subnet{Zone: &vpcv1.ZoneReference{Name: &validZoneUSSouth1}}, nil).AnyTimes()
	ibmcloudClient.EXPECT().GetSubnet(gomock.Any(), validPrivateSubnetUSSouth2ID).Return(&vpcv1.Subnet{Zone: &vpcv1.ZoneReference{Name: &validZoneUSSouth1}}, nil).AnyTimes()
	ibmcloudClient.EXPECT().GetSubnet(gomock.Any(), "subnet-invalid-zone").Return(&vpcv1.Subnet{Zone: &vpcv1.ZoneReference{Name: &[]string{"invalid"}[0]}}, nil).AnyTimes()

	ibmcloudClient.EXPECT().GetVPCZonesForRegion(gomock.Any(), validRegion).Return([]string{"us-south-1", "us-south-2", "us-south-3"}, nil).AnyTimes()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			editedInstallConfig := validInstallConfig()
			for _, edit := range tc.edits {
				edit(editedInstallConfig)
			}

			aggregatedErrors := Validate(ibmcloudClient, editedInstallConfig)
			if tc.errorMsg != "" {
				assert.Regexp(t, tc.errorMsg, aggregatedErrors)
			} else {
				assert.NoError(t, aggregatedErrors)
			}
		})
	}
}
