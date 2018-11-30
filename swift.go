package main

import (
	"fmt"
	"io"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/containers"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/objects"
	"github.com/gophercloud/gophercloud/pagination"
	log "github.com/sirupsen/logrus"
)

type Swift struct {
	config Config
}

func NewSwift(c Config) *Swift {
	return &Swift{
		config: c,
	}
}

func (s *Swift) Init() error {
	// Make sure whether the container exists
	ojs, err := s.getObjectStorageClient()
	if err != nil {
		return err
	}

	exists := false
	containers.List(ojs, containers.ListOpts{}).EachPage(func(p pagination.Page) (bool, error) {
		names, err := containers.ExtractNames(p)
		if err != nil {
			return false, err
		}

		for _, name := range names {
			log.Debugf("Container found [name=%s]", name)
			if name == s.config.Container {
				exists = true
				return false, nil
			}
		}

		return true, nil
	})

	if !exists {
		if s.config.CreateContainerIfNotExists {
			if err = s.CreateContainer(); err != nil {
				return fmt.Errorf("Couldn't create container. [%s]", err)
			}
		} else {
			return fmt.Errorf("Container '%s' does not exist.", s.config.Container)
		}
	}

	return nil
}

func (s *Swift) CreateContainer() (err error) {
	client, err := s.getObjectStorageClient()
	if err != nil {
		return err
	}

	rs := containers.Create(client, s.config.Container, containers.CreateOpts{})
	return rs.Err
}

func (s *Swift) DeleteContainer() (err error) {
	client, err := s.getObjectStorageClient()
	if err != nil {
		return err
	}

	ls, err := s.List()
	if err != nil {
		return err
	}

	// Recursive deletion for all objects in the container
	for _, obj := range ls {
		drs := objects.Delete(client, s.config.Container, obj.Name, objects.DeleteOpts{})
		if drs.Err != nil {
			return drs.Err
		}
	}

	rs := containers.Delete(client, s.config.Container)
	return rs.Err
}

func (s *Swift) List() (ls []objects.Object, err error) {
	client, err := s.getObjectStorageClient()
	if err != nil {
		return nil, err
	}

	ls = make([]objects.Object, 0, 10)
	err = objects.List(client, s.config.Container, objects.ListOpts{
		Full: true,
	}).EachPage(func(p pagination.Page) (bool, error) {
		ls, err = objects.ExtractInfo(p)
		if err != nil {
			return false, err
		}
		return true, nil
	})

	return ls, err
}

func (s *Swift) Get(name string) (header *objects.GetHeader, err error) {
	client, err := s.getObjectStorageClient()
	if err != nil {
		return nil, err
	}

	return objects.Get(client, s.config.Container, name, objects.GetOpts{}).Extract()
}

func (s *Swift) Download(name string) (content io.ReadCloser, size int64, err error) {
	client, err := s.getObjectStorageClient()
	if err != nil {
		return nil, 0, err
	}

	rs := objects.Download(client, s.config.Container, name, objects.DownloadOpts{})
	if rs.Err != nil {
		return nil, 0, rs.Err
	}

	info, err := rs.Extract()
	if err != nil {
		return nil, 0, err
	}

	return rs.Body, info.ContentLength, nil
}

func (s *Swift) Put(name string, content io.Reader) error {
	client, err := s.getObjectStorageClient()
	if err != nil {
		return err
	}

	// temporary object name
	tmpname := "tmp_" + name

	// delete a temporary file from container
	defer func() {
		objects.Delete(client, s.config.Container, tmpname, objects.DeleteOpts{})
	}()

	cOpts := objects.CreateOpts{
		Content: content,
	}
	rCreate := objects.Create(client, s.config.Container, tmpname, cOpts)
	if rCreate.Err != nil {
		return rCreate.Err
	}

	dest := fmt.Sprintf("%s%s%s", s.config.Container, Delimiter, name)
	rCopy := objects.Copy(client, s.config.Container, tmpname, objects.CopyOpts{
		Destination: dest,
	})
	if rCopy.Err != nil {
		return rCopy.Err
	}

	return nil
}

func (s *Swift) Delete(name string) (err error) {
	client, err := s.getObjectStorageClient()
	if err != nil {
		return err
	}

	return objects.Delete(client, s.config.Container, name, objects.DeleteOpts{}).Err
}

func (s *Swift) Rename(oldName, newName string) (err error) {
	client, err := s.getObjectStorageClient()
	if err != nil {
		return err
	}

	dest := fmt.Sprintf("%s%s%s", s.config.Container, Delimiter, newName)
	rCopy := objects.Copy(client, s.config.Container, oldName, objects.CopyOpts{
		Destination: dest,
	})
	if rCopy.Err != nil {
		return rCopy.Err
	}

	return s.Delete(oldName)
}

func (s *Swift) getObjectStorageClient() (*gophercloud.ServiceClient, error) {
	auth, err := s.getAuthClient()
	if err != nil {
		return nil, err
	}

	opts := gophercloud.EndpointOpts{}
	if s.config.OsRegion != "" {
		opts.Region = s.config.OsRegion
	}

	return openstack.NewObjectStorageV1(auth, opts)
}

func (s *Swift) getAuthClient() (*gophercloud.ProviderClient, error) {
	var (
		err  error
		opts gophercloud.AuthOptions
	)

	if (s.config.OsUserID != "" || s.config.OsUsername != "") && s.config.OsPassword != "" {
		opts = gophercloud.AuthOptions{
			IdentityEndpoint: s.config.OsIdentityEndpoint,
			UserID:           s.config.OsUserID,
			Username:         s.config.OsUsername,
			Password:         s.config.OsPassword,
			DomainID:         s.config.OsDomainID,
			DomainName:       s.config.OsDomainName,
			TenantID:         s.config.OsTenantID,
			TenantName:       s.config.OsTenantName,
		}

	} else if opts, err = openstack.AuthOptionsFromEnv(); err != nil {
		return nil, err
	}

	return openstack.AuthenticatedClient(opts)
}
