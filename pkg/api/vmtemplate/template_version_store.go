package vmtemplate

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/rancher/apiserver/pkg/apierror"
	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/wrangler/pkg/schemas/validation"

	ctlharvesterv1 "github.com/harvester/harvester/pkg/generated/controllers/harvesterhci.io/v1beta1"
	"github.com/harvester/harvester/pkg/ref"
)

type templateVersionStore struct {
	types.Store

	templateCache        ctlharvesterv1.VirtualMachineTemplateCache
	templateVersionCache ctlharvesterv1.VirtualMachineTemplateVersionCache
	keyPairCache         ctlharvesterv1.KeyPairCache
}

func (s *templateVersionStore) Create(request *types.APIRequest, schema *types.APISchema, data types.APIObject) (types.APIObject, error) {
	newData := data.Data()
	ns := newData.String("metadata", "namespace")
	templateID := newData.String("spec", "templateId")
	if templateID == "" {
		return types.APIObject{}, apierror.NewAPIError(validation.InvalidBodyContent, "TemplateId is empty")
	}

	templateNs, templateName := ref.Parse(templateID)
	if ns != templateNs {
		return types.APIObject{}, apierror.NewAPIError(validation.InvalidBodyContent, "Template version and template should belong to same namespace")
	}

	if _, err := s.templateCache.Get(templateNs, templateName); err != nil {
		return types.APIObject{}, apierror.NewAPIError(validation.InvalidBodyContent, err.Error())
	}

	keyPairIDs := newData.StringSlice("spec", "keyPairIds")
	if len(keyPairIDs) != 0 {
		for _, v := range keyPairIDs {
			keyPairNs, keyPairName := ref.Parse(v)
			_, err := s.keyPairCache.Get(keyPairNs, keyPairName)
			if err != nil {
				return types.APIObject{}, apierror.NewAPIError(validation.InvalidBodyContent, fmt.Sprintf("KeyPairID %s is invalid, %v", v, err))
			}
		}
	}

	newData.SetNested(templateName+"-", "metadata", "generateName")
	data.Object = newData
	return s.Store.Create(request, request.Schema, data)
}

func (s *templateVersionStore) Update(request *types.APIRequest, schema *types.APISchema, data types.APIObject, id string) (types.APIObject, error) {
	return types.APIObject{}, apierror.NewAPIError(validation.ActionNotAvailable, "Update templateVersion is not supported")
}

func (s *templateVersionStore) Delete(request *types.APIRequest, schema *types.APISchema, id string) (types.APIObject, error) {
	if err := s.canDeleteTemplateVersion(request.Namespace, request.Name); err != nil {
		return types.APIObject{}, apierror.NewAPIError(validation.ServerError, err.Error())
	}

	return s.Store.Delete(request, request.Schema, id)
}

func (s *templateVersionStore) canDeleteTemplateVersion(namespace, name string) error {
	vr, err := s.templateVersionCache.Get(namespace, name)
	if err != nil {
		return err
	}

	vtNS, vtname := ref.Parse(vr.Spec.TemplateID)
	vt, err := s.templateCache.Get(vtNS, vtname)
	if err != nil {
		return err
	}

	versionID := ref.Construct(namespace, name)
	if vt.Spec.DefaultVersionID == versionID {
		return errors.New("Cannot delete the default templateVersion")
	}

	return nil
}
