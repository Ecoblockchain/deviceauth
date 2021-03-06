// Copyright 2016 Mender Software AS
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
package inventory

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/mendersoftware/deviceauth/utils"

	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/mendersoftware/go-lib-micro/requestid"
	"github.com/pkg/errors"
)

const (
	// devices endpoint
	InventoryDevicesUri = "/api/0.1.0/devices"
)

// ClientConfig conveys client configuration
type ClientConfig struct {
	// Inventory service address
	InventoryAddr string
}

// ClientRunner is an interface of inventory client
type ClientRunner interface {
	AddDevice(req AddReq, client requestid.ApiRequester) error
	log.ContextLogger
}

// Client is an opaque implementation of inventory client. Implements
// ClientRunner interface
type Client struct {
	log  *log.Logger
	conf ClientConfig
}

// AddReq contains request data of request to add a device.
type AddReq struct {
	// Device ID
	Id string `json:"id"`
}

func (ic *Client) AddDevice(areq AddReq, client requestid.ApiRequester) error {
	ic.log.Debugf("add device %s to inventory", areq.Id)

	ireq, err := json.Marshal(areq)
	if err != nil {
		return errors.Wrapf(err, "failed to prepare device admission request")
	}

	contentReader := bytes.NewReader(ireq)

	req, err := http.NewRequest(
		http.MethodPost,
		utils.JoinURL(ic.conf.InventoryAddr, InventoryDevicesUri),
		contentReader)
	if err != nil {
		return errors.Wrapf(err, "failed to create request")
	}

	req.Header.Set("Content-Type", "application/json")

	rsp, err := client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "failed to add device")
	}
	defer rsp.Body.Close()

	switch rsp.StatusCode {
	case http.StatusConflict:
		ic.log.Warnf("inventory entry for device %s already exists", areq.Id)
	case http.StatusCreated:
		ic.log.Infof("inventory entry for device %s created", areq.Id)
	default:
		ic.log.Errorf("failed to create inventory entry for device")
		if err == nil {
			err = errors.New("unexpected response status")
		}
		return errors.Wrapf(err,
			"device add request failed with status %v", rsp.Status)
	}
	return nil
}

func (ic *Client) UseLog(l *log.Logger) {
	ic.log = l.F(log.Ctx{})
}

func NewClientWithLogger(c ClientConfig, l *log.Logger) *Client {
	l = l.F(log.Ctx{})
	client := NewClient(c)
	client.UseLog(l)
	return client
}

func NewClient(c ClientConfig) *Client {
	return &Client{
		log:  log.New(log.Ctx{}),
		conf: c,
	}
}
